//go:build integration

package nsq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/messaging"
)

func TestSubscriberBoundedDeliveryWithRealNSQ(t *testing.T) {
	if os.Getenv("MESSAGING_INTEGRATION") != "1" {
		t.Skip("set MESSAGING_INTEGRATION=1 and start the messaging test containers")
	}
	lookupd := envOr("NSQ_LOOKUPD_ADDR", "127.0.0.1:4161")
	nsqd := envOr("NSQD_ADDR", "127.0.0.1:4150")
	topic := fmt.Sprintf("component-base-nsq-%d", time.Now().UnixNano())
	channel := topic + "-channel"

	var handlerCalls atomic.Int32
	var failedCalls atomic.Int32
	failedCh := make(chan messaging.FailedMessage, 1)
	options := messaging.SubscriberOptions{
		MaxInFlight: 1, MaxAttempts: 8,
		RetryBackoff: messaging.RetryBackoffOptions{BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
		FailedMessageHandler: func(_ context.Context, failed messaging.FailedMessage) error {
			if failedCalls.Add(1) == 1 {
				return errors.New("audit store unavailable")
			}
			failedCh <- failed
			return nil
		},
	}
	subscriber, err := NewSubscriberWithOptions([]string{lookupd}, nil, options)
	if err != nil {
		t.Fatal(err)
	}
	defer subscriber.Close()
	if err := subscriber.Subscribe(topic, channel, func(context.Context, *messaging.Message) error {
		handlerCalls.Add(1)
		return errors.New("handler failed")
	}); err != nil {
		t.Fatal(err)
	}
	publisher, err := NewPublisher(nsqd, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	message := messaging.NewMessage("nsq-message-1", []byte("payload"))
	if err := publisher.PublishMessage(t.Context(), topic, message); err != nil {
		t.Fatal(err)
	}

	select {
	case failed := <-failedCh:
		if failed.Attempts != 8 || failed.Message == nil || failed.Message.UUID != message.UUID || string(failed.Message.Payload) != "payload" || failed.Cause == nil {
			t.Fatalf("failed message = %#v", failed)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for exhausted NSQ delivery")
	}
	if got := handlerCalls.Load(); got != 8 {
		t.Fatalf("handler calls = %d, want 8", got)
	}
	if got := failedCalls.Load(); got < 2 {
		t.Fatalf("failed handler calls = %d, want at least 2", got)
	}
}

func TestSubscriberDecodeFailureNeverCallsBusinessHandler(t *testing.T) {
	if os.Getenv("MESSAGING_INTEGRATION") != "1" {
		t.Skip("set MESSAGING_INTEGRATION=1 and start the messaging test containers")
	}
	lookupd := envOr("NSQ_LOOKUPD_ADDR", "127.0.0.1:4161")
	nsqd := envOr("NSQD_ADDR", "127.0.0.1:4150")
	topic := fmt.Sprintf("component-base-nsq-decode-%d", time.Now().UnixNano())
	channel := topic + "-channel"
	var handlerCalls atomic.Int32
	failedCh := make(chan messaging.FailedMessage, 1)
	subscriber, err := NewSubscriberWithOptions([]string{lookupd}, nil, messaging.SubscriberOptions{
		MaxInFlight: 1, MaxAttempts: 8,
		RetryBackoff:         messaging.RetryBackoffOptions{BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
		FailedMessageHandler: func(_ context.Context, failed messaging.FailedMessage) error { failedCh <- failed; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer subscriber.Close()
	if err := subscriber.Subscribe(topic, channel, func(context.Context, *messaging.Message) error {
		handlerCalls.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	publisher, err := NewPublisher(nsqd, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	if err := publisher.Publish(t.Context(), topic, []byte("not-a-component-message")); err != nil {
		t.Fatal(err)
	}
	select {
	case failed := <-failedCh:
		if failed.Attempts != 8 || failed.Cause == nil {
			t.Fatalf("failed message = %#v", failed)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for exhausted decode failure")
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("handler calls = %d, want 0", got)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
