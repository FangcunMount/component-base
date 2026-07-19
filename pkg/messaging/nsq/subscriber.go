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
	consumers []*nsq.Consumer
	config    *nsq.Config
	lookupd   []string
	options   messaging.SubscriberOptions
	mu        sync.RWMutex
	stopped   bool
}

// NewSubscriber 创建 NSQ 订阅者
// lookupdAddrs: NSQLookupd 地址列表
// cfg: NSQ 配置
func NewSubscriber(lookupdAddrs []string, cfg *nsq.Config) (messaging.Subscriber, error) {
	return NewSubscriberWithOptions(lookupdAddrs, cfg, messaging.SubscriberOptions{})
}

// NewSubscriberWithOptions is the additive bounded-delivery constructor.
// A zero option preserves the historical cfg behavior.
func NewSubscriberWithOptions(lookupdAddrs []string, cfg *nsq.Config, opts messaging.SubscriberOptions) (messaging.Subscriber, error) {
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
		consumers: make([]*nsq.Consumer, 0),
		config:    cfg,
		lookupd:   lookupdAddrs,
		options:   opts,
		stopped:   false,
	}, nil
}

// Subscribe 订阅主题
func (s *subscriber) Subscribe(topic, channel string, handler messaging.Handler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return fmt.Errorf("subscriber is stopped")
	}

	// 创建 consumer
	consumer, err := nsq.NewConsumer(topic, channel, s.config)
	if err != nil {
		return fmt.Errorf("failed to create NSQ consumer: %w", err)
	}

	// 并发度与 MaxInFlight 保持一致，确保可以并行处理消息
	concurrency := s.config.MaxInFlight
	if concurrency < 1 {
		concurrency = 1
	}

	// 添加消息处理器（并发）
	consumer.AddConcurrentHandlers(nsq.HandlerFunc(func(message *nsq.Message) error {
		// 将 NSQ 的 Message 转换为领域层的 Message
		domainMsg, ok, err := messaging.DecodeMessagePayload(message.Body)
		if err != nil {
			domainMsg = &messaging.Message{UUID: string(message.ID[:]), Payload: message.Body, Metadata: make(map[string]string)}
			return s.failDelivery(context.Background(), topic, channel, message, domainMsg, fmt.Errorf("failed to decode message envelope: %w", err))
		}
		if !ok {
			domainMsg = &messaging.Message{
				UUID:     string(message.ID[:]),
				Payload:  message.Body,
				Metadata: make(map[string]string),
			}
		}
		if domainMsg.UUID == "" {
			domainMsg.UUID = string(message.ID[:])
		}
		if domainMsg.Metadata == nil {
			domainMsg.Metadata = make(map[string]string)
		}
		domainMsg.Attempts = message.Attempts
		domainMsg.Timestamp = message.Timestamp
		domainMsg.Topic = topic
		domainMsg.Channel = channel
		if s.options.MaxAttempts > 0 && int(message.Attempts) > s.options.MaxAttempts {
			return s.failDelivery(context.Background(), topic, channel, message, domainMsg, errors.New("transport delivery exhausted"))
		}

		// 注入 Ack/Nack 函数
		var handlerErr error
		domainMsg.SetAckFunc(func() error {
			message.Finish()
			return nil
		})
		domainMsg.SetNackFunc(func() error {
			if handlerErr == nil {
				handlerErr = errors.New("message nacked by handler")
			}
			return s.failDelivery(context.Background(), topic, channel, message, domainMsg, handlerErr)
		})

		// 创建 context（可以注入 trace、timeout 等）
		ctx := context.Background()

		// 调用业务层的 handler
		if err := handler(ctx, domainMsg); err != nil {
			handlerErr = err
			// 如果处理失败，自动 Nack（重新入队）
			if !domainMsg.IsSettled() {
				return domainMsg.Nack()
			}
			return nil
		}

		// 处理成功，自动 Ack
		if !domainMsg.IsSettled() {
			domainMsg.Ack()
		}
		return nil
	}), concurrency)

	// 连接到 NSQLookupd
	if err := consumer.ConnectToNSQLookupds(s.lookupd); err != nil {
		consumer.Stop()
		return fmt.Errorf("failed to connect to lookupd: %w", err)
	}

	// 保存 consumer 引用
	s.consumers = append(s.consumers, consumer)

	return nil
}

func (s *subscriber) failDelivery(ctx context.Context, topic, channel string, raw *nsq.Message, message *messaging.Message, cause error) error {
	if s.options.MaxAttempts <= 0 {
		raw.Requeue(-1)
		return nil
	}
	if int(raw.Attempts) < s.options.MaxAttempts {
		raw.Requeue(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return nil
	}
	if s.options.FailedMessageHandler == nil {
		raw.Requeue(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return errors.New("failed-message handler is not configured")
	}
	failed := messaging.FailedMessage{Provider: "nsq", Topic: topic, Channel: channel, Message: message, Attempts: s.options.MaxAttempts, Cause: cause}
	if err := s.options.FailedMessageHandler(ctx, failed); err != nil {
		raw.Requeue(messaging.RetryDelay(s.options.RetryBackoff, int(raw.Attempts), message.UUID))
		return fmt.Errorf("failed to persist exhausted NSQ message: %w", err)
	}
	raw.Finish()
	return nil
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
