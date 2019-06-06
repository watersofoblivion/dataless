package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"

	"github.com/watersofoblivion/dataless/lib/rest"
)

var (
	batchKey           = os.Getenv("BATCH_KEY")
	deliveryStreamName = os.Getenv("DELIVERY_STREAM_NAME")
	kf                 = firehose.New(session.New())
)

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	batch := make(map[string][]json.RawMessage)
	if err := json.Unmarshal([]byte(req.Body), &batch); err != nil {
		return rest.Respond(http.StatusBadRequest, err, nil)
	}

	records, found := batch[batchKey]
	if numKeys := len(batch); numKeys != 1 {
		return rest.Respond(http.StatusBadRequest, fmt.Errorf("expected exactly 1 key, found %d", numKeys), nil)
	}
	if !found {
		return rest.Respond(http.StatusBadRequest, fmt.Errorf("expected batch key %q", batchKey), nil)
	}

	inputRecords := make([]*firehose.Record, len(records))
	for i, record := range records {
		inputRecords[i] = &firehose.Record{Data: record}
	}

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(deliveryStreamName)
	input.SetRecords(inputRecords)
	resp, err := kf.PutRecordBatchWithContext(ctx, input)
	if err != nil {
		return rest.Respond(http.StatusInternalServerError, err, nil)
	}

	responseRecords := make([]map[string]interface{}, len(resp.RequestResponses))
	for i, record := range resp.RequestResponses {
		failed := record.ErrorCode != nil || record.ErrorMessage != nil
		responseRecords[i] = map[string]interface{}{"failed": failed}
		if failed {
			responseRecords[i]["error_code"] = aws.StringValue(record.ErrorCode)
			responseRecords[i]["error_message"] = aws.StringValue(record.ErrorMessage)
		}
	}

	return rest.Respond(http.StatusOK, map[string]interface{}{
		"failed_count": aws.Int64Value(resp.FailedPutCount),
		"records":      responseRecords,
	}, nil)
}
