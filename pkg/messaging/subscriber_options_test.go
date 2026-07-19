package messaging

import (
	"context"
	"testing"
	"time"
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

func TestRetryDelayUsesExponentialCapAndDeterministicJitter(t *testing.T) {
	options := RetryBackoffOptions{BaseDelay: 30 * time.Second, MaxDelay: 5 * time.Minute, JitterFraction: .2}
	first := RetryDelay(options, 1, "event-1")
	capped := RetryDelay(options, 8, "event-1")
	if first < 24*time.Second || first > 36*time.Second {
		t.Fatalf("first delay = %s, want within 20%% jitter", first)
	}
	if capped < 4*time.Minute || capped > 6*time.Minute {
		t.Fatalf("capped delay = %s, want within 20%% jitter", capped)
	}
	if again := RetryDelay(options, 8, "event-1"); again != capped {
		t.Fatalf("delay is not deterministic: %s != %s", again, capped)
	}
}
