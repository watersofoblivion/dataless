package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"

	"github.com/watersofoblivion/dataless/go/api"
)

var (
	kf                 = firehose.New(session.New())
	deliveryStreamName = os.Getenv("DELIVERY_STREAM_NAME")
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	evts, err := api.NewEvents(req)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, err)
	}

	input, err := evts.PutRecordsBatchInput()
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	resp, err := kf.PutRecordBatchWithContext(ctx, input)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	batch := api.NewBatchWriteResponse(len(evts.Events), resp)
	bs, err := json.Marshal(batch)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bs),
	}, nil
}
