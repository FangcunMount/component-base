package messaging

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"sort"
)

const (
	messageEnvelopeType          = "component-base.messaging.message.v1"
	currentMessageSchemaRevision = 2
)

type wireMessageEnvelope struct {
	Type           string            `json:"type"`
	SchemaRevision int               `json:"schema_revision,omitempty"`
	UUID           string            `json:"uuid,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Payload        []byte            `json:"payload"`
	Checksum       string            `json:"checksum,omitempty"`
}

// EncodeMessagePayload serializes a Message for transports that cannot carry metadata.
func EncodeMessagePayload(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}
	env := wireMessageEnvelope{
		Type:           messageEnvelopeType,
		SchemaRevision: currentMessageSchemaRevision,
		UUID:           msg.UUID,
		Metadata:       copyMetadata(msg.Metadata),
		Payload:        msg.Payload,
	}
	env.Checksum = messageEnvelopeChecksum(env.UUID, env.Metadata, env.Payload)
	payload, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message envelope: %w", err)
	}
	return payload, nil
}

// DecodeMessagePayload decodes a metadata-preserving transport envelope.
// It returns ok=false for legacy raw payloads.
func DecodeMessagePayload(payload []byte) (*Message, bool, error) {
	// Probe only the discriminator first. Syntactically invalid or unmarked
	// payloads remain legacy raw messages; once the component envelope is
	// recognized, all of its fields are decoded strictly.
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil || probe.Type != messageEnvelopeType {
		return nil, false, nil
	}

	var env wireMessageEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		var recoverable struct {
			UUID     string            `json:"uuid,omitempty"`
			Metadata map[string]string `json:"metadata,omitempty"`
		}
		_ = json.Unmarshal(payload, &recoverable)
		return &Message{
			UUID: recoverable.UUID, Metadata: copyMetadata(recoverable.Metadata), Payload: append([]byte(nil), payload...),
		}, true, fmt.Errorf("failed to decode recognized message envelope: %w", err)
	}
	decoded := &Message{UUID: env.UUID, Metadata: copyMetadata(env.Metadata), Payload: env.Payload}
	switch env.SchemaRevision {
	case 0, 1:
		// Revision 1 did not carry a schema revision or checksum.
	case currentMessageSchemaRevision:
		if env.Checksum == "" {
			return decoded, true, fmt.Errorf("message envelope checksum is required")
		}
		expected := messageEnvelopeChecksum(env.UUID, env.Metadata, env.Payload)
		if subtle.ConstantTimeCompare([]byte(env.Checksum), []byte(expected)) != 1 {
			return decoded, true, fmt.Errorf("message envelope checksum mismatch")
		}
	default:
		return decoded, true, fmt.Errorf("unsupported message envelope schema revision %d", env.SchemaRevision)
	}
	return decoded, true, nil
}

func messageEnvelopeChecksum(uuid string, metadata map[string]string, payload []byte) string {
	digest := sha256.New()
	writeChecksumPart(digest, []byte(uuid))
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeChecksumPart(digest, []byte(key))
		writeChecksumPart(digest, []byte(metadata[key]))
	}
	writeChecksumPart(digest, payload)
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func writeChecksumPart(digest hash.Hash, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write(value)
}

func copyMetadata(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
