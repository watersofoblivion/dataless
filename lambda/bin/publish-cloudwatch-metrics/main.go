package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-xray-sdk-go/xray"
)

var (
	cw              cloudwatchiface.CloudWatchAPI
	metricNamespace = os.Getenv("METRIC_NAMESPACE")
	location        = time.FixedZone("UTC", 0)
)

func init() {
	s := session.New()
	c := cloudwatch.New(s)
	xray.AWS(c.Client)
	cw = c
}

func main() {
	lambda.Start(handler)
}

// CloudWatch service limits
const (
	MetricDataLimit      = 20
	MetricDimensionLimit = 10
)

var publisher = NewMetricsPublisher()

func handler(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) (events.KinesisAnalyticsOutputDeliveryResponse, error) {
	// Create a response
	resp := events.KinesisAnalyticsOutputDeliveryResponse{}

	// Process all the input records
	for _, record := range input.Records {
		// Parse the input record
		metric := new(Metric)
		if err := json.Unmarshal(record.Data, metric); err != nil {
			// Mark the input record as failed and move on
			log.Printf("error unmarshaling record %s (dropped): %s", record.RecordID, err)
			resp.Records = append(resp.Records, events.KinesisAnalyticsOutputDeliveryResponseRecord{
				RecordID: record.RecordID,
				Result:   events.KinesisAnalyticsOutputDeliveryOK,
			})
			continue
		}

		// Publish the record
		publisher.Publish(ctx, record.RecordID, metric)
	}

	// Flush any remaining records
	publisher.Flush(ctx)

	// Append output records
	resp.Records = append(resp.Records, publisher.Records()...)

	// Done!
	return resp, nil
}

type MetricsPublisher struct {
	namespaces map[string]*NamespacePublisher
}

func NewMetricsPublisher() *MetricsPublisher {
	return &MetricsPublisher{
		namespaces: make(map[string]*NamespacePublisher),
	}
}

func (publisher *MetricsPublisher) Publish(ctx context.Context, recordID string, metric *Metric) {
	namespacePublisher, found := publisher.namespaces[metric.Namespace]
	if !found {
		namespacePublisher = NewNamespacePublisher(metric.Namespace)
		publisher.namespaces[metric.Namespace] = namespacePublisher
	}

	namespacePublisher.Publish(ctx, recordID, metric)
}

func (publisher *MetricsPublisher) Flush(ctx context.Context) {
	for _, namespace := range publisher.namespaces {
		namespace.Flush(ctx)
	}
}

func (publisher *MetricsPublisher) Records() []events.KinesisAnalyticsOutputDeliveryResponseRecord {
	records := make([]events.KinesisAnalyticsOutputDeliveryResponseRecord, 0)
	for _, namespace := range publisher.namespaces {
		records = append(records, namespace.records...)
	}
	return records
}

type NamespacePublisher struct {
	metrics            []*cloudwatch.MetricDatum
	recordIDs          []string
	putMetricDataInput *cloudwatch.PutMetricDataInput
	records            []events.KinesisAnalyticsOutputDeliveryResponseRecord
}

func NewNamespacePublisher(namespace string) *NamespacePublisher {
	putMetricDataInput := new(cloudwatch.PutMetricDataInput)
	putMetricDataInput.SetNamespace(namespace)

	return &NamespacePublisher{
		metrics:            make([]*cloudwatch.MetricDatum, 0, MetricDataLimit),
		recordIDs:          make([]string, 0, MetricDataLimit),
		putMetricDataInput: putMetricDataInput,
	}
}

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

func (publisher *NamespacePublisher) Flush(ctx context.Context) {
	numMetrics := len(publisher.metrics)
	if numMetrics > 0 {
		publisher.putMetricDataInput.SetMetricData(publisher.metrics)

		status := events.KinesisAnalyticsOutputDeliveryOK
		if _, err := cw.PutMetricDataWithContext(ctx, publisher.putMetricDataInput); err != nil {
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

// Metric is a metric to publish to CloudWatch
type Metric struct {
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	At         time.Time         `json:"at"`
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.  This properly
// handles the timestamp format and timezone.
func (metric *Metric) UnmarshalJSON(bs []byte) error {
	var err error

	v := make(map[string]interface{})
	if err = json.Unmarshal(bs, &v); err != nil {
		return err
	}

	metric.Namespace = v["namespace"].(string)
	metric.Name = v["name"].(string)
	metric.At, err = time.ParseInLocation("2006-01-02 15:04:05.000", v["at"].(string), location)
	if err != nil {
		return err
	}
	metric.Value = v["value"].(float64)

	delete(v, "namespace")
	delete(v, "name")
	delete(v, "at")
	delete(v, "value")

	metric.Dimensions = make(map[string]string)
	for k, v := range v {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected dimension %q to be a string, found %T", k, v)
		}
		metric.Dimensions[k] = s
	}

	if numDimensions := len(metric.Dimensions); numDimensions >= MetricDimensionLimit {
		return fmt.Errorf("%d dimensions on metric %q %q is greater than the limit of %d dimensions", numDimensions, metricNamespace, metric.Name, MetricDimensionLimit)
	}

	return nil
}
