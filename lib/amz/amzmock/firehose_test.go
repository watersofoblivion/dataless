package amzmock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFirehose(t *testing.T) {
	ctx := context.Background()
	err := fmt.Errorf("the-error")

	fh := new(Firehose)

	t.Run("PutRecordBatchWithContext", func(t *testing.T) {
		input := new(firehose.PutRecordBatchInput)
		input.SetDeliveryStreamName("the-delivery-stream")
		input.SetRecords([]*firehose.Record{
			new(firehose.Record).SetData([]byte("one")),
			new(firehose.Record).SetData([]byte("two")),
		})

		output := new(firehose.PutRecordBatchOutput)
		output.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
			new(firehose.PutRecordBatchResponseEntry).SetRecordId(uuid.New().String()),
		})

		fh.On("PutRecordBatchWithContext", ctx, input).Return(output, err)
		out, e := fh.PutRecordBatchWithContext(ctx, input)

		fh.AssertExpectations(t)
		assert.Equal(t, output, out)
		assert.Equal(t, err, e)

		t.Run("with nil output", func(t *testing.T) {
			fh := new(Firehose)

			fh.On("PutRecordBatchWithContext", mock.Anything, mock.Anything).Return(nil, nil)
			assert.NotPanics(t, func() { out, e = fh.PutRecordBatchWithContext(ctx, input) })
			fh.AssertExpectations(t)
			assert.Nil(t, out)
			assert.Nil(t, e)
		})
	})
}
