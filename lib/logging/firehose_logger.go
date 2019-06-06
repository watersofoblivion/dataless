package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
)

const (
	FirehoseMaxBatchSize = 500
)

type FirehoseLogger struct {
	Timeout            time.Duration
	BatchSize          int
	Ticker             *time.Ticker
	DeliveryStreamName string
	Firehose           firehoseiface.FirehoseAPI
	wg                 *sync.WaitGroup
	values             chan interface{}
	errors             chan error
	done               chan struct{}
}

func NewFirehoseLogger(deliveryStreamName string, fh firehoseiface.FirehoseAPI) *FirehoseLogger {
	return &FirehoseLogger{
		BatchSize:          FirehoseMaxBatchSize,
		Ticker:             time.NewTicker(48 * time.Hour),
		DeliveryStreamName: deliveryStreamName,
		Firehose:           fh,
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
	input.SetDeliveryStreamName(logger.DeliveryStreamName)

	for batch := range batches {
		input.SetRecords(batch)

		resp, err := logger.Firehose.PutRecordBatchWithContext(ctx, input)
		if err != nil {
			return err
		}

		for i, record := range resp.RequestResponses {
			if aws.StringValue(record.ErrorCode) != "" || aws.StringValue(record.ErrorMessage) != "" {
				errors <- awserr.New(record.ErrorCode, record.ErrorMessage, nil)
			}
		}
	}
}
