package messaging

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestMessageSettlementIsIdempotent(t *testing.T) {
	msg := NewMessage("msg-1", []byte("payload"))
	ackCount := 0
	nackCount := 0
	msg.SetAckFunc(func() error {
		ackCount++
		return nil
	})
	msg.SetNackFunc(func() error {
		nackCount++
		return nil
	})

	if err := msg.Ack(); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	if err := msg.Ack(); err != nil {
		t.Fatalf("second Ack: %v", err)
	}
	if err := msg.Nack(); err != nil {
		t.Fatalf("Nack after Ack: %v", err)
	}

	if ackCount != 1 {
		t.Fatalf("ackCount = %d, want 1", ackCount)
	}
	if nackCount != 0 {
		t.Fatalf("nackCount = %d, want 0", nackCount)
	}
	if !msg.IsSettled() {
		t.Fatalf("message should be settled")
	}
}

func TestMessageSettlementReturnsFirstErrorWithoutRepeating(t *testing.T) {
	msg := NewMessage("msg-1", []byte("payload"))
	ackCount := 0
	wantErr := errors.New("ack failed")
	msg.SetAckFunc(func() error {
		ackCount++
		return wantErr
	})

	if err := msg.Ack(); !errors.Is(err, wantErr) {
		t.Fatalf("Ack error = %v, want %v", err, wantErr)
	}
	if err := msg.Ack(); err != nil {
		t.Fatalf("second Ack error = %v, want nil", err)
	}
	if ackCount != 1 {
		t.Fatalf("ackCount = %d, want 1", ackCount)
	}
}

func TestMessageEnvelopeRoundTripAndLegacyFallback(t *testing.T) {
	msg := NewMessage("msg-1", []byte(`{"ok":true}`))
	msg.Metadata["event_type"] = "sample.created"

	payload, err := EncodeMessagePayload(msg)
	if err != nil {
		t.Fatalf("EncodeMessagePayload: %v", err)
	}

	decoded, ok, err := DecodeMessagePayload(payload)
	if err != nil {
		t.Fatalf("DecodeMessagePayload: %v", err)
	}
	if !ok {
		t.Fatalf("DecodeMessagePayload ok = false")
	}
	if decoded.UUID != msg.UUID {
		t.Fatalf("UUID = %q, want %q", decoded.UUID, msg.UUID)
	}
	if string(decoded.Payload) != string(msg.Payload) {
		t.Fatalf("Payload = %s, want %s", decoded.Payload, msg.Payload)
	}
	if decoded.Metadata["event_type"] != "sample.created" {
		t.Fatalf("event_type metadata = %q", decoded.Metadata["event_type"])
	}
	if !strings.Contains(string(payload), `"schema_revision":2`) || !strings.Contains(string(payload), `"checksum":`) {
		t.Fatalf("revision 2 envelope fields are missing: %s", payload)
	}

	_, ok, err = DecodeMessagePayload([]byte("legacy raw payload"))
	if err != nil {
		t.Fatalf("DecodeMessagePayload legacy: %v", err)
	}
	if ok {
		t.Fatalf("legacy raw payload should not decode as envelope")
	}
}

func TestDecodeMessagePayloadAcceptsRevisionOneEnvelope(t *testing.T) {
	payload := []byte(`{"type":"component-base.messaging.message.v1","uuid":"legacy-1","metadata":{"event_type":"sample.created"},"payload":"bGVnYWN5"}`)
	decoded, ok, err := DecodeMessagePayload(payload)
	if err != nil || !ok {
		t.Fatalf("DecodeMessagePayload() ok/error = %v/%v", ok, err)
	}
	if decoded.UUID != "legacy-1" || string(decoded.Payload) != "legacy" {
		t.Fatalf("decoded legacy envelope = %#v", decoded)
	}
}

func TestRevisionTwoEnvelopeRemainsReadableByRevisionOneDecoder(t *testing.T) {
	message := NewMessage("message-1", []byte("payload"))
	message.Metadata["event_type"] = "sample.created"
	payload, err := EncodeMessagePayload(message)
	if err != nil {
		t.Fatal(err)
	}
	var legacy struct {
		Type     string            `json:"type"`
		UUID     string            `json:"uuid"`
		Metadata map[string]string `json:"metadata"`
		Payload  []byte            `json:"payload"`
	}
	if err := json.Unmarshal(payload, &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Type != messageEnvelopeType || legacy.UUID != message.UUID || string(legacy.Payload) != "payload" || legacy.Metadata["event_type"] != "sample.created" {
		t.Fatalf("legacy decoder view = %#v", legacy)
	}
}

func TestDecodeMessagePayloadRejectsRecognizedCorruption(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "invalid base64", payload: []byte(`{"type":"component-base.messaging.message.v1","schema_revision":2,"payload":"%%%","checksum":"bad"}`)},
		{name: "checksum mismatch", payload: []byte(`{"type":"component-base.messaging.message.v1","schema_revision":2,"uuid":"message-1","payload":"cGF5bG9hZA==","checksum":"bad"}`)},
		{name: "unsupported revision", payload: []byte(`{"type":"component-base.messaging.message.v1","schema_revision":3,"payload":"cGF5bG9hZA=="}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoded, ok, err := DecodeMessagePayload(test.payload)
			if err == nil || !ok || decoded == nil {
				t.Fatalf("DecodeMessagePayload() = %#v, %v, %v; want recognized error", decoded, ok, err)
			}
			if test.name != "invalid base64" && string(decoded.Payload) != "payload" {
				t.Fatalf("recoverable payload = %q", decoded.Payload)
			}
		})
	}
}
