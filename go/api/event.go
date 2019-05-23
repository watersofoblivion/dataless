package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/google/uuid"
)

type Events struct {
	Events []*Event `json:"events"`
}

func NewEvents(req events.APIGatewayProxyRequest) (*Events, error) {
	evts := new(Events)
	return evts, json.Unmarshal([]byte(req.Body), evts)
}

func (events *Events) Records() ([]*firehose.Record, error) {
	var err error

	records := make([]*firehose.Record, len(events.Events))
	for i, event := range events.Events {
		if records[i], err = event.Record(); err != nil {
			return nil, err
		}
	}
	return records, nil
}

func (events *Events) PutRecordBatchInput(deliveryStreamName string) (*firehose.PutRecordBatchInput, error) {
	records, err := events.Records()
	if err != nil {
		return nil, err
	}

	input := new(firehose.PutRecordBatchInput)
	input.SetDeliveryStreamName(deliveryStreamName)
	input.SetRecords(records)
	return input, nil
}

type Event struct {
	Session    uuid.UUID
	Context    uuid.UUID
	Parent     uuid.UUID
	ActorType  string
	Actor      uuid.UUID
	EventType  string
	Event      uuid.UUID
	ObjectType string
	Object     uuid.UUID
	OccurredAt time.Time
}

func (event *Event) Record() (*firehose.Record, error) {
	buf := new(bytes.Buffer)

	enc := base64.NewEncoder(base64.StdEncoding, buf)
	if err := json.NewEncoder(enc).Encode(event); err != nil {
		return nil, err
	}
	enc.Close()

	return &firehose.Record{Data: buf.Bytes()}, nil
}

func (event *Event) MarshalJSON() ([]byte, error) {
	evt := map[string]interface{}{
		"session_id":  event.Session.String(),
		"context_id":  event.Context.String(),
		"actor_type":  event.ActorType,
		"actor_id":    event.Actor.String(),
		"event_type":  event.EventType,
		"event_id":    event.Event.String(),
		"object_type": event.ObjectType,
		"object_id":   event.Object.String(),
		"occurred_at": event.OccurredAt.Format("2006-01-02 15:04:05.000"),
	}

	if event.Parent != uuid.Nil {
		evt["parent_id"] = event.Parent.String()
	}

	return json.Marshal(evt)
}
