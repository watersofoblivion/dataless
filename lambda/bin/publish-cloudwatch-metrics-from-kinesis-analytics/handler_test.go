package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TESTING", "true")
}

func TestHandler(t *testing.T) {
	numNamespaces := 2
	numMetrics := 2
	namespaces := make([]string, 2)
	recordIDs := make([]string, 4)

	ctx := context.Background()
	cw := new(mockCloudWatch)
	publisher = NewMetricsPublisher(cw)

	evt := events.KinesisAnalyticsOutputDeliveryEvent{}
	for namespace := 0; namespace < numNamespaces; namespace++ {
		namespaces[namespace] = fmt.Sprintf("namespace-%d", namespace)

		for metric := 0; metric < numMetrics; metric++ {
			idx := namespace*2 + metric

			name := fmt.Sprintf("namespace-%d-metric-%d", namespace, metric)
			recordIDs[idx] = fmt.Sprintf("%s-id", name)
			data, err := json.Marshal(map[string]interface{}{
				"namespace": namespaces[namespace],
				"name":      name,
				"at":        time.Now().Format(DateFormat),
				"value":     float64(idx),
			})
			require.NoError(t, err)

			evt.Records = append(evt.Records, events.KinesisAnalyticsOutputDeliveryEventRecord{
				RecordID: recordIDs[idx],
				Data:     data,
			})
		}
	}

	cw.On("PutMetricDataWithContext", ctx, mock.Anything).Return(nil, nil)
	cw.On("PutMetricDataWithContext", ctx, mock.Anything).Return(nil, nil)
	resp := handler(ctx, evt)
	cw.AssertExpectations(t)

	require.Len(t, resp.Records, len(recordIDs))
	for _, recordID := range recordIDs {
		expected := events.KinesisAnalyticsOutputDeliveryResponseRecord{
			RecordID: recordID,
			Result:   events.KinesisAnalyticsOutputDeliveryOK,
		}
		assert.Contains(t, resp.Records, expected)
	}

	t.Run("Drops malformed records", func(t *testing.T) {
		cw := new(mockCloudWatch)
		publisher = NewMetricsPublisher(cw)

		evt := events.KinesisAnalyticsOutputDeliveryEvent{
			Records: []events.KinesisAnalyticsOutputDeliveryEventRecord{
				events.KinesisAnalyticsOutputDeliveryEventRecord{
					RecordID: "dropped",
					Data:     []byte(`{"invalid":"json`),
				},
			},
		}

		resp := handler(ctx, evt)

		cw.AssertExpectations(t)
		assert.Contains(t, resp.Records, events.KinesisAnalyticsOutputDeliveryResponseRecord{
			RecordID: "dropped",
			Result:   events.KinesisAnalyticsOutputDeliveryOK,
		})
	})
}
