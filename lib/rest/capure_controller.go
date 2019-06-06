package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"

	"github.com/watersofoblivion/dataless/lib/amz/amzmock"
)

const (
	CaptureControllerDefaultMaxBatchSize = 500
)

type CaptureController struct {
	BatchKey           string
	EnforceNewline     bool
	DeliveryStreamName string
	Firehose           firehoseiface.FirehoseAPI
}

func NewCaptureController(batchKey, deliveryStreamName string, fh firehoseiface.FirehoseAPI) *CaptureController {
	return &CaptureController{
		BatchKey:           batchKey,
		EnforceNewline:     true,
		DeliveryStreamName: deliveryStreamName,
		Firehose:           fh,
	}
}

func MockedCaptureController(t *testing.T, batchKey, deliveryStreamName string, fn func(controller *CaptureController, fh *amzmock.Firehose)) {
	fh := new(amzmock.Firehose)
	controller := NewCaptureController(batchKey, deliveryStreamName, fh)
	fn(controller, fh)
	fh.AssertExpectations(t)
}

var newline = []byte("\n")

func (controller *CaptureController) Capture(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	batch := make(map[string][]json.RawMessage)
	if err := json.Unmarshal([]byte(req.Body), &batch); err != nil {
		return Respond(http.StatusBadRequest, err, nil)
	}

	if batchLen := len(batch); batchLen > 1 {
		err := fmt.Errorf("too many keys: %d", batchLen)
		return Respond(http.StatusBadRequest, err, nil)
	}

	events, found := batch[controller.BatchKey]
	if !found {
		err := fmt.Errorf("batch key %q not given", controller.BatchKey)
		return Respond(http.StatusBadRequest, err, nil)
	}

	records := make([]*firehose.Record, len(events))
	for i, event := range events {
		records[i] = new(firehose.Record).SetData([]byte(event))
		if controller.EnforceNewline && !bytes.HasSuffix(records[i].Data, newline) {
			records[i].Data = append(records[i].Data, newline...)
		}
	}

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(controller.DeliveryStreamName)
	input.SetRecords(records)

	output, err := controller.Firehose.PutRecordBatchWithContext(ctx, input)
	if err != nil {
		return Respond(http.StatusInternalServerError, err, nil)
	}

	failedCount := aws.Int64Value(output.FailedPutCount)
	respRecords := make([]captureRecord, len(records))
	for i, resp := range output.RequestResponses {
		errCode := aws.StringValue(resp.ErrorCode)
		errMsg := aws.StringValue(resp.ErrorMessage)
		respRecords[i] = captureRecord{
			Failed:       errCode != "" || errMsg != "",
			ErrorCode:    errCode,
			ErrorMessage: errMsg,
		}
	}
	resp := captureResponse{
		FailedCount: failedCount,
		Records:     respRecords,
	}

	return Respond(http.StatusOK, resp, nil)
}

type captureResponse struct {
	FailedCount int64           `json:"failed_count"`
	Records     []captureRecord `json:"records"`
}

type captureRecord struct {
	Failed       bool   `json:"failed"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
