package events

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/jonboulle/clockwork"

	"github.com/watersofoblivion/dataless/lib/amz/amzmock"
	"github.com/watersofoblivion/dataless/lib/rest"
)

type Controller struct {
	Clock                clockwork.Clock
	Events               *rest.CaptureController
	AdvertisingInfoTable AdvertisingInfoTable
}

const (
	EnvVarEventsDeliveryStreamName string = "EVENTS_DELIVERY_STREAM_NAME"
	EnvVarAdvertisingInfoTableName string = "ADVERTISING_INFO_TABLE_NAME"
)

const (
	BatchKeyEvents string = "events"
)

func EnvController() *Controller {
	sess := session.New()

	deliveryStreamName := os.Getenv(EnvVarEventsDeliveryStreamName)
	fh := firehose.New(sess)
	events := rest.NewCaptureController(BatchKeyEvents, deliveryStreamName, fh)

	tableName := os.Getenv(EnvVarAdvertisingInfoTableName)
	ddb := dynamodb.New(sess)
	advertisingInfo := NewAdvertisingInfoTable(tableName, ddb)

	return NewController(events, advertisingInfo)
}

func NewController(events *rest.CaptureController, advertisingInfo AdvertisingInfoTable) *Controller {
	return &Controller{
		Clock:                clockwork.NewRealClock(),
		Events:               events,
		AdvertisingInfoTable: advertisingInfo,
	}
}

func MockedController(t *testing.T, eventsDeliveryStreamName string, fn func(controller *Controller, clock clockwork.Clock, fh *amzmock.Firehose, advertisingInfo *MockAdvertisingInfoTable)) {
	advertisingInfo := new(MockAdvertisingInfoTable)
	fh := new(amzmock.Firehose)
	events := rest.NewCaptureController(BatchKeyEvents, eventsDeliveryStreamName, fh)
	controller := NewController(events, advertisingInfo)
	controller.Clock = clockwork.NewFakeClockAt(time.Now())

	fn(controller, controller.Clock, fh, advertisingInfo)

	fh.AssertExpectations(t)
	advertisingInfo.AssertExpectations(t)
}

func (controller *Controller) Capture(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Events.Capture(ctx, req)
}

func (controller *Controller) PublishToCloudWatch(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) events.KinesisAnalyticsOutputDeliveryResponse {
	return events.KinesisAnalyticsOutputDeliveryResponse{}
}

func (controller *Controller) AdvertisingInfo(ctx context.Context, evt events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return rest.Respond(http.StatusInternalServerError, fmt.Errorf("not implemented"), nil)
}
