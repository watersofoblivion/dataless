package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"

	"github.com/watersofoblivion/dataless/lib/svc/rest"
)

type CaptureController struct {
	MaxBatchSize       int
	BatchKey           string
	DeliveryStreamName string
	Firehose           firehoseiface.FirehoseAPI
	input              *firehose.PutRecordBatchInput
}

func NewCaptureController(batchKey, deliveryStreamName string, fh firehoseiface.FirehoseAPI) *CaptureController {
	return &CaptureController{
		BatchKey:           batchKey,
		DeliveryStreamName: deliveryStreamName,
		Firehose:           fh,
		input:              new(firehose.PutRecordBatchInput).SetDeliveryStreamName(deliveryStreamName),
	}
}

var newline = []byte("\n")

func (controller *CaptureController) Capture(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	batch := make(map[string][]json.RawMessage)
	if err := json.Unmarshal([]byte(req.Body), &batch); err != nil {
		return rest.Respond(http.StatusBadRequest, err, nil)
	}

	events, found := batch[controller.BatchKey]
	if !found {
		err := fmt.Errorf("batch key %q not found", controller.BatchKey)
		return rest.Respond(http.StatusBadRequest, err, nil)
	}

	records := make([]*firehose.Record, len(events))
	for i, event := range events {
		data := []byte(event)
		if !bytes.HasSuffix(data, newline) {
			data = append(data, newline...)
		}
		records[i] = new(firehose.Record).SetData(data)
	}

	output, err := controller.Firehose.PutRecordBatchWithContext(ctx, controller.input)
	if err != nil {
		return rest.Respond(http.StatusInternalServerError, err, nil)
	}

	failedCount := aws.Int64Value(output.FailedPutCount)
	responses := make([]map[string]interface{}, len(records))
	_ = responses

	resp := map[string]interface{}{
		"failed": failedCount,
	}

	return rest.Respond(http.StatusOK, resp, nil)
}
