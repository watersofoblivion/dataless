package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-xray-sdk-go/xray"
)

var publisher *MetricsPublisher

func init() {
	s := session.New()
	cw := cloudwatch.New(s)
	xray.AWS(cw.Client)

	publisher = NewMetricsPublisher(cw)
}

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
