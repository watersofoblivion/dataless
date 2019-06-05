package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/watersofoblivion/dataless/lib/amzmock"
)

func TestCollector(t *testing.T) {
	ctx := context.Background()
	deliveryStreamName := "delivery-stream-name"
	fh := new(amzmock.Firehose)

	var publisher *FirehosePublisher
	t.Run("Constructor", func(t *testing.T) {
		publisher = NewFirehosePublisher(deliveryStreamName, fh)

		assert.Equal(t, time.Duration(0), publisher.Timeout)
		assert.Equal(t, FirehoseMaxBatchSize, publisher.BatchSize)
		assert.NotNil(t, publisher.Ticker)
	})

	go publisher.Go(ctx)

	record := map[string]string{"foo": "bar"}
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(record)
	require.NoError(t, err)
	bs := buf.Bytes()

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(deliveryStreamName)
	input.SetRecords([]*firehose.Record{
		{Data: bs},
	})

	output := new(firehose.PutRecordBatchOutput)
	output.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
		{RecordId: aws.String("the-record-id")},
	})

	fh.On("PutRecordBatchWithContext", ctx, input).Return(output, nil)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer publisher.Close(ctx)

		publisher.Publish(record)
		publisher.Publish(make(chan int))
	}()

	_, expectedErr := json.Marshal(make(chan int))

	errCount := 0
	for err := range publisher.Errors() {
		errCount++
		assert.Equal(t, expectedErr, err)
	}
	assert.Equal(t, 1, errCount)

	wg.Wait()
	fh.AssertExpectations(t)

	t.Run("retries failed records", func(t *testing.T) {
		fh := new(amzmock.Firehose)

		publisher := NewFirehosePublisher(deliveryStreamName, fh)
		go publisher.Go(ctx)

		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)

		recordOne := map[string]string{"record": "one"}
		err := encoder.Encode(recordOne)
		require.NoError(t, err)
		bsOne := buf.Bytes()

		buf.Reset()

		recordTwo := map[string]string{"record": "two"}
		err = encoder.Encode(recordTwo)
		require.NoError(t, err)
		bsTwo := buf.Bytes()

		buf.Reset()

		recordThree := map[string]string{"record": "three"}
		err = encoder.Encode(recordThree)
		require.NoError(t, err)
		bsThree := buf.Bytes()

		inputOne := new(firehose.PutRecordBatchInput)
		inputOne.SetDeliveryStreamName(deliveryStreamName)
		inputOne.SetRecords([]*firehose.Record{
			{Data: bsOne},
			{Data: bsTwo},
			{Data: bsThree},
		})

		outputOne := new(firehose.PutRecordBatchOutput)
		outputOne.SetFailedPutCount(2)
		outputOne.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
			{RecordId: aws.String("record-one"), ErrorCode: aws.String("error-code"), ErrorMessage: aws.String("error-message")},
			{RecordId: aws.String("record-two")},
			{RecordId: aws.String("record-three"), ErrorCode: aws.String("error-code"), ErrorMessage: aws.String("error-message")},
		})

		// TODO: Shouldn't be mock.Anything
		fh.On("PutRecordBatchWithContext", ctx, mock.Anything).Return(outputOne, nil).Once()

		inputTwo := new(firehose.PutRecordBatchInput)
		inputTwo.SetDeliveryStreamName(deliveryStreamName)
		inputTwo.SetRecords([]*firehose.Record{
			{Data: bsOne},
			{Data: bsThree},
		})

		outputTwo := new(firehose.PutRecordBatchOutput)
		outputTwo.SetFailedPutCount(1)
		outputTwo.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
			{RecordId: aws.String("record-one"), ErrorCode: aws.String("error-code"), ErrorMessage: aws.String("error-message")},
			{RecordId: aws.String("record-three")},
		})

		// TODO: Shouldn't be mock.Anything
		fh.On("PutRecordBatchWithContext", ctx, mock.Anything).Return(outputTwo, nil).Once()

		inputThree := new(firehose.PutRecordBatchInput)
		inputThree.SetDeliveryStreamName(deliveryStreamName)
		inputThree.SetRecords([]*firehose.Record{
			{Data: bsOne},
		})

		outputThree := new(firehose.PutRecordBatchOutput)
		outputThree.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
			{RecordId: aws.String("record-one")},
		})

		// TODO: Shouldn't be mock.Anything
		fh.On("PutRecordBatchWithContext", ctx, mock.Anything).Return(outputThree, nil).Once()

		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer publisher.Close(ctx)

			for _, record := range []map[string]string{recordOne, recordTwo, recordThree} {
				publisher.Publish(record)
			}
		}()

		for err := range publisher.Errors() {
			assert.NoError(t, err)
		}

		wg.Wait()
		fh.AssertExpectations(t)
	})

	t.Run("Close", func(t *testing.T) {
		t.Run("times out on context", func(t *testing.T) {

		})
	})

	t.Run("flushes", func(t *testing.T) {
		t.Run("on full buffer", func(t *testing.T) {
			fh := new(amzmock.Firehose)

			publisher := NewFirehosePublisher(deliveryStreamName, fh)
			go publisher.Go(ctx)

			records := make([]*firehose.Record, publisher.BatchSize)
			responses := make([]*firehose.PutRecordBatchResponseEntry, publisher.BatchSize)
			for i := range records {
				records[i] = &firehose.Record{Data: bs}
				responses[i] = &firehose.PutRecordBatchResponseEntry{
					RecordId: aws.String(fmt.Sprintf("record-%d", i)),
				}
			}

			inputOne := new(firehose.PutRecordBatchInput)
			inputOne.SetDeliveryStreamName(deliveryStreamName)
			inputOne.SetRecords(records)

			outputOne := new(firehose.PutRecordBatchOutput)
			outputOne.SetRequestResponses(responses)

			fh.On("PutRecordBatchWithContext", ctx, inputOne).Return(outputOne, nil)

			inputTwo := new(firehose.PutRecordBatchInput)
			inputTwo.SetDeliveryStreamName(deliveryStreamName)
			inputTwo.SetRecords(records[:1])

			outputTwo := new(firehose.PutRecordBatchOutput)
			outputTwo.SetRequestResponses(responses[:1])

			fh.On("PutRecordBatchWithContext", ctx, inputTwo).Return(outputTwo, nil)

			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer publisher.Close(ctx)

				for i := 0; i < publisher.BatchSize+1; i++ {
					publisher.Publish(record)
				}
			}()

			for err := range publisher.Errors() {
				assert.NoError(t, err)
			}

			wg.Wait()
			fh.AssertExpectations(t)
		})

		t.Run("on tick", func(t *testing.T) {
			fh := new(amzmock.Firehose)

			publisher := NewFirehosePublisher(deliveryStreamName, fh)
			publisher.Ticker = time.NewTicker(100 * time.Millisecond)
			go publisher.Go(ctx)

			fh.On("PutRecordBatchWithContext", ctx, input).Return(output, nil)

			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer publisher.Close(ctx)

				publisher.Publish(record)
				time.Sleep(150 * time.Millisecond)
			}()

			for err := range publisher.Errors() {
				assert.NoError(t, err)
			}

			wg.Wait()
			fh.AssertExpectations(t)
		})
	})

	t.Run("passes error", func(t *testing.T) {
		t.Run("on SDK error", func(t *testing.T) {
			returnedErr := fmt.Errorf("the-error")
			expectedErr := fmt.Errorf("1 events dropped: %s", returnedErr)

			fh := new(amzmock.Firehose)

			publisher := NewFirehosePublisher(deliveryStreamName, fh)
			publisher.Timeout = 100 * time.Millisecond
			go publisher.Go(ctx)

			fh.On("PutRecordBatchWithContext", mock.Anything, mock.Anything).Return(nil, returnedErr)

			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer publisher.Close(ctx)

				publisher.Publish(record)
			}()

			errCount := 0
			for err := range publisher.Errors() {
				errCount++
				assert.Equal(t, expectedErr, err)
			}
			assert.Equal(t, 1, errCount)

			wg.Wait()
			fh.AssertExpectations(t)
		})
	})
}
