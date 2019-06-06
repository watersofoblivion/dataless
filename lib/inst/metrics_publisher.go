package inst

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

// MetricsPublisher collects CloudWatch metrics and publishes them as a batch.
//
//   ctx := context.Background()
//   cw := cloudwatch.New(session.New())
//
//   // Create the publisher
//   publisher := NewPublisher(cw)
//
//   // Publish metrics
//   publisher.Publish(ctx, "record-id-one", &Metric{ ... })
//   publisher.Publish(ctx, "record-id-two", &Metric{ ... })
//
//   // Flush any pending metrics
//   publisher.Flush(ctx)
//
//   // Get response records
//   records := publisher.Records()
//
// The publisher will automatically flush when any buffers full up to
// MetricDataLimit metrics.  Be sure to call Flush() when done to flush any
// still-buffered metrics.
type MetricsPublisher struct {
	cw         cloudwatchiface.CloudWatchAPI
	namespaces map[string]*NamespacePublisher
}

// NewMetricsPublisher constructs a new metrics publisher.
func NewMetricsPublisher(cw cloudwatchiface.CloudWatchAPI) *MetricsPublisher {
	return &MetricsPublisher{
		cw:         cw,
		namespaces: make(map[string]*NamespacePublisher),
	}
}

// Publish a metric to CloudWatch.  The recordID is the ID of the input
// KinesisAnalytics record this metrics corresponds to.
func (publisher *MetricsPublisher) Publish(ctx context.Context, recordID string, metric *Metric) {
	namespacePublisher, found := publisher.namespaces[metric.Namespace]
	if !found {
		namespacePublisher = NewNamespacePublisher(publisher.cw, metric.Namespace)
		publisher.namespaces[metric.Namespace] = namespacePublisher
	}

	namespacePublisher.Publish(ctx, recordID, metric)
}

// Flush buffered metrics in all namespaces to CloudWatch.
func (publisher *MetricsPublisher) Flush(ctx context.Context) {
	for _, namespace := range publisher.namespaces {
		namespace.Flush(ctx)
	}
}

// Records processed and flushed to CloudWatch across all namespaces.
func (publisher *MetricsPublisher) Records() []events.KinesisAnalyticsOutputDeliveryResponseRecord {
	records := make([]events.KinesisAnalyticsOutputDeliveryResponseRecord, 0)
	for _, namespace := range publisher.namespaces {
		records = append(records, namespace.records...)
	}
	return records
}
