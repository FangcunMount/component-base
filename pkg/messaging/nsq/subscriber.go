package nsq

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/FangcunMount/component-base/pkg/messaging"
	"github.com/nsqio/go-nsq"
)

// subscriber NSQ 订阅者实现
type subscriber struct {
	consumers        []*nsq.Consumer
	config           *nsq.Config
	lookupd          []string
	options          messaging.SubscriberOptions
	mu               sync.RWMutex
	handoffMu        sync.Mutex
	producerMu       sync.Mutex
	handoffConsumers map[string]*nsq.Consumer
	producers        map[string]nsqProducer
	terminalCauses   map[string]error
	newProducer      nsqProducerFactory
	connectHandoff   func(*nsq.Consumer, string) error
	stopped          bool
}

type nsqProducer interface {
	Publish(string, []byte) error
	Stop()
}

type nsqProducerFactory func(string, *nsq.Config) (nsqProducer, error)

// NewSubscriber 创建 NSQ 订阅者
// lookupdAddrs: NSQLookupd 地址列表
// cfg: NSQ 配置
func NewSubscriber(lookupdAddrs []string, cfg *nsq.Config) (messaging.Subscriber, error) {
	return NewSubscriberWithOptions(lookupdAddrs, cfg, messaging.SubscriberOptions{})
}

// NewSubscriberWithOptions is the additive bounded-delivery constructor.
// A zero option preserves the historical cfg behavior.
func NewSubscriberWithOptions(lookupdAddrs []string, cfg *nsq.Config, opts messaging.SubscriberOptions) (messaging.Subscriber, error) {
	if opts.MaxAttempts < 0 {
		return nil, fmt.Errorf("max attempts cannot be negative")
	}
	if opts.MaxAttempts > 0 && opts.FailedMessageHandler == nil {
		return nil, fmt.Errorf("failed-message handler is required when max attempts is configured")
	}
	if cfg == nil {
		cfg = nsq.NewConfig()
	} else {
		copy := *cfg
		cfg = &copy
	}
	if opts.MaxInFlight > 0 {
		cfg.MaxInFlight = opts.MaxInFlight
	}
	if opts.MaxAttempts > 0 {
		// The adapter owns the terminal handoff so a failed dead-letter write can
		// be retried without invoking the business handler again.
		cfg.MaxAttempts = 0
	}

	if len(lookupdAddrs) == 0 {
		return nil, fmt.Errorf("lookupd addresses cannot be empty")
	}

	return &subscriber{
		consumers:        make([]*nsq.Consumer, 0),
		config:           cfg,
		lookupd:          lookupdAddrs,
		options:          opts,
		handoffConsumers: make(map[string]*nsq.Consumer),
		producers:        make(map[string]nsqProducer),
		terminalCauses:   make(map[string]error),
		newProducer: func(address string, config *nsq.Config) (nsqProducer, error) {
			return nsq.NewProducer(address, config)
		},
		connectHandoff: func(consumer *nsq.Consumer, address string) error {
			return consumer.ConnectToNSQD(address)
		},
		stopped: false,
	}, nil
}

// Subscribe 订阅主题
func (s *subscriber) Subscribe(topic, channel string, handler messaging.Handler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return fmt.Errorf("subscriber is stopped")
	}

	consumer, err := s.newConsumer(topic, channel, handler)
	if err != nil {
		return err
	}
	consumers := []*nsq.Consumer{consumer}
	if s.options.MaxAttempts > 0 {
		handoff, handoffErr := s.newHandoffConsumer(topic, channel)
		if handoffErr != nil {
			consumer.Stop()
			return handoffErr
		}
		consumers = append(consumers, handoff)
	}
	for _, current := range consumers {
		if err := current.ConnectToNSQLookupds(s.lookupd); err != nil {
			for _, cleanup := range consumers {
				cleanup.Stop()
			}
			return fmt.Errorf("failed to connect to lookupd: %w", err)
		}
	}
	if len(consumers) > 1 {
		s.handoffMu.Lock()
		s.handoffConsumers[failedHandoffTopic(topic, channel)] = consumers[1]
		s.handoffMu.Unlock()
	}
	s.consumers = append(s.consumers, consumers...)
	return nil
}

