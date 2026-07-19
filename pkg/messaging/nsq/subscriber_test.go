package nsq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/messaging"
	gonq "github.com/nsqio/go-nsq"
)

func TestNewSubscriberWithOptionsRequiresFailedMessageHandler(t *testing.T) {
	_, err := NewSubscriberWithOptions([]string{"127.0.0.1:4161"}, nil, messaging.SubscriberOptions{MaxAttempts: 8})
	if err == nil {
		t.Fatal("bounded subscriber accepted a nil failed-message handler")
	}
	if _, err := NewSubscriber([]string{"127.0.0.1:4161"}, nil); err != nil {
		t.Fatalf("legacy zero-option subscriber: %v", err)
	}
}

func TestNewSubscriberWithOptionsRejectsNegativeMaxAttempts(t *testing.T) {
	_, err := NewSubscriberWithOptions([]string{"127.0.0.1:4161"}, nil, messaging.SubscriberOptions{MaxAttempts: -1})
	if err == nil {
		t.Fatal("subscriber accepted negative max attempts")
	}
}

func TestFailedHandoffRoundTripPreservesOriginalFailure(t *testing.T) {
	message := messaging.NewMessage("message-1", []byte("payload"))
	message.Metadata["event_type"] = "sample.created"
	message.Timestamp = 123
	payload, err := encodeFailedHandoff("topic", "channel", message, 8, errors.New("handler failed"))
	if err != nil {
		t.Fatal(err)
	}
	failed, err := decodeFailedHandoff(payload)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Provider != "nsq" || failed.Topic != "topic" || failed.Channel != "channel" || failed.Attempts != 8 || failed.Cause.Error() != "handler failed" {
		t.Fatalf("failed handoff = %#v", failed)
	}
	if failed.Message.UUID != message.UUID || string(failed.Message.Payload) != "payload" || failed.Message.Metadata["event_type"] != "sample.created" || failed.Message.Timestamp != 123 {
		t.Fatalf("handoff message = %#v", failed.Message)
	}
}

func TestFailedHandoffTopicIsDeterministicAndValid(t *testing.T) {
	first := failedHandoffTopic("a-very-long-topic-name-that-nearly-reaches-the-provider-limit", "channel")
	second := failedHandoffTopic("a-very-long-topic-name-that-nearly-reaches-the-provider-limit", "channel")
	if first != second || !gonq.IsValidTopicName(first) || len(first) > 64 {
		t.Fatalf("handoff topic = %q/%q", first, second)
	}
	if first == failedHandoffTopic("a-very-long-topic-name-that-nearly-reaches-the-provider-limit", "other-channel") {
		t.Fatal("handoff topic does not include channel identity")
	}
}

func TestTerminalFailurePublishesHandoffBeforeFinishing(t *testing.T) {
	producer := &fakeNSQProducer{}
	s := newTestSubscriber(t, func(string, *gonq.Config) (nsqProducer, error) { return producer, nil })
	raw, delegate := newRawMessage(8, "nsqd:4150", []byte("wire"))
	message := messaging.NewMessage("message-1", []byte("payload"))

	if err := s.failDelivery(context.Background(), "topic", "channel", raw, message, errors.New("handler failed")); err != nil {
		t.Fatal(err)
	}
	if delegate.finishes != 1 || delegate.requeues != 0 {
		t.Fatalf("settlement finish/requeue = %d/%d", delegate.finishes, delegate.requeues)
	}
	if producer.topic != failedHandoffTopic("topic", "channel") {
		t.Fatalf("published topic = %q", producer.topic)
	}
	failed, err := decodeFailedHandoff(producer.payload)
	if err != nil || failed.Cause.Error() != "handler failed" || failed.Message.UUID != "message-1" {
		t.Fatalf("published handoff = %#v, %v", failed, err)
	}
}

