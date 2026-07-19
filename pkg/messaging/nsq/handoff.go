package nsq

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/FangcunMount/component-base/pkg/messaging"
)

const (
	failedHandoffEnvelopeType = "component-base.messaging.failed.v1"
	failedHandoffChannel      = "cb-failed-handler"
)

type failedHandoffEnvelope struct {
	Type      string            `json:"type"`
	Provider  string            `json:"provider"`
	Topic     string            `json:"topic"`
	Channel   string            `json:"channel"`
	UUID      string            `json:"uuid"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Payload   []byte            `json:"payload"`
	Attempts  int               `json:"attempts"`
	Timestamp int64             `json:"timestamp,omitempty"`
	Cause     string            `json:"cause"`
}

func failedHandoffTopic(topic, channel string) string {
	digest := sha256.Sum256([]byte(topic + "\x00" + channel))
	return "cb.failed." + hex.EncodeToString(digest[:12])
}

func encodeFailedHandoff(topic, channel string, message *messaging.Message, attempts int, cause error) ([]byte, error) {
	if message == nil || message.UUID == "" || topic == "" || channel == "" || attempts < 1 || cause == nil {
		return nil, fmt.Errorf("invalid NSQ failed-message handoff")
	}
	envelope := failedHandoffEnvelope{
		Type: failedHandoffEnvelopeType, Provider: "nsq", Topic: topic, Channel: channel,
		UUID: message.UUID, Metadata: copyStringMap(message.Metadata), Payload: append([]byte(nil), message.Payload...),
		Attempts: attempts, Timestamp: message.Timestamp, Cause: cause.Error(),
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode NSQ failed-message handoff: %w", err)
	}
	return payload, nil
}

func decodeFailedHandoff(payload []byte) (messaging.FailedMessage, error) {
	var envelope failedHandoffEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return messaging.FailedMessage{}, fmt.Errorf("decode NSQ failed-message handoff: %w", err)
	}
	if envelope.Type != failedHandoffEnvelopeType || envelope.Provider != "nsq" || envelope.Topic == "" || envelope.Channel == "" || envelope.UUID == "" || envelope.Attempts < 1 || envelope.Cause == "" {
		return messaging.FailedMessage{}, fmt.Errorf("invalid NSQ failed-message handoff")
	}
	message := &messaging.Message{
		UUID: envelope.UUID, Metadata: copyStringMap(envelope.Metadata), Payload: append([]byte(nil), envelope.Payload...),
		Attempts: uint16(envelope.Attempts), Timestamp: envelope.Timestamp, Topic: envelope.Topic, Channel: envelope.Channel,
	}
	return messaging.FailedMessage{
		Provider: "nsq", Topic: envelope.Topic, Channel: envelope.Channel,
		Message: message, Attempts: envelope.Attempts, Cause: errors.New(envelope.Cause),
	}, nil
}

func copyStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
