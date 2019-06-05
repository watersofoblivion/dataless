package svc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/watersofoblivion/dataless/lib/amzmock"
)

func TestCaptureController(t *testing.T) {
	ctx := context.Background()
	batchKey := "the-batch-key"
	deliveryStreamName := "the-delivery-stream-name"
	fh := new(amzmock.Firehose)

	controller := NewCaptureController(batchKey, deliveryStreamName, fh)

	t.Run("Capture", func(t *testing.T) {
		bs, err := json.Marshal(map[string]interface{}{
			batchKey: []map[string]interface{}{
				{"Foo": "Bar", "Baz": "Quux"},
				{"Oof": "Rab", "Zab": "Xuuq"},
			},
		})
		require.NoError(t, err)

		req := events.APIGatewayProxyRequest{
			Body: string(bs),
		}

		resp, err := controller.Capture(ctx, req)

		assert.NoError(t, err)
		_ = resp
	})
}
