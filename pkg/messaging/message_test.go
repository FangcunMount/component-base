package messaging

import (
	"errors"
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

	_, ok, err = DecodeMessagePayload([]byte("legacy raw payload"))
	if err != nil {
		t.Fatalf("DecodeMessagePayload legacy: %v", err)
	}
	if ok {
		t.Fatalf("legacy raw payload should not decode as envelope")
	}
}
