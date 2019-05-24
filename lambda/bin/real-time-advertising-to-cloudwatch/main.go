package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-xray-sdk-go/xray"
)

var (
	sess            = session.New()
	cw              = cloudwatch.New(sess)
	encoder         = json.NewEncoder(os.Stdout)
	metricNamespace = os.Getenv("METRIC_NAMESPACE")
	location        = time.FixedZone("UTC", 0)
)

func init() {
	xray.AWS(cw.Client)
}

func main() {
	lambda.Start(handler)
}

// MetricDataLimit is the maximum number of metrics allowed in a batch of data
// sent to CloudWatch.
const MetricDataLimit = 20

// AdvertisingMetric is a metric to publish to CloudWatch
type AdvertisingMetric struct {
	Name  string    `json:"name"`
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.  This properly
// handles the timestamp format and timezone.
func (metric *AdvertisingMetric) UnmarshalJSON(bs []byte) error {
	var err error

	v := make(map[string]interface{})
	if err = json.Unmarshal(bs, &v); err != nil {
		return err
	}

	metric.Name = v["name"].(string)
	metric.At, err = time.ParseInLocation("2006-01-02 15:04:05.000", v["at"].(string), location)
	if err != nil {
		return err
	}
	metric.Value = v["value"].(float64)

	return nil
}

func handler(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) (events.KinesisAnalyticsOutputDeliveryResponse, error) {
	// Create a response
	output := events.KinesisAnalyticsOutputDeliveryResponse{
		Records: make([]events.KinesisAnalyticsOutputDeliveryResponseRecord, 0, len(input.Records)),
	}

	// Create the request to the upstream service
	putMetricDataInput := new(cloudwatch.PutMetricDataInput)
	putMetricDataInput.SetNamespace(metricNamespace)

	// Create buffers for the metrics and output records
	metricData := make([]*cloudwatch.MetricDatum, 0, MetricDataLimit)
	outputRecords := make([]*events.KinesisAnalyticsOutputDeliveryResponseRecord, 0, MetricDataLimit)

	// Process all the input records
	for _, record := range input.Records {
		// Create the output record
		outputRecord := &events.KinesisAnalyticsOutputDeliveryResponseRecord{RecordID: record.RecordID}

		// Parse the input record
		metric := new(AdvertisingMetric)
		if err := json.Unmarshal(record.Data, metric); err != nil {
			// Mark the input record as failed and move on
			log.Printf("Error unmarshaling record %s: %s", record.RecordID, err)
			outputRecord.Result = events.KinesisAnalyticsOutputDeliveryFailed
			output.Records = append(output.Records, *outputRecord)
			continue
		}

		log.Printf("Metric: %s = %0.2f @ %s", metric.Name, metric.Value, metric.At.Format(time.RFC3339Nano))

		// Append the metric data and output record to the buffers
		metricData = append(metricData, &cloudwatch.MetricDatum{
			MetricName: aws.String(metric.Name),
			Timestamp:  aws.Time(metric.At),
			Value:      aws.Float64(metric.Value),
		})
		outputRecords = append(outputRecords, outputRecord)

		// If the buffer is full, flush.
		if numMetrics := len(metricData); numMetrics == MetricDataLimit {
			// Complete the request
			putMetricDataInput.SetMetricData(metricData)

			// Publish the metrics and record the status for the output records
			status := events.KinesisAnalyticsOutputDeliveryOK
			if _, err := cw.PutMetricDataWithContext(ctx, putMetricDataInput); err != nil {
				log.Printf("metric delivery failed for %d metrics: %s", numMetrics, err)
				status = events.KinesisAnalyticsOutputDeliveryFailed
			}

			// Set the status of the output records and add them to the response
			for _, outputRecord := range outputRecords {
				outputRecord.Result = status
				output.Records = append(output.Records, *outputRecord)
			}

			// Reset the buffers
			metricData = make([]*cloudwatch.MetricDatum, 0, MetricDataLimit)
			outputRecords = make([]*events.KinesisAnalyticsOutputDeliveryResponseRecord, 0, MetricDataLimit)
		}
	}

	// Flush any remaining metric data
	numMetrics := len(metricData)
	if numMetrics > 0 {
		putMetricDataInput.SetMetricData(metricData)

		status := events.KinesisAnalyticsOutputDeliveryOK
		json.NewEncoder(os.Stdout).Encode(putMetricDataInput)
		if _, err := cw.PutMetricDataWithContext(ctx, putMetricDataInput); err != nil {
			log.Printf("metric delivery failed for %d metrics: %s", numMetrics, err)
			status = events.KinesisAnalyticsOutputDeliveryFailed
		}

		for _, outputRecord := range outputRecords {
			outputRecord.Result = status
			output.Records = append(output.Records, *outputRecord)
		}
	}

	// Done!
	json.NewEncoder(os.Stdout).Encode(output)
	return output, nil
}
