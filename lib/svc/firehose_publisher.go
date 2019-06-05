package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/cenkalti/backoff"
)

const (
	FirehoseMaxBatchSize = 500
)

type FirehosePublisher struct {
	Timeout            time.Duration
	BatchSize          int
	Ticker             *time.Ticker
	fh                 firehoseiface.FirehoseAPI
	deliveryStreamName string
	wg                 *sync.WaitGroup
	values             chan interface{}
	errors             chan error
	done               chan struct{}
}

func NewFirehosePublisher(deliveryStreamName string, fh firehoseiface.FirehoseAPI) *FirehosePublisher {
	return &FirehosePublisher{
		BatchSize:          FirehoseMaxBatchSize,
		Ticker:             time.NewTicker(48 * time.Hour),
		fh:                 fh,
		deliveryStreamName: deliveryStreamName,
		wg:                 new(sync.WaitGroup),
		values:             make(chan interface{}),
		errors:             make(chan error),
		done:               make(chan struct{}),
	}
}

func (publisher *FirehosePublisher) Go(ctx context.Context) {
	records := make(chan []byte)
	batches := make(chan []*firehose.Record)

	publisher.wg.Add(1)
	go publisher.serialize(ctx, publisher.values, records, publisher.errors)

	publisher.wg.Add(1)
	go publisher.publish(ctx, records, batches)

	publisher.wg.Add(1)
	go publisher.flush(ctx, batches, publisher.errors)

	publisher.wg.Wait()
	publisher.done <- struct{}{}
}

func (publisher *FirehosePublisher) Publish(v interface{}) {
	publisher.values <- v
}

func (publisher *FirehosePublisher) Errors() <-chan error {
	return publisher.errors
}

func (publisher *FirehosePublisher) Close(ctx context.Context) {
	close(publisher.values)
	defer close(publisher.errors)

	select {
	case <-ctx.Done():
	case <-publisher.done:
	}
}

func (publisher *FirehosePublisher) serialize(ctx context.Context, values <-chan interface{}, records chan<- []byte, errors chan<- error) {
	defer publisher.wg.Done()
	defer close(records)

	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	for value := range values {
		if err := encoder.Encode(value); err != nil {
			errors <- err
		} else {
			records <- buf.Bytes()
		}
		buf.Reset()
	}
}

func (publisher *FirehosePublisher) publish(ctx context.Context, records <-chan []byte, batches chan<- []*firehose.Record) {
	defer publisher.wg.Done()

	batch := make([]*firehose.Record, 0, FirehoseMaxBatchSize)

	defer func() {
		defer close(batches)

		if len(batch) > 0 {
			batches <- batch
		}
	}()

	for {
		select {
		case <-publisher.Ticker.C:
			batches <- batch
			batch = make([]*firehose.Record, 0, publisher.BatchSize)
		case bs, more := <-records:
			if !more {
				return
			}
			batch = append(batch, &firehose.Record{Data: bs})
			if len(batch) == publisher.BatchSize {
				batches <- batch
				batch = make([]*firehose.Record, 0, publisher.BatchSize)
			}
		}
	}
}

func (publisher *FirehosePublisher) flush(ctx context.Context, batches <-chan []*firehose.Record, errors chan<- error) {
	defer publisher.wg.Done()

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(publisher.deliveryStreamName)

	var batch []*firehose.Record

	publishBatch := func() error {
		resp, err := publisher.fh.PutRecordBatchWithContext(ctx, input)
		if err != nil {
			return err
		}

		failed := aws.Int64Value(resp.FailedPutCount)
		if failed == 0 {
			return nil
		}

		newBatch := make([]*firehose.Record, 0, failed)
		for i, record := range resp.RequestResponses {
			if aws.StringValue(record.ErrorCode) != "" || aws.StringValue(record.ErrorMessage) != "" {
				newBatch = append(newBatch, batch[i])
			}
		}
		batch = newBatch
		return fmt.Errorf("retrying %d records", len(batch))
	}

	notifyBackoff := func(err error, bo time.Duration) {
		log.Printf("%s.  Backing off %s", err, bo)
	}

	for batch = range batches {
		input.SetRecords(batch)

		eb := backoff.NewExponentialBackOff()
		eb.InitialInterval = 100 * time.Millisecond
		eb.MaxElapsedTime = publisher.Timeout
		ebCtx := backoff.WithContext(eb, ctx)

		if err := backoff.RetryNotify(publishBatch, ebCtx, notifyBackoff); err != nil {
			errors <- fmt.Errorf("%d events dropped: %s", len(batch), err)
		}
	}
}
