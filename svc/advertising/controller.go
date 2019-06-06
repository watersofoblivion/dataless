package advertising

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/jonboulle/clockwork"

	"github.com/watersofoblivion/dataless/lib/amz/amzmock"
	"github.com/watersofoblivion/dataless/lib/inst"
	"github.com/watersofoblivion/dataless/lib/rest"
)

type Controller struct {
	Clock          clockwork.Clock
	Impressions    *rest.CaptureController
	Clicks         *rest.CaptureController
	Metrics        *inst.MetricsPublisher
	AdTrafficTable AdTrafficTable
}

const (
	EnvVarImpressionsDeliveryStreamName string = "IMPRESSIONS_DELIVERY_STREAM_NAME"
	EnvVarClicksDeliveryStreamName      string = "CLICKS_DELIVERY_STREAM_NAME"
	EnvVarAdTrafficTableName            string = "AD_TRAFFIC_TABLE_NAME"
)

const (
	BatchKeyImpressions string = "impressions"
	BatchKeyClicks      string = "clicks"
)

func EnvController() *Controller {
	sess := session.New()

	fh := firehose.New(sess)

	impressionsDeliveryStreamName := os.Getenv(EnvVarImpressionsDeliveryStreamName)
	impressions := rest.NewCaptureController(BatchKeyImpressions, impressionsDeliveryStreamName, fh)

	clicksDeliveryStreamName := os.Getenv(EnvVarClicksDeliveryStreamName)
	clicks := rest.NewCaptureController(BatchKeyClicks, clicksDeliveryStreamName, fh)

	cw := cloudwatch.New(sess)
	metrics := inst.NewMetricsPublisher(cw)

	tableName := os.Getenv(EnvVarAdTrafficTableName)
	ddb := dynamodb.New(sess)
	adTraffic := NewAdTrafficTable(tableName, ddb)

	return NewController(impressions, clicks, metrics, adTraffic)
}

func NewController(impressions, clicks *rest.CaptureController, metrics *inst.MetricsPublisher, adTraffic AdTrafficTable) *Controller {
	return &Controller{
		Clock:          clockwork.NewRealClock(),
		Impressions:    impressions,
		Clicks:         clicks,
		Metrics:        metrics,
		AdTrafficTable: adTraffic,
	}
}

func MockedController(t *testing.T, impressionsDeliveryStreamName, clicksDeliveryStreamName string, fn func(controller *Controller, clock clockwork.Clock, fh *amzmock.Firehose, adTraffic *MockAdTrafficTable)) {
	adTraffic := new(MockAdTrafficTable)
	fh := new(amzmock.Firehose)
	impressions := rest.NewCaptureController(BatchKeyImpressions, impressionsDeliveryStreamName, fh)
	clicks := rest.NewCaptureController(BatchKeyClicks, clicksDeliveryStreamName, fh)
	cw := new(amzmock.CloudWatch)
	metrics := inst.NewMetricsPublisher(cw)
	controller := NewController(impressions, clicks, metrics, adTraffic)
	controller.Clock = clockwork.NewFakeClockAt(time.Now())

	fn(controller, controller.Clock, fh, adTraffic)

	fh.AssertExpectations(t)
	adTraffic.AssertExpectations(t)
}

func (controller *Controller) CaptureImpressions(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Impressions.Capture(ctx, req)
}

func (controller *Controller) CaptureClicks(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Clicks.Capture(ctx, req)
}

func (controller *Controller) PublishToCloudWatch(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) (events.KinesisAnalyticsOutputDeliveryResponse, error) {
	resp := events.KinesisAnalyticsOutputDeliveryResponse{}

	for _, record := range input.Records {
		metric := new(inst.Metric)
		fmt.Println(string(record.Data))
		if err := json.Unmarshal(record.Data, metric); err != nil {
			log.Printf("error unmarshaling record %s (dropped): %s", record.RecordID, err)
			resp.Records = append(resp.Records, events.KinesisAnalyticsOutputDeliveryResponseRecord{
				RecordID: record.RecordID,
				Result:   events.KinesisAnalyticsOutputDeliveryOK,
			})
			continue
		}

		controller.Metrics.Publish(ctx, record.RecordID, metric)
	}

	controller.Metrics.Flush(ctx)

	resp.Records = append(resp.Records, controller.Metrics.Records()...)
	log.Printf("Records:")
	for _, record := range resp.Records {
		json.NewEncoder(os.Stdout).Encode(record)
	}
	return resp, nil
}

func (controller *Controller) GetAdTraffic(ctx context.Context, evt events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return rest.Respond(http.StatusInternalServerError, fmt.Errorf("not implemented"), nil)
}
