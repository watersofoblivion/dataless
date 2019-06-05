package logging

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

type FirehoseLogger struct {
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

func NewFirehoseLogger(deliveryStreamName string, fh firehoseiface.FirehoseAPI) *FirehoseLogger {
	return &FirehoseLogger{
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

func (logger *FirehoseLogger) Go(ctx context.Context) {
	records := make(chan []byte)
	batches := make(chan []*firehose.Record)

	logger.wg.Add(1)
	go logger.serialize(ctx, logger.values, records, logger.errors)

	logger.wg.Add(1)
	go logger.publish(ctx, records, batches)

	logger.wg.Add(1)
	go logger.flush(ctx, batches, logger.errors)

	logger.wg.Wait()
	logger.done <- struct{}{}
}

func (logger *FirehoseLogger) Log(v interface{}) {
	logger.values <- v
}

func (logger *FirehoseLogger) Errors() <-chan error {
	return logger.errors
}

func (logger *FirehoseLogger) Close(ctx context.Context) {
	close(logger.values)
	defer close(logger.errors)

	select {
	case <-ctx.Done():
	case <-logger.done:
	}
}

func (logger *FirehoseLogger) serialize(ctx context.Context, values <-chan interface{}, records chan<- []byte, errors chan<- error) {
	defer logger.wg.Done()
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

func (logger *FirehoseLogger) publish(ctx context.Context, records <-chan []byte, batches chan<- []*firehose.Record) {
	defer logger.wg.Done()

	batch := make([]*firehose.Record, 0, FirehoseMaxBatchSize)

	defer func() {
		defer close(batches)

		if len(batch) > 0 {
			batches <- batch
		}
	}()

	for {
		select {
		case <-logger.Ticker.C:
			batches <- batch
			batch = make([]*firehose.Record, 0, logger.BatchSize)
		case bs, more := <-records:
			if !more {
				return
			}
			batch = append(batch, &firehose.Record{Data: bs})
			if len(batch) == logger.BatchSize {
				batches <- batch
				batch = make([]*firehose.Record, 0, logger.BatchSize)
			}
		}
	}
}

func (logger *FirehoseLogger) flush(ctx context.Context, batches <-chan []*firehose.Record, errors chan<- error) {
	defer logger.wg.Done()

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(logger.deliveryStreamName)

	var batch []*firehose.Record

	publishBatch := func() error {
		resp, err := logger.fh.PutRecordBatchWithContext(ctx, input)
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
		eb.MaxElapsedTime = logger.Timeout
		ebCtx := backoff.WithContext(eb, ctx)

		if err := backoff.RetryNotify(publishBatch, ebCtx, notifyBackoff); err != nil {
			errors <- fmt.Errorf("%d events dropped: %s", len(batch), err)
		}
	}
}
