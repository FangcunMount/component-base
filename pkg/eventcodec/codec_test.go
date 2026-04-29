package eventcodec

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/event"
)

func TestEncodeDecodeDomainEventEnvelope(t *testing.T) {
	t.Parallel()

	evt := event.New("sample.created", "Sample", "sample-1", map[string]string{"id": "sample-1"})
	payload, err := EncodeDomainEvent(evt)
	if err != nil {
		t.Fatalf("EncodeDomainEvent() error = %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.EventType != "sample.created" || env.AggregateID != "sample-1" {
		t.Fatalf("envelope = %#v", env)
	}

	decoded, err := DecodeDomainEvent(payload)
	if err != nil {
		t.Fatalf("DecodeDomainEvent() error = %v", err)
	}
	if decoded.EventID() != evt.EventID() || decoded.EventType() != evt.EventType() {
		t.Fatalf("decoded = %#v, want event id/type preserved", decoded)
	}
}

func TestMetadataFromEvent(t *testing.T) {
	t.Parallel()

	evt := event.Event[struct{}]{
		BaseEvent: event.BaseEvent{
			ID:                 "evt-1",
			EventTypeValue:     "sample.created",
			OccurredAtValue:    time.Date(2026, 4, 29, 1, 2, 3, 4_000_000, time.UTC),
			AggregateTypeValue: "Sample",
			AggregateIDValue:   "sample-1",
		},
	}
	metadata := MetadataFromEvent(evt, "api-server")
	if metadata["source"] != "api-server" || metadata["event_type"] != "sample.created" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if metadata["occurred_at"] != "2026-04-29T01:02:03.004Z" {
		t.Fatalf("occurred_at = %q", metadata["occurred_at"])
	}
}
