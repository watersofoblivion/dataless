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
	"github.com/jonboulle/clockwork"

	"github.com/watersofoblivion/dataless/lib/svc"
)

type Controller struct {
	*svc.Controller
	clock           clockwork.Clock
	advertisingInfo AdvertisingInfoTable
}

const (
	EnvVarAdvertisingInfoTableName string = "ADVERTISING_INFO_TABLE_NAME"
)

func EnvController() *Controller {
	clock := clockwork.NewRealClock()

	sess := session.New()

	tableName := os.Getenv(EnvVarAdvertisingInfoTableName)
	ddb := dynamodb.New(sess)
	advertisingInfo := NewAdvertisingInfoTable(tableName, ddb)

	return NewController(clock, advertisingInfo)
}

func NewController(clock clockwork.Clock, advertisingInfo AdvertisingInfoTable) *Controller {
	return &Controller{
		Controller:      svc.NewController(),
		clock:           clock,
		advertisingInfo: advertisingInfo,
	}
}

func MockedController(t *testing.T, fn func(controller *Controller, clock clockwork.Clock, advertisingInfo *MockAdvertisingInfoTable)) {
	clock := clockwork.NewFakeClockAt(time.Now())
	advertisingInfo := new(MockAdvertisingInfoTable)
	controller := NewController(clock, advertisingInfo)

	fn(controller, clock, advertisingInfo)

	advertisingInfo.AssertExpectations(t)
}

func (controller *Controller) Capture(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Respond(http.StatusInternalServerError, fmt.Errorf("not implemented"), nil)
}

func (controller *Controller) PublishToCloudWatch(ctx context.Context, input events.KinesisAnalyticsOutputDeliveryEvent) events.KinesisAnalyticsOutputDeliveryResponse {
	return events.KinesisAnalyticsOutputDeliveryResponse{}
}

func (controller *Controller) AdvertisingInfoTable(ctx context.Context, evt events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return controller.Respond(http.StatusInternalServerError, fmt.Errorf("not implemented"), nil)
}
