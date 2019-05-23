package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"

	"github.com/watersofoblivion/dataless/go/api"
)

const DefaultBatchKey = "records"

var (
	kf                 = firehose.New(session.New())
	batchKey           = os.Getenv("BATCH_KEY")
	deliveryStreamName = os.Getenv("DELIVERY_STREAM_NAME")
)

func init() {
	if batchKey == "" {
		batchKey = DefaultBatchKey
	}
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	evts := make(map[string][]map[string]interface{})
	if err := json.Unmarshal([]byte(req.Body), &evts); err != nil {
		return api.ErrorResponse(http.StatusBadRequest, err)
	}

	inputRecords, found := evts[batchKey]
	if !found {
		return api.ErrorResponse(http.StatusBadRequest, fmt.Errorf("expected batch key %q", batchKey))
	}

	outputRecords := make([]*firehose.Record, len(inputRecords))
	buf := new(bytes.Buffer)
	enc := base64.NewEncoder(base64.StdEncoding, buf)
	for i, record := range inputRecords {
		if err := json.NewEncoder(enc).Encode(record); err != nil {
			return api.ErrorResponse(http.StatusBadRequest, err)
		}

		enc.Close()
		outputRecords[i] = &firehose.Record{Data: buf.Bytes()}
		buf.Reset()
	}

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(deliveryStreamName)
	input.SetRecords(outputRecords)
	resp, err := kf.PutRecordBatchWithContext(ctx, input)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	responseRecords := make([]map[string]interface{}, len(inputRecords))
	for i, record := range resp.RequestResponses {
		if record.ErrorCode != nil && record.ErrorMessage != nil {
			responseRecords[i] = map[string]interface{}{
				"failed":        true,
				"error_code":    aws.StringValue(record.ErrorCode),
				"error_message": aws.StringValue(record.ErrorMessage),
			}
		} else {
			responseRecords[i] = map[string]interface{}{
				"failed": false,
			}
		}
	}

	bs, err := json.Marshal(map[string]interface{}{
		"failed_count": aws.Int64Value(resp.FailedPutCount),
		"records":      responseRecords,
	})
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bs),
	}, nil
}
