//go:build integration

package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestSubscriberBoundedDeliveryAndDLQWithRealRabbitMQ(t *testing.T) {
	if os.Getenv("MESSAGING_INTEGRATION") != "1" {
		t.Skip("set MESSAGING_INTEGRATION=1 and start the messaging test containers")
	}
	url := rabbitEnvOr("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/")
	topic := fmt.Sprintf("component-base-rabbit-%d", time.Now().UnixNano())
	channel := topic + "-channel"
	var handlerCalls atomic.Int32
	var failedCalls atomic.Int32
	failedCh := make(chan messaging.FailedMessage, 1)
	subscriber, err := NewSubscriberWithOptions(url, messaging.SubscriberOptions{
		MaxInFlight: 1, MaxAttempts: 8,
		RetryBackoff: messaging.RetryBackoffOptions{BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
		FailedMessageHandler: func(_ context.Context, failed messaging.FailedMessage) error {
			if failedCalls.Add(1) == 1 {
				return errors.New("audit store unavailable")
			}
			failedCh <- failed
			return nil
		},
	})
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
	publisher, err := NewPublisher(url)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	message := messaging.NewMessage("rabbit-message-1", []byte("payload"))
	if err := publisher.PublishMessage(t.Context(), topic, message); err != nil {
		t.Fatal(err)
	}
	select {
	case failed := <-failedCh:
		if failed.Attempts != 8 || failed.Message == nil || failed.Message.UUID != message.UUID || string(failed.Message.Payload) != "payload" || failed.Cause == nil {
			t.Fatalf("failed message = %#v", failed)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for exhausted RabbitMQ delivery")
	}
	if got := handlerCalls.Load(); got != 8 {
		t.Fatalf("handler calls = %d, want 8", got)
	}
	if got := failedCalls.Load(); got < 2 {
		t.Fatalf("failed handler calls = %d, want at least 2", got)
	}
	assertRabbitDLQ(t, url, channel, message.UUID)
}

func assertRabbitDLQ(t *testing.T, url, channel, messageID string) {
	t.Helper()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for RabbitMQ DLQ message")
		case <-ticker.C:
			delivery, ok, getErr := ch.Get(channel+".dlq", false)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if !ok {
				continue
			}
			_ = delivery.Ack(false)
			if delivery.MessageId != messageID || rabbitAttempt(delivery.Headers) != 8 {
				t.Fatalf("DLQ message id/attempt = %q/%d", delivery.MessageId, rabbitAttempt(delivery.Headers))
			}
			return
		}
	}
}

func rabbitEnvOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