func TestRecognizedCorruptEnvelopeNeverCallsBusinessHandler(t *testing.T) {
	s := newTestSubscriber(t, nil)
	encoded, err := messaging.EncodeMessagePayload(messaging.NewMessage("message-1", []byte("payload")))
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
	raw, delegate := newRawMessage(1, "nsqd:4150", corrupt)
	handlerCalls := 0
	if err := s.handleMessage(context.Background(), "topic", "channel", raw, func(context.Context, *messaging.Message) error {
		handlerCalls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if handlerCalls != 0 || delegate.requeues != 1 || delegate.finishes != 0 {
		t.Fatalf("handler/requeue/finish = %d/%d/%d", handlerCalls, delegate.requeues, delegate.finishes)
	}
}

func TestLegacyRawPayloadStillReachesBusinessHandler(t *testing.T) {
	s := newTestSubscriber(t, nil)
	raw, delegate := newRawMessage(1, "nsqd:4150", []byte("legacy raw payload"))
	handlerCalls := 0
	if err := s.handleMessage(context.Background(), "topic", "channel", raw, func(_ context.Context, message *messaging.Message) error {
		handlerCalls++
		if string(message.Payload) != "legacy raw payload" {
			t.Fatalf("payload = %q", message.Payload)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if handlerCalls != 1 || delegate.finishes != 1 || delegate.requeues != 0 {
		t.Fatalf("handler/finish/requeue = %d/%d/%d", handlerCalls, delegate.finishes, delegate.requeues)
	}
}

func TestBusinessHandlerRunsAtMostConfiguredAttempts(t *testing.T) {
	producer := &fakeNSQProducer{}
	s := newTestSubscriber(t, func(string, *gonq.Config) (nsqProducer, error) { return producer, nil })
	handlerCalls := 0
	for attempt := 1; attempt <= 8; attempt++ {
		raw, delegate := newRawMessage(uint16(attempt), "nsqd:4150", []byte("legacy raw payload"))
		if err := s.handleMessage(context.Background(), "topic", "channel", raw, func(context.Context, *messaging.Message) error {
			handlerCalls++
			return errors.New("handler failed")
		}); err != nil {
			t.Fatal(err)
		}
		if attempt < 8 && delegate.requeues != 1 {
			t.Fatalf("attempt %d requeues = %d", attempt, delegate.requeues)
		}
		if attempt == 8 && delegate.finishes != 1 {
			t.Fatalf("terminal attempt finishes = %d", delegate.finishes)
		}
	}
	if handlerCalls != 8 {
		t.Fatalf("handler calls = %d", handlerCalls)
	}
	ninth, ninthDelegate := newRawMessage(9, "nsqd:4150", []byte("legacy raw payload"))
	if err := s.handleMessage(context.Background(), "topic", "channel", ninth, func(context.Context, *messaging.Message) error {
		handlerCalls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if handlerCalls != 8 || ninthDelegate.finishes != 1 {
		t.Fatalf("post-budget handler/finish = %d/%d", handlerCalls, ninthDelegate.finishes)
	}
}

func TestHandoffRetryNeverReentersBusinessHandlerAndKeepsCause(t *testing.T) {
	var calls int
	var received []messaging.FailedMessage
	s := newTestSubscriber(t, nil)
	s.options.RetryBackoff = messaging.RetryBackoffOptions{BaseDelay: time.Second, MaxDelay: time.Minute}
	s.options.FailedMessageHandler = func(_ context.Context, failed messaging.FailedMessage) error {
		calls++
		received = append(received, failed)
		if calls == 1 {
			return errors.New("audit unavailable")
		}
		return nil
	}
	payload, err := encodeFailedHandoff("topic", "channel", messaging.NewMessage("message-1", []byte("payload")), 8, errors.New("handler failed"))
	if err != nil {
		t.Fatal(err)
	}
	first, firstDelegate := newRawMessage(1, "nsqd:4150", payload)
	if err := s.handleHandoff(context.Background(), failedHandoffTopic("topic", "channel"), first); err != nil {
		t.Fatal(err)
	}
	if firstDelegate.requeues != 1 || firstDelegate.backoff {
		t.Fatalf("first handoff settlement = %#v", firstDelegate)
	}
	second, secondDelegate := newRawMessage(2, "nsqd:4150", payload)
	if err := s.handleHandoff(context.Background(), failedHandoffTopic("topic", "channel"), second); err != nil {
		t.Fatal(err)
	}
	if secondDelegate.finishes != 1 || calls != 2 {
		t.Fatalf("second handoff finish/calls = %d/%d", secondDelegate.finishes, calls)
	}
	for _, failed := range received {
		if failed.Attempts != 8 || failed.Cause.Error() != "handler failed" || failed.Message.UUID != "message-1" {
			t.Fatalf("failed-message contract drifted: %#v", failed)
		}
	}
}

func TestDuplicateHandoffDeliveriesPreserveTheSameFailure(t *testing.T) {
	payload, err := encodeFailedHandoff("topic", "channel", messaging.NewMessage("message-1", []byte("payload")), 8, errors.New("handler failed"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	received := make([]messaging.FailedMessage, 0, 2)
	s := newTestSubscriber(t, nil)
	s.options.FailedMessageHandler = func(_ context.Context, failed messaging.FailedMessage) error {
		mu.Lock()
		received = append(received, failed)
		mu.Unlock()
		return nil
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for attempt := 1; attempt <= 2; attempt++ {
		attempt := attempt
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			raw, _ := newRawMessage(uint16(attempt), "nsqd:4150", payload)
			if handleErr := s.handleHandoff(context.Background(), failedHandoffTopic("topic", "channel"), raw); handleErr != nil {
				t.Errorf("handle duplicate handoff: %v", handleErr)
			}
		}()
	}
	close(start)
	wg.Wait()
	if len(received) != 2 {
		t.Fatalf("failed-message calls = %d", len(received))
	}
	for _, failed := range received {
		if failed.Attempts != 8 || failed.Cause.Error() != "handler failed" || failed.Message.UUID != "message-1" {
			t.Fatalf("duplicate handoff contract drifted: %#v", failed)
		}
	}
}

func TestFailedMessageHandlerAndStopCanRaceSafely(t *testing.T) {
	payload, err := encodeFailedHandoff("topic", "channel", messaging.NewMessage("message-1", []byte("payload")), 8, errors.New("handler failed"))
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	s := newTestSubscriber(t, nil)
	s.options.FailedMessageHandler = func(context.Context, messaging.FailedMessage) error {
		close(entered)
		<-release
		return nil
	}
	raw, delegate := newRawMessage(1, "nsqd:4150", payload)
	handled := make(chan error, 1)
	go func() {
		handled <- s.handleHandoff(context.Background(), failedHandoffTopic("topic", "channel"), raw)
	}()
	<-entered
	stopped := make(chan struct{})
	go func() {
		s.Stop()
		close(stopped)
	}()
	<-stopped
	close(release)
	if err := <-handled; err != nil {
		t.Fatal(err)
	}
	if delegate.finishes != 1 {
		t.Fatalf("handoff finishes = %d", delegate.finishes)
	}
}

func TestTerminalPublishFailureNeverReentersBusinessHandler(t *testing.T) {
	producer := &fakeNSQProducer{err: errors.New("nsqd unavailable")}
	s := newTestSubscriber(t, func(string, *gonq.Config) (nsqProducer, error) { return producer, nil })
	handlerCalls := 0
	eighth, eighthDelegate := newRawMessage(8, "nsqd:4150", []byte("legacy raw payload"))
	if err := s.handleMessage(context.Background(), "topic", "channel", eighth, func(context.Context, *messaging.Message) error {
		handlerCalls++
		return errors.New("handler failed")
	}); err == nil || !errors.Is(err, producer.err) {
		t.Fatalf("terminal publish error = %v", err)
	}
	if handlerCalls != 1 || eighthDelegate.requeues != 1 || eighthDelegate.finishes != 0 {
		t.Fatalf("terminal publish failure handler/requeue/finish = %d/%d/%d", handlerCalls, eighthDelegate.requeues, eighthDelegate.finishes)
	}
	producer.err = nil
	ninth, ninthDelegate := newRawMessage(9, "nsqd:4150", []byte("legacy raw payload"))
	if err := s.handleMessage(context.Background(), "topic", "channel", ninth, func(context.Context, *messaging.Message) error {
		handlerCalls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if handlerCalls != 1 || ninthDelegate.finishes != 1 {
		t.Fatalf("post-budget handler/finish = %d/%d", handlerCalls, ninthDelegate.finishes)
	}
	failed, err := decodeFailedHandoff(producer.payload)
	if err != nil || failed.Cause.Error() != "handler failed" {
		t.Fatalf("terminal handoff cause = %#v, %v", failed, err)
	}
}

func TestProducerCacheIsSharedAndStoppedOnce(t *testing.T) {
	producer := &fakeNSQProducer{}
	created := 0
	s := newTestSubscriber(t, func(string, *gonq.Config) (nsqProducer, error) {
		created++
		return producer, nil
	})
	first, err := s.producerFor("nsqd:4150")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.producerFor("nsqd:4150")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || created != 1 {
		t.Fatalf("producer cache identity/creates = %v/%d", first == second, created)
	}
	s.Stop()
	s.Stop()
	if producer.stops != 1 {
		t.Fatalf("producer stops = %d", producer.stops)
	}
}

func TestConcurrentSubscriptionsShareNSQDProducer(t *testing.T) {
	producer := &fakeNSQProducer{}
	created := 0
	s := newTestSubscriber(t, func(string, *gonq.Config) (nsqProducer, error) {
		created++
		return producer, nil
	})
	const callers = 16
	results := make([]nsqProducer, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for index := range results {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[index], _ = s.producerFor("nsqd:4150")
		}()
	}
	close(start)
	wg.Wait()
	if created != 1 {
		t.Fatalf("producer creates = %d", created)
	}
	for _, result := range results {
		if result != producer {
			t.Fatal("producer cache returned different instances")
		}
	}
}

func newTestSubscriber(t *testing.T, factory nsqProducerFactory) *subscriber {
	t.Helper()
	base, err := NewSubscriberWithOptions([]string{"127.0.0.1:4161"}, nil, messaging.SubscriberOptions{
		MaxAttempts: 8, FailedMessageHandler: func(context.Context, messaging.FailedMessage) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	s := base.(*subscriber)
	s.handoffConsumers[failedHandoffTopic("topic", "channel")] = nil
	s.connectHandoff = func(*gonq.Consumer, string) error { return nil }
	if factory != nil {
		s.newProducer = factory
	}
	return s
}

type fakeNSQProducer struct {
	topic   string
	payload []byte
	err     error
	stops   int
}

func (p *fakeNSQProducer) Publish(topic string, payload []byte) error {
	p.topic = topic
	p.payload = append([]byte(nil), payload...)
	return p.err
}

func (p *fakeNSQProducer) Stop() { p.stops++ }

type fakeMessageDelegate struct {
	finishes int
	requeues int
	touches  int
	delay    time.Duration
	backoff  bool
}

func (d *fakeMessageDelegate) OnFinish(*gonq.Message) { d.finishes++ }
func (d *fakeMessageDelegate) OnRequeue(_ *gonq.Message, delay time.Duration, backoff bool) {
	d.requeues++
	d.delay = delay
	d.backoff = backoff
}
func (d *fakeMessageDelegate) OnTouch(*gonq.Message) { d.touches++ }

func newRawMessage(attempts uint16, address string, body []byte) (*gonq.Message, *fakeMessageDelegate) {
	id := gonq.MessageID{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', 'a', 'b', 'c', 'd', 'e', 'f'}
	raw := gonq.NewMessage(id, body)
	raw.Attempts = attempts
	raw.NSQDAddress = address
	delegate := &fakeMessageDelegate{}
	raw.Delegate = delegate
	return raw, delegate
}
