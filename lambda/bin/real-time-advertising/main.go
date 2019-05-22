package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/google/uuid"
)

var (
	sess    = session.New()
	cw      = cloudwatch.New(sess)
	encoder = json.NewEncoder(os.Stdout)
)

func init() {
	xray.AWS(cw.Client)
}

type Advertising struct {
	Ad         uuid.UUID `json:"ad_id"`
	Impression time.Time `json:"impression_at"`
	Click      time.Time `json:"click_at"`
}

func (advertising *Advertising) Clicked() bool {
	return advertising.Click.After(advertising.Impression)
}

const MetricDataLimit = 20

var (
	wg = new(sync.WaitGroup)
)

func publishMetrics(metrics <-chan *cloudwatch.MetricDatum) {
	defer wg.Done()

	input := new(cloudwatch.PutMetricDataInput)
	input.SetNamespace("Warehouse/Advertising")
	input.SetMetricData(make([]*cloudwatch.MetricDatum, 0, MetricDataLimit))

	defer func() {
		if len(input.MetricData) > 0 {
			if _, err := cw.PutMetricData(input); err != nil {
				log.Printf("error publishing metric data: %s", err)
			}
		}
	}()

	for datum := range metrics {
		input.MetricData = append(input.MetricData, datum)

		if len(input.MetricData) == MetricDataLimit {
			if _, err := cw.PutMetricData(input); err != nil {
				log.Printf("error publishing metric data: %s", err)
			}

			input.SetMetricData(make([]*cloudwatch.MetricDatum, 0, MetricDataLimit))
		}
	}
}

func generateMetricData(ads <-chan *Advertising, metrics chan<- *cloudwatch.MetricDatum) {
	defer wg.Done()
	defer close(metrics)

	for range ads {

	}
}

func handler(ctx context.Context, evt events.KinesisAnalyticsOutputDeliveryEvent) (events.KinesisAnalyticsOutputDeliveryResponse, error) {
	resp := events.KinesisAnalyticsOutputDeliveryResponse{
		Records: make([]events.KinesisAnalyticsOutputDeliveryResponseRecord, len(evt.Records)),
	}

	for i, record := range evt.Records {
		respRecord := events.KinesisAnalyticsOutputDeliveryResponseRecord{RecordID: record.RecordID}
		resp.Records[i] = respRecord

		advertising := new(Advertising)

		if err := json.Unmarshal(record.Data, advertising); err != nil {
			respRecord.Result = events.KinesisAnalyticsOutputDeliveryFailed
			continue
		}

		respRecord.Result = events.KinesisAnalyticsOutputDeliveryOK
	}

	return resp, nil
}

func main() {
	defer wg.Wait()

	ads := make(chan *Advertising)
	metrics := make(chan *cloudwatch.MetricDatum)

	defer close(ads)
	go publishMetrics(metrics)
	go generateMetricData(ads, metrics)

	lambda.Start(handler)
}
