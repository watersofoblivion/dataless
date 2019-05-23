package api

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Events struct {
	Events []*Event `json:"events"`
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