func (s *subscriber) newConsumer(topic, channel string, handler messaging.Handler) (*nsq.Consumer, error) {
	consumer, err := nsq.NewConsumer(topic, channel, s.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ consumer: %w", err)
	}
	consumer.AddConcurrentHandlers(nsq.HandlerFunc(func(message *nsq.Message) error {
		return s.handleMessage(context.Background(), topic, channel, message, handler)
	}), max(s.config.MaxInFlight, 1))
	return consumer, nil
}

func (s *subscriber) handleMessage(ctx context.Context, topic, channel string, raw *nsq.Message, handler messaging.Handler) error {
	domainMsg, ok, err := messaging.DecodeMessagePayload(raw.Body)
	if err != nil {
		if domainMsg == nil {
			domainMsg = &messaging.Message{Payload: append([]byte(nil), raw.Body...)}
		}
		s.prepareMessage(domainMsg, topic, channel, raw)
		return s.failDelivery(ctx, topic, channel, raw, domainMsg, fmt.Errorf("failed to decode message envelope: %w", err))
	}
	if !ok {
		domainMsg = &messaging.Message{UUID: string(raw.ID[:]), Payload: raw.Body, Metadata: make(map[string]string)}
	}
	s.prepareMessage(domainMsg, topic, channel, raw)
	if s.options.MaxAttempts > 0 && int(raw.Attempts) > s.options.MaxAttempts {
		return s.failDelivery(ctx, topic, channel, raw, domainMsg, s.terminalCause(raw))
	}

	var handlerErr error
	domainMsg.SetAckFunc(func() error {
		raw.Finish()
		return nil
	})
	domainMsg.SetNackFunc(func() error {
		if handlerErr == nil {
			handlerErr = errors.New("message nacked by handler")
		}
		return s.failDelivery(ctx, topic, channel, raw, domainMsg, handlerErr)
	})
	if err := handler(ctx, domainMsg); err != nil {
		handlerErr = err
		if !domainMsg.IsSettled() {
			return domainMsg.Nack()
		}
		return nil
	}
	if !domainMsg.IsSettled() {
		return domainMsg.Ack()
	}
	return nil
}

func (s *subscriber) prepareMessage(domainMsg *messaging.Message, topic, channel string, raw *nsq.Message) {
	if domainMsg.UUID == "" {
		domainMsg.UUID = string(raw.ID[:])
	}
	if domainMsg.Metadata == nil {
		domainMsg.Metadata = make(map[string]string)
	}
	domainMsg.Attempts = raw.Attempts
	domainMsg.Timestamp = raw.Timestamp
	domainMsg.Topic = topic
	domainMsg.Channel = channel
}

func (s *subscriber) newHandoffConsumer(topic, channel string) (*nsq.Consumer, error) {
	handoffTopic := failedHandoffTopic(topic, channel)
	consumer, err := nsq.NewConsumer(handoffTopic, failedHandoffChannel, s.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ handoff consumer: %w", err)
	}
	consumer.AddConcurrentHandlers(nsq.HandlerFunc(func(raw *nsq.Message) error {
		return s.handleHandoff(context.Background(), handoffTopic, raw)
	}), max(s.config.MaxInFlight, 1))
	return consumer, nil
}

func (s *subscriber) handleHandoff(ctx context.Context, handoffTopic string, raw *nsq.Message) error {
	failed, decodeErr := decodeFailedHandoff(raw.Body)
	if decodeErr != nil {
		failed = messaging.FailedMessage{
			Provider: "nsq", Topic: handoffTopic, Channel: failedHandoffChannel, Attempts: s.options.MaxAttempts,
			Message: &messaging.Message{UUID: string(raw.ID[:]), Payload: append([]byte(nil), raw.Body...), Attempts: raw.Attempts, Timestamp: raw.Timestamp, Topic: handoffTopic, Channel: failedHandoffChannel},
			Cause:   decodeErr,
		}
	}
	if err := s.options.FailedMessageHandler(ctx, failed); err != nil {
		messageID := failed.Message.UUID
		raw.RequeueWithoutBackoff(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), messageID))
		return nil
	}
	raw.Finish()
	return nil
}

