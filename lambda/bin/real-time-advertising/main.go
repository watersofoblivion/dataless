package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var encoder = json.NewEncoder(os.Stdout)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, evt events.KinesisAnalyticsOutputDeliveryEvent) (events.KinesisAnalyticsOutputDeliveryResponse, error) {
	log.Printf("Events Output")
	if err := encoder.Encode(evt); err != nil {
		log.Printf("Error encoding request: %s", err)
	}
	return events.KinesisAnalyticsOutputDeliveryResponse{}, nil
}
