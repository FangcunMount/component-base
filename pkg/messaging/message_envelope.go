package messaging

import (
	"encoding/json"
	"fmt"
)

const messageEnvelopeType = "component-base.messaging.message.v1"

type wireMessageEnvelope struct {
	Type     string            `json:"type"`
	UUID     string            `json:"uuid,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Payload  []byte            `json:"payload"`
}

// EncodeMessagePayload serializes a Message for transports that cannot carry metadata.
func EncodeMessagePayload(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}
	env := wireMessageEnvelope{
		Type:     messageEnvelopeType,
		UUID:     msg.UUID,
		Metadata: copyMetadata(msg.Metadata),
		Payload:  msg.Payload,
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message envelope: %w", err)
	}
	return payload, nil
}

// DecodeMessagePayload decodes a metadata-preserving transport envelope.
// It returns ok=false for legacy raw payloads.
func DecodeMessagePayload(payload []byte) (*Message, bool, error) {
	var env wireMessageEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, false, nil
	}
	if env.Type != messageEnvelopeType {
		return nil, false, nil
	}
	return &Message{
		UUID:     env.UUID,
		Metadata: copyMetadata(env.Metadata),
		Payload:  env.Payload,
	}, true, nil
}

func copyMetadata(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
