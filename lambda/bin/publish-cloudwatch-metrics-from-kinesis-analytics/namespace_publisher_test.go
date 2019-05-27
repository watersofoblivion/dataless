package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockCloudWatch struct {
	mock.Mock
	cloudwatchiface.CloudWatchAPI
}

func (mock *mockCloudWatch) PutMetricDataWithContext(ctx aws.Context, data *cloudwatch.PutMetricDataInput, options ...request.Option) (*cloudwatch.PutMetricDataOutput, error) {
	args := mock.Called(ctx, data)
	return nil, args.Error(1)
}

func TestNamespacePublisher(t *testing.T) {
	ctx := context.Background()

	namespace := "namespace"
	recordIDOne := "record-id-one"
	metricOne := &Metric{
		Namespace: namespace,
		Name:      "metric-one",
		At:        time.Now(),
		Value:     1.2,
	}
	recordIDTwo := "record-id-two"
	metricTwo := &Metric{
		Namespace: namespace,
		Name:      "metric-two",
		At:        time.Now(),
		Value:     3.4,
	}

	t.Run("Publish", func(t *testing.T) {
		cw := new(mockCloudWatch)
		publisher := NewNamespacePublisher(cw, namespace)

		publisher.Publish(ctx, recordIDOne, metricOne)
		publisher.Publish(ctx, recordIDTwo, metricTwo)
		cw.AssertExpectations(t)

		cw.On("PutMetricDataWithContext", ctx, publisher.putMetricDataInput).Return(nil, nil)
		publisher.Flush(ctx)
		cw.AssertExpectations(t)

		records := publisher.Records()
		require.Len(t, records, 2)
		assert.Equal(t, recordIDOne, records[0].RecordID)
		assert.Equal(t, events.KinesisAnalyticsOutputDeliveryOK, records[0].Result)
		assert.Equal(t, recordIDTwo, records[1].RecordID)
		assert.Equal(t, events.KinesisAnalyticsOutputDeliveryOK, records[1].Result)

		t.Run("when PutMetricData returns an error", func(t *testing.T) {
			cw := new(mockCloudWatch)
			publisher := NewNamespacePublisher(cw, namespace)

			publisher.Publish(ctx, recordIDOne, metricOne)
			publisher.Publish(ctx, recordIDTwo, metricTwo)
			cw.AssertExpectations(t)

			cw.On("PutMetricDataWithContext", ctx, publisher.putMetricDataInput).Return(nil, fmt.Errorf("an-error"))
			publisher.Flush(ctx)
			cw.AssertExpectations(t)

			records := publisher.Records()
			require.Len(t, records, 2)
			assert.Equal(t, recordIDOne, records[0].RecordID)
			assert.Equal(t, events.KinesisAnalyticsOutputDeliveryFailed, records[0].Result)
			assert.Equal(t, recordIDTwo, records[1].RecordID)
			assert.Equal(t, events.KinesisAnalyticsOutputDeliveryFailed, records[1].Result)
		})

		t.Run("flushes on a full batch", func(t *testing.T) {
			ids := make([]string, MetricDataLimit+1)

			cw := new(mockCloudWatch)
			publisher := NewNamespacePublisher(cw, namespace)

			cw.On("PutMetricDataWithContext", ctx, publisher.putMetricDataInput).Return(nil, nil)
			for i := 0; i < MetricDataLimit+1; i++ {
				ids[i] = fmt.Sprintf("record-%d", i)
				publisher.Publish(ctx, ids[i], &Metric{
					Namespace: namespace,
					Name:      fmt.Sprintf("metric-%d", i),
					At:        time.Now(),
					Value:     float64(i),
				})
			}
			cw.AssertExpectations(t)

			records := publisher.Records()
			require.Len(t, records, MetricDataLimit)
			for i := 0; i < MetricDataLimit; i++ {
				assert.Equal(t, ids[i], records[i].RecordID)
				assert.Equal(t, events.KinesisAnalyticsOutputDeliveryOK, records[i].Result)
			}

			cw.On("PutMetricDataWithContext", ctx, publisher.putMetricDataInput).Return(nil, nil)
			publisher.Flush(ctx)
			cw.AssertExpectations(t)

			records = publisher.Records()
			require.Len(t, records, MetricDataLimit+1)
			assert.Equal(t, fmt.Sprintf("record-%d", MetricDataLimit), records[MetricDataLimit].RecordID)
			assert.Equal(t, events.KinesisAnalyticsOutputDeliveryOK, records[MetricDataLimit].Result)
		})
	})
}
