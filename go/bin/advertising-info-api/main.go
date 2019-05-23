package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/watersofoblivion/dataless/go/api"
)

var (
	ddb       = dynamodb.New(session.New())
	tableName = os.Getenv("TABLE_NAME")
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	request, err := api.NewAdvertisingInfoRequest(req)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, err)
	}

	query, err := request.Query(tableName)
	if err != nil {
		return api.ErrorResponse(http.StatusBadRequest, err)
	}

	res, err := ddb.QueryWithContext(ctx, query)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	resp, err := api.NewAdvertisingInfoResponse(res)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	bs, err := json.Marshal(resp)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bs),
	}, nil
}
