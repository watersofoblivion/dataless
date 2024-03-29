package advertising

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"

	"github.com/watersofoblivion/sam/amz/amzmock"
	"github.com/watersofoblivion/sam/inst"
	"github.com/watersofoblivion/sam/rest"
)

// Controller is the main controller for the Advertising service.
type Controller struct {
	// Clock for mocking.
	Clock clockwork.Clock

	// Nested controllers to capture impressions and clicks data and publish it to
	// a Kinesis Firehose.
	Impressions *rest.CaptureController
	Clicks      *rest.CaptureController

	// Publisher to push metrics from the real-time apps into CloudWatch.
	Metrics *inst.MetricsPublisher

	// The DynamoDB table holding Ad traffic information.
	AdTrafficTable AdTrafficTable
}

// Environment variables
const (
	EnvVarImpressionsDeliveryStreamName string = "IMPRESSIONS_DELIVERY_STREAM_NAME"
	EnvVarClicksDeliveryStreamName      string = "CLICKS_DELIVERY_STREAM_NAME"
	EnvVarAdTrafficTableName            string = "AD_TRAFFIC_TABLE_NAME"
)

// Batch keys for data ingestion
const (
	BatchKeyImpressions string = "impressions"
	BatchKeyClicks      string = "clicks"
)

// EnvController constructs a controller from environment variables.  This is
// the constructor to use in the Lambda executable.
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

// NewController constructs a controller from the injected dependencies.
func NewController(impressions, clicks *rest.CaptureController, metrics *inst.MetricsPublisher, adTraffic AdTrafficTable) *Controller {
	return &Controller{
		Clock:          clockwork.NewRealClock(),
		Impressions:    impressions,
		Clicks:         clicks,
		Metrics:        metrics,
		AdTrafficTable: adTraffic,
	}
}

// MockController constructs a controller with mock dependencies.  The
// controller and the mocks are passed to the callback.  After the callback
// returns, expectations are automatically asserted on the mocks.
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

// CaptureImpressions writes impression events to a Kinesis Firehose by
// delegating to the Impressions CaptureController.
func (controller *Controller) CaptureImpressions(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Impressions.Capture(ctx, req)
}

// CaptureClicks writes click events to a Kinesis Firehose by delegating to the
// Clicks CaptureController.
func (controller *Controller) CaptureClicks(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Clicks.Capture(ctx, req)
}

// PublishToCloudWatc captures events from the Kinesis Analytics real-time
// applications and relays them into CloudWatch metrics.
func (controller *Controller) PublishToCloudWatch(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) events.KinesisAnalyticsOutputDeliveryResponse {
	results := map[string]string{}

	for _, record := range input.Records {
		metric := new(inst.Metric)
		if err := json.Unmarshal(record.Data, metric); err != nil {
			log.Printf("error unmarshaling record %s (dropped): %s", record.RecordID, err)
			results[record.RecordID] = events.KinesisAnalyticsOutputDeliveryOK
			continue
		}

		controller.Metrics.Publish(ctx, record.RecordID, metric)
	}

	controller.Metrics.Flush(ctx)

	for _, record := range controller.Metrics.Records() {
		results[record.RecordID] = record.Result
	}

	resp := events.KinesisAnalyticsOutputDeliveryResponse{}
	for _, record := range input.Records {
		resp.Records = append(resp.Records, events.KinesisAnalyticsOutputDeliveryResponseRecord{
			RecordID: record.RecordID,
			Result:   results[record.RecordID],
		})
	}

	return resp
}

// GetAdTraffic queries the Ad Traffic DynamoDB table for summary information
// about a particular Ad over a given time range.
//
// The time range can be controlled with the "start" and "end" query parameters.
// Results are paged, and paging can be controlled by the "page" and "limit"
// query parameters.
func (controller *Controller) GetAdTraffic(ctx context.Context, evt events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ad, err := uuid.Parse(evt.PathParameters["ad-id"])
	if err != nil {
		return rest.Respond(http.StatusBadRequest, err, nil)
	}

	start := time.Now()
	if startParam, found := evt.QueryStringParameters["start"]; found {
		start, err = time.Parse(DateFormatDay, startParam)
		if err != nil {
			return rest.Respond(http.StatusBadRequest, err, nil)
		}
	}

	end := time.Now()
	if endParam, found := evt.QueryStringParameters["end"]; found {
		end, err = time.Parse(DateFormatDay, endParam)
		if err != nil {
			return rest.Respond(http.StatusBadRequest, err, nil)
		}
	}

	var page map[string]*dynamodb.AttributeValue
	if pageParam, found := evt.QueryStringParameters["page"]; found {
		b64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(pageParam))
		if err := json.NewDecoder(b64).Decode(&page); err != nil {
			return rest.Respond(http.StatusBadRequest, err, nil)
		}
	}

	limit := int64(0)
	if limitParam, found := evt.QueryStringParameters["limit"]; found {
		limit, err = strconv.ParseInt(limitParam, 10, 64)
		if err != nil {
			return rest.Respond(http.StatusBadRequest, err, nil)
		}
	}

	days, next, err := controller.AdTrafficTable.Days(ctx, ad, start, end, page, limit)
	if err != nil {
		return rest.Respond(http.StatusInternalServerError, err, nil)
	}

	resp := struct {
		Count int64    `json:"count"`
		Next  string   `json:"next"`
		Days  []*AdDay `json:"days"`
	}{
		Count: int64(len(days)),
		Days:  days,
	}
	if len(next) > 0 {
		buf := new(bytes.Buffer)
		b64 := base64.NewEncoder(base64.StdEncoding, buf)
		if err := json.NewEncoder(b64).Encode(next); err != nil {
			return rest.Respond(http.StatusInternalServerError, err, nil)
		}
		b64.Close()
		resp.Next = buf.String()
	}

	return rest.Respond(http.StatusOK, resp, nil)
}
