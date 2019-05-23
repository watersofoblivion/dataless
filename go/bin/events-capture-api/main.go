package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
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
	evts := new(api.Events)
	if err := json.Unmarshal([]byte(req.Body), evts); err != nil {
		return api.ErrorResponse(http.StatusBadRequest, err)
	}

	buf := new(bytes.Buffer)
	records := make([]*firehose.Record, len(evts.Events))
	for i, evt := range evts.Events {
		buf.Reset()

		enc := base64.NewEncoder(base64.StdEncoding, buf)
		if err := json.NewEncoder(enc).Encode(evt); err != nil {
			return api.ErrorResponse(http.StatusInternalServerError, err)
		}
		enc.Close()

		records[i] = &firehose.Record{Data: buf.Bytes()}
	}

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(deliveryStreamName)
	input.SetRecords(records)
	resp, err := kf.PutRecordBatchWithContext(ctx, input)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	batch := &api.BatchWriteResponse{Records: make([]*api.BatchWriteRecord, len(evts.Events))}
	for i, record := range resp.RequestResponses {
		batch.Records[i] = new(api.BatchWriteRecord)

		if record.ErrorCode != nil && record.ErrorMessage != nil {
			batch.FailedCount++

			batch.Records[i] = &api.BatchWriteRecord{
				Failed:       true,
				ErrorCode:    aws.StringValue(record.ErrorCode),
				ErrorMessage: aws.StringValue(record.ErrorMessage),
			}
		}
	}

	bs, err := json.Marshal(batch)
	if err != nil {
		return api.ErrorResponse(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bs),
	}, nil
}
