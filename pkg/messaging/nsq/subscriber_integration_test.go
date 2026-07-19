//go:build integration

package nsq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	cleanupNSQTopics(t, topic, failedHandoffTopic(topic, channel))
	createNSQTopicAndChannel(t, topic, channel)

	var handlerCalls atomic.Int32
	var failedCalls atomic.Int32
	firstAuditAttempt := make(chan struct{}, 1)
	failedCh := make(chan messaging.FailedMessage, 1)
	options := messaging.SubscriberOptions{
		MaxInFlight: 1, MaxAttempts: 8,
		RetryBackoff: messaging.RetryBackoffOptions{BaseDelay: 250 * time.Millisecond, MaxDelay: 250 * time.Millisecond},
		FailedMessageHandler: func(_ context.Context, failed messaging.FailedMessage) error {
			failedCalls.Add(1)
			select {
			case firstAuditAttempt <- struct{}{}:
			default:
			}
			return errors.New("audit store unavailable")
		},
	}
	firstSubscriber, err := NewSubscriberWithOptions([]string{lookupd}, nil, options)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstSubscriber.Subscribe(topic, channel, func(context.Context, *messaging.Message) error {
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
	case <-firstAuditAttempt:
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for first NSQ handoff persistence attempt")
	}
	if err := firstSubscriber.Close(); err != nil {
		t.Fatal(err)
	}

	secondOptions := options
	secondOptions.FailedMessageHandler = func(_ context.Context, failed messaging.FailedMessage) error {
		failedCalls.Add(1)
		failedCh <- failed
		return nil
	}
	secondSubscriber, err := NewSubscriberWithOptions([]string{lookupd}, nil, secondOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer secondSubscriber.Close()
	if err := secondSubscriber.Subscribe(topic, channel, func(context.Context, *messaging.Message) error {
		handlerCalls.Add(1)
		return errors.New("business handler must not receive terminal handoff")
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case failed := <-failedCh:
		if failed.Attempts != 8 || failed.Message == nil || failed.Message.UUID != message.UUID || string(failed.Message.Payload) != "payload" || failed.Cause == nil || failed.Cause.Error() != "handler failed" {
			t.Fatalf("failed message = %#v", failed)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for NSQ handoff after subscriber restart")
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
	cleanupNSQTopics(t, topic, failedHandoffTopic(topic, channel))
	createNSQTopicAndChannel(t, topic, channel)
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
	encoded, err := messaging.EncodeMessagePayload(messaging.NewMessage("corrupt-message-1", []byte("payload")))
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["checksum"] = "corrupt"
	corrupt, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(t.Context(), topic, corrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case failed := <-failedCh:
		if failed.Attempts != 8 || failed.Message == nil || failed.Message.UUID != "corrupt-message-1" || string(failed.Message.Payload) != "payload" || failed.Cause == nil || !strings.Contains(failed.Cause.Error(), "checksum mismatch") {
			t.Fatalf("failed message = %#v", failed)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for exhausted decode failure")
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("handler calls = %d, want 0", got)
	}
}

func cleanupNSQTopics(t *testing.T, topics ...string) {
	t.Helper()
	httpAddress := envOr("NSQD_HTTP_ADDR", "127.0.0.1:4151")
	t.Cleanup(func() {
		for _, topic := range topics {
			response, err := postNSQAdmin(httpAddress, "/topic/delete?topic="+url.QueryEscape(topic))
			if err != nil {
				t.Errorf("delete NSQ topic %q: %v", topic, err)
				continue
			}
			_ = response.Body.Close()
			if response.StatusCode != http.StatusNotFound && (response.StatusCode < 200 || response.StatusCode >= 300) {
				t.Errorf("delete NSQ topic %q: status %s", topic, response.Status)
			}
		}
	})
}

func createNSQTopicAndChannel(t *testing.T, topic, channel string) {
	t.Helper()
	httpAddress := envOr("NSQD_HTTP_ADDR", "127.0.0.1:4151")
	for _, endpoint := range []string{
		"/topic/create?topic=" + url.QueryEscape(topic),
		"/channel/create?topic=" + url.QueryEscape(topic) + "&channel=" + url.QueryEscape(channel),
	} {
		response, err := postNSQAdmin(httpAddress, endpoint)
		if err != nil {
			t.Fatalf("prepare NSQ topic/channel: %v", err)
		}
		_ = response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			t.Fatalf("prepare NSQ topic/channel: status %s", response.Status)
		}
	}
}

func postNSQAdmin(address, path string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+address+path, nil)
	if err != nil {
		return nil, err
	}
	return (&http.Client{Timeout: 5 * time.Second}).Do(request)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
