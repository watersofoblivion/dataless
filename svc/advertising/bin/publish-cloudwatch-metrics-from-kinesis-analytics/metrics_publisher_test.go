package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestMetricsPublisher(t *testing.T) {
	numNamespaces := 2
	numMetrics := 2
	namespaces := make([]string, 2)
	recordIDs := make([]string, 4)
	metrics := make([]*Metric, 4)

	ctx := context.Background()
	cw := new(mockCloudWatch)
	publisher := NewMetricsPublisher(cw)

	for namespace := 0; namespace < numNamespaces; namespace++ {
		namespaces[namespace] = fmt.Sprintf("namespace-%d", namespace)

		for metric := 0; metric < numMetrics; metric++ {
			idx := namespace*2 + metric
			name := fmt.Sprintf("namespace-%d-metric-%d", namespace, metric)
			recordIDs[idx] = fmt.Sprintf("%s-id", name)
			metrics[idx] = &Metric{
				Namespace: namespaces[namespace],
				Name:      name,
				At:        time.Now(),
				Value:     float64(idx),
			}

			publisher.Publish(ctx, recordIDs[idx], metrics[idx])
		}
	}

	cw.AssertExpectations(t)

	cw.On("PutMetricDataWithContext", ctx, publisher.namespaces[namespaces[0]].putMetricDataInput).Return(nil, nil)
	cw.On("PutMetricDataWithContext", ctx, publisher.namespaces[namespaces[1]].putMetricDataInput).Return(nil, nil)
	publisher.Flush(ctx)
	cw.AssertExpectations(t)

	records := publisher.Records()

	expected := []events.KinesisAnalyticsOutputDeliveryResponseRecord{}
	for _, namespace := range namespaces {
		expected = append(expected, publisher.namespaces[namespace].Records()...)
	}

	for _, record := range expected {
		assert.Contains(t, records, record)
	}
}
