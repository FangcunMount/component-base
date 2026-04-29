package eventmessaging

import (
	"testing"

	"github.com/FangcunMount/component-base/pkg/event"
)

func TestBuildMessageUsesEventPayloadAndMetadata(t *testing.T) {
	t.Parallel()

	evt := event.New("sample.created", "Sample", "sample-1", map[string]string{"id": "sample-1"})
	msg, err := BuildMessage(evt, "api-server")
	if err != nil {
		t.Fatalf("BuildMessage() error = %v", err)
	}
	if msg.UUID != evt.EventID() {
		t.Fatalf("message UUID = %q, want %q", msg.UUID, evt.EventID())
	}
	if msg.Metadata["event_type"] != "sample.created" || msg.Metadata["source"] != "api-server" {
		t.Fatalf("metadata = %#v", msg.Metadata)
	}
	if len(msg.Payload) == 0 {
		t.Fatal("payload is empty")
	}
}
