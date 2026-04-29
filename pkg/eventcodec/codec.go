package eventcodec

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/event"
)

const OccurredAtLayout = "2006-01-02T15:04:05.000Z07:00"

type PayloadEncoder func(event.DomainEvent) ([]byte, error)
type PayloadDecoder func([]byte) (event.DomainEvent, error)

type Envelope struct {
	ID            string          `json:"id"`
	EventType     string          `json:"eventType"`
	OccurredAt    time.Time       `json:"occurredAt"`
	AggregateType string          `json:"aggregateType"`
	AggregateID   string          `json:"aggregateID"`
	Data          json.RawMessage `json:"data"`
}

type storedDomainEvent struct {
	event.BaseEvent
	Data json.RawMessage `json:"data"`
}

func EncodeDomainEvent(evt event.DomainEvent) ([]byte, error) {
	if evt == nil {
		return nil, fmt.Errorf("domain event is nil")
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return payload, nil
}

func DecodeEnvelope(payload []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, fmt.Errorf("failed to parse event envelope: %w", err)
	}
	return &env, nil
}

func DecodeDomainEvent(payload []byte) (event.DomainEvent, error) {
	env, err := DecodeEnvelope(payload)
	if err != nil {
		return nil, err
	}
	return storedDomainEvent{
		BaseEvent: event.BaseEvent{
			ID:                 env.ID,
			EventTypeValue:     env.EventType,
			OccurredAtValue:    env.OccurredAt,
			AggregateTypeValue: env.AggregateType,
			AggregateIDValue:   env.AggregateID,
		},
		Data: env.Data,
	}, nil
}

func MetadataFromEvent(evt event.DomainEvent, source string) map[string]string {
	if evt == nil {
		return map[string]string{}
	}
	return map[string]string{
		"event_type":     evt.EventType(),
		"aggregate_type": evt.AggregateType(),
		"aggregate_id":   evt.AggregateID(),
		"occurred_at":    evt.OccurredAt().Format(OccurredAtLayout),
		"source":         source,
	}
}
