package inst

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

// CloudWatch service limits
const (
	MetricDataLimit      = 20
	MetricDimensionLimit = 10
)

// NamespacePublisher collects metrics in a single CloudWatch Metrics namespace
// and publishes them as a batch.
//
//   ctx := context.Background()
//   cw := cloudwatch.New(session.New())
//
//   // Create the publisher
//   publisher := NewNamespacePublisher(cw, "My/Namespace")
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
// The publisher will automatically flush when the buffer is full (up to
// MetricDataLimit metrics).  Be sure to call Flush() when done to flush any
// still-buffered metrics.
type NamespacePublisher struct {
	cw                 cloudwatchiface.CloudWatchAPI
	metrics            []*cloudwatch.MetricDatum
	recordIDs          []string
	putMetricDataInput *cloudwatch.PutMetricDataInput
	records            []events.KinesisAnalyticsOutputDeliveryResponseRecord
}

// NewNamespacePublisher constructs a publisher for a namespace.
func NewNamespacePublisher(cw cloudwatchiface.CloudWatchAPI, namespace string) *NamespacePublisher {
	putMetricDataInput := new(cloudwatch.PutMetricDataInput)
	putMetricDataInput.SetNamespace(namespace)

	return &NamespacePublisher{
		cw:                 cw,
		metrics:            make([]*cloudwatch.MetricDatum, 0, MetricDataLimit),
		recordIDs:          make([]string, 0, MetricDataLimit),
		putMetricDataInput: putMetricDataInput,
	}
}

// Publish a metric to CloudWatch.  The recordID is the ID of the input
// KinesisAnalytics record this metrics corresponds to.
func (publisher *NamespacePublisher) Publish(ctx context.Context, recordID string, metric *Metric) {
	publisher.recordIDs = append(publisher.recordIDs, recordID)
	publisher.metrics = append(publisher.metrics, &cloudwatch.MetricDatum{
		MetricName: aws.String(metric.Name),
		Timestamp:  aws.Time(metric.At),
		Value:      aws.Float64(metric.Value),
	})

	if numMetrics := len(publisher.metrics); numMetrics == MetricDataLimit {
		publisher.Flush(ctx)
	}
}

// Flush buffered metrics to CloudWatch.
func (publisher *NamespacePublisher) Flush(ctx context.Context) {
	numMetrics := len(publisher.metrics)
	if numMetrics > 0 {
		publisher.putMetricDataInput.SetMetricData(publisher.metrics)

		status := events.KinesisAnalyticsOutputDeliveryOK
		if _, err := publisher.cw.PutMetricDataWithContext(ctx, publisher.putMetricDataInput); err != nil {
			log.Printf("metric delivery failed for %d metrics: %s", numMetrics, err)
			status = events.KinesisAnalyticsOutputDeliveryFailed
		}

		for _, recordID := range publisher.recordIDs {
			publisher.records = append(publisher.records, events.KinesisAnalyticsOutputDeliveryResponseRecord{
				RecordID: recordID,
				Result:   status,
			})
		}

		publisher.metrics = make([]*cloudwatch.MetricDatum, 0, MetricDataLimit)
		publisher.recordIDs = make([]string, 0, MetricDataLimit)
	}
}

// Records processed and flushed to CloudWatch.
func (publisher *NamespacePublisher) Records() []events.KinesisAnalyticsOutputDeliveryResponseRecord {
	return publisher.records
}
