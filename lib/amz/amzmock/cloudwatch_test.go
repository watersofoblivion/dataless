package amzmock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
)

func TestCloudWatch(t *testing.T) {
	ctx := context.Background()

	t.Run("PutMetricDataWithContext", func(t *testing.T) {
		mock := new(CloudWatch)

		input := new(cloudwatch.PutMetricDataInput)
		output := new(cloudwatch.PutMetricDataOutput)

		mock.On("PutMetricDataWithContext", ctx, input).Return(output, nil)

		returned, err := mock.PutMetricDataWithContext(ctx, input)

		mock.AssertExpectations(t)
		assert.Equal(t, output, returned)
		assert.NoError(t, err)

		t.Run("returns error", func(t *testing.T) {
			mock := new(CloudWatch)

			expectedErr := fmt.Errorf("the-error")

			mock.On("PutMetricDataWithContext", ctx, input).Return(nil, expectedErr)

			returned, err := mock.PutMetricDataWithContext(ctx, input)

			mock.AssertExpectations(t)
			assert.Nil(t, returned)
			assert.Error(t, err)
			assert.Equal(t, expectedErr, err)
		})
	})
}
