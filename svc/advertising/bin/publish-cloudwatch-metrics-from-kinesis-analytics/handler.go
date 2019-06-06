package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

var publisher *MetricsPublisher

func init() {
	s := session.New()
	cw := cloudwatch.New(s)

	publisher = NewMetricsPublisher(cw)
}

func Handler(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) events.KinesisAnalyticsOutputDeliveryResponse {
	resp := events.KinesisAnalyticsOutputDeliveryResponse{}

	for _, record := range input.Records {
		metric := new(Metric)
		if err := json.Unmarshal(record.Data, metric); err != nil {
			log.Printf("error unmarshaling record %s (dropped): %s", record.RecordID, err)
			resp.Records = append(resp.Records, events.KinesisAnalyticsOutputDeliveryResponseRecord{
				RecordID: record.RecordID,
				Result:   events.KinesisAnalyticsOutputDeliveryOK,
			})
			continue
		}

		publisher.Publish(ctx, record.RecordID, metric)
	}

	publisher.Flush(ctx)

	resp.Records = append(resp.Records, publisher.Records()...)
	return resp
}