func (s *subscriber) failDelivery(ctx context.Context, topic, channel string, raw *nsq.Message, message *messaging.Message, cause error) error {
	if s.options.MaxAttempts <= 0 {
		raw.Requeue(-1)
		return nil
	}
	if int(raw.Attempts) < s.options.MaxAttempts {
		raw.RequeueWithoutBackoff(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return nil
	}
	if cause == nil {
		cause = errors.New("transport delivery exhausted")
	}
	s.rememberTerminalCause(raw, cause)
	payload, err := encodeFailedHandoff(topic, channel, message, s.options.MaxAttempts, cause)
	if err != nil {
		raw.RequeueWithoutBackoff(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return err
	}
	if err := s.ensureHandoffReady(topic, channel, raw.NSQDAddress); err != nil {
		raw.RequeueWithoutBackoff(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return err
	}
	producer, err := s.producerFor(raw.NSQDAddress)
	if err != nil {
		raw.RequeueWithoutBackoff(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return fmt.Errorf("create NSQ handoff producer: %w", err)
	}
	if publishErr := producer.Publish(failedHandoffTopic(topic, channel), payload); publishErr != nil {
		raw.RequeueWithoutBackoff(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return fmt.Errorf("publish NSQ failed-message handoff: %w", publishErr)
	}
	s.forgetTerminalCause(raw)
	raw.Finish()
	return nil
}

func (s *subscriber) ensureHandoffReady(topic, channel, address string) error {
	if address == "" {
		return fmt.Errorf("NSQD address is unavailable for failed-message handoff")
	}
	handoffTopic := failedHandoffTopic(topic, channel)
	s.handoffMu.Lock()
	defer s.handoffMu.Unlock()
	consumer, exists := s.handoffConsumers[handoffTopic]
	if !exists {
		return fmt.Errorf("NSQ handoff consumer %s is not configured", handoffTopic)
	}
	if err := s.connectHandoff(consumer, address); err != nil && !errors.Is(err, nsq.ErrAlreadyConnected) {
		return fmt.Errorf("connect NSQ handoff consumer to %s: %w", address, err)
	}
	return nil
}

func (s *subscriber) producerFor(address string) (nsqProducer, error) {
	if address == "" {
		return nil, fmt.Errorf("NSQD address is unavailable for failed-message handoff")
	}
	s.producerMu.Lock()
	defer s.producerMu.Unlock()
	if producer := s.producers[address]; producer != nil {
		return producer, nil
	}
	config := *s.config
	producer, err := s.newProducer(address, &config)
	if err != nil {
		return nil, err
	}
	s.producers[address] = producer
	return producer, nil
}

func (s *subscriber) rememberTerminalCause(raw *nsq.Message, cause error) {
	s.producerMu.Lock()
	s.terminalCauses[string(raw.ID[:])] = cause
	s.producerMu.Unlock()
}

func (s *subscriber) terminalCause(raw *nsq.Message) error {
	s.producerMu.Lock()
	defer s.producerMu.Unlock()
	if cause := s.terminalCauses[string(raw.ID[:])]; cause != nil {
		return cause
	}
	return errors.New("transport delivery exhausted after prior handler failure")
}

func (s *subscriber) forgetTerminalCause(raw *nsq.Message) {
	s.producerMu.Lock()
	delete(s.terminalCauses, string(raw.ID[:]))
	s.producerMu.Unlock()
}

// SubscribeWithMiddleware 订阅消息（支持中间件）
func (s *subscriber) SubscribeWithMiddleware(topic, channel string, handler messaging.Handler, middlewares ...messaging.Middleware) error {
	// 应用中间件
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	// 调用标准 Subscribe
	return s.Subscribe(topic, channel, handler)
}

// Stop 停止所有订阅
func (s *subscriber) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.stopped = true

	for _, consumer := range s.consumers {
		consumer.Stop()
	}
	s.producerMu.Lock()
	for address, producer := range s.producers {
		producer.Stop()
		delete(s.producers, address)
	}
	s.producerMu.Unlock()
}

// Close 关闭订阅者
func (s *subscriber) Close() error {
	s.Stop()

	// 等待所有 consumer 停止
	for _, consumer := range s.consumers {
		<-consumer.StopChan
	}

	return nil
}

// Stats 获取订阅者统计信息
func (s *subscriber) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	for i, consumer := range s.consumers {
		consumerStats := consumer.Stats()
		stats[fmt.Sprintf("consumer_%d", i)] = map[string]interface{}{
			"messages": consumerStats.MessagesReceived,
			"finished": consumerStats.MessagesFinished,
			"requeued": consumerStats.MessagesRequeued,
		}
	}
	return stats
}
