package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/watersofoblivion/dataless/lib/amz/amzmock"
	"github.com/watersofoblivion/dataless/lib/rest/restassert"
)

func TestCaptureController(t *testing.T) {
	ctx := context.Background()
	batchKey := "the-batch-key"
	deliveryStreamName := "the-delivery-stream-name"

	t.Run("Capture", func(t *testing.T) {
		evts := []map[string]string{
			{"Foo": "Bar", "Baz": "Quux"},
			{"Oof": "Rab", "Zab": "Xuuq"},
		}
		batch := map[string][]map[string]string{batchKey: evts}
		bs, err := json.Marshal(batch)
		require.NoError(t, err)

		req := events.APIGatewayProxyRequest{Body: string(bs)}

		records := make([]*firehose.Record, len(evts))
		for i, evt := range evts {
			bs, err := json.Marshal(evt)
			require.NoError(t, err)
			data := append(bs, []byte("\n")...)
			records[i] = new(firehose.Record).SetData(data)
		}
		input := new(firehose.PutRecordBatchInput)
		input.SetDeliveryStreamName(deliveryStreamName)
		input.SetRecords(records)

		output := new(firehose.PutRecordBatchOutput)
		output.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
			new(firehose.PutRecordBatchResponseEntry).SetRecordId("record-one"),
			new(firehose.PutRecordBatchResponseEntry).SetRecordId("record-two"),
		})

		captureResp := &captureResponse{
			FailedCount: 0,
			Records: []captureRecord{
				{Failed: false},
				{Failed: false},
			},
		}

		MockedCaptureController(t, batchKey, deliveryStreamName, func(controller *CaptureController, fh *amzmock.Firehose) {
			fh.On("PutRecordBatchWithContext", ctx, input).Return(output, nil)

			resp, err := controller.Capture(ctx, req)

			assert.NoError(t, err)
			restassert.Response(t, resp, http.StatusOK, captureResp, nil)
		})

		t.Run("handles failed puts", func(t *testing.T) {
			errCode := "the-error-code"
			errMsg := "the-error-message"

			output := new(firehose.PutRecordBatchOutput)
			output.SetFailedPutCount(1)
			output.SetRequestResponses([]*firehose.PutRecordBatchResponseEntry{
				new(firehose.PutRecordBatchResponseEntry).SetRecordId("record-one"),
				new(firehose.PutRecordBatchResponseEntry).SetRecordId("record-two").
					SetErrorCode(errCode).
					SetErrorMessage(errMsg),
			})

			captureResp := &captureResponse{
				FailedCount: 1,
				Records: []captureRecord{
					{Failed: false},
					{
						Failed:       true,
						ErrorCode:    errCode,
						ErrorMessage: errMsg,
					},
				},
			}

			MockedCaptureController(t, batchKey, deliveryStreamName, func(controller *CaptureController, fh *amzmock.Firehose) {
				fh.On("PutRecordBatchWithContext", ctx, input).Return(output, nil)

				resp, err := controller.Capture(ctx, req)

				assert.NoError(t, err)
				restassert.Response(t, resp, http.StatusOK, captureResp, nil)
			})
		})

		t.Run("returns", func(t *testing.T) {
			t.Run("Bad Request", func(t *testing.T) {
				t.Run("on non-JSON request", func(t *testing.T) {
					invalid := `{"invalid":"json`
					expectedErr := json.Unmarshal([]byte(invalid), (interface{})(nil))
					req := events.APIGatewayProxyRequest{Body: invalid}

					MockedCaptureController(t, batchKey, deliveryStreamName, func(controller *CaptureController, fh *amzmock.Firehose) {
						resp, err := controller.Capture(ctx, req)

						assert.NoError(t, err)
						restassert.Response(t, resp, http.StatusBadRequest, expectedErr, nil)
					})
				})

				t.Run("on missing batch key", func(t *testing.T) {
					expectedErr := fmt.Errorf("batch key %q not given", batchKey)
					req := events.APIGatewayProxyRequest{Body: `{}`}

					MockedCaptureController(t, batchKey, deliveryStreamName, func(controller *CaptureController, fh *amzmock.Firehose) {
						resp, err := controller.Capture(ctx, req)

						assert.NoError(t, err)
						restassert.Response(t, resp, http.StatusBadRequest, expectedErr, nil)
					})
				})

				t.Run("on multiple keys", func(t *testing.T) {
					batch := map[string][]map[string]string{
						batchKey:    evts,
						"other-key": make([]map[string]string, 0),
					}
					bs, err := json.Marshal(batch)
					require.NoError(t, err)
					expectedErr := fmt.Errorf("too many keys: %d", len(batch))
					req := events.APIGatewayProxyRequest{Body: string(bs)}

					MockedCaptureController(t, batchKey, deliveryStreamName, func(controller *CaptureController, fh *amzmock.Firehose) {
						resp, err := controller.Capture(ctx, req)

						assert.NoError(t, err)
						restassert.Response(t, resp, http.StatusBadRequest, expectedErr, nil)
					})
				})
			})

			t.Run("Internal Server Error", func(t *testing.T) {
				t.Run("on SDK error", func(t *testing.T) {
					expectedErr := fmt.Errorf("the-error")

					MockedCaptureController(t, batchKey, deliveryStreamName, func(controller *CaptureController, fh *amzmock.Firehose) {
						fh.On("PutRecordBatchWithContext", ctx, input).Return(nil, expectedErr)

						resp, err := controller.Capture(ctx, req)

						assert.NoError(t, err)
						restassert.Response(t, resp, http.StatusInternalServerError, expectedErr, nil)
					})
				})
			})
		})
	})
}
