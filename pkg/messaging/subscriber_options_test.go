package messaging

import (
	"context"
	"testing"
)

func TestSubscriberOptionsFailedMessageContractRetainsOriginalMessage(t *testing.T) {
	message := NewMessage("event-1", []byte("payload"))
	called := false
	options := SubscriberOptions{MaxInFlight: 4, MaxAttempts: 8, FailedMessageHandler: func(_ context.Context, failed FailedMessage) error {
		called = failed.Message == message && failed.Attempts == 8 && failed.Provider == "nsq"
		return nil
	}}
	if options.MaxInFlight != 4 || options.MaxAttempts != 8 {
		t.Fatalf("options = %#v", options)
	}
	if err := options.FailedMessageHandler(t.Context(), FailedMessage{Provider: "nsq", Message: message, Attempts: 8}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("failed-message handler did not retain delivery identity")
	}
}
