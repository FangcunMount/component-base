package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/FangcunMount/component-base/pkg/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitDeliveryAttemptHeader = "x-delivery-attempt"
	rabbitFailedOnlyHeader      = "x-failed-message-only"
	rabbitLastErrorHeader       = "x-last-delivery-error"
)

type subscriber struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	consumers map[string]*consumer
	stopCh    chan struct{}
	options   messaging.SubscriberOptions
	mu        sync.Mutex
	publishMu sync.Mutex
}

type consumer struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewSubscriber(url string) (messaging.Subscriber, error) {
	return NewSubscriberWithOptions(url, messaging.SubscriberOptions{})
}

// NewSubscriberWithOptions enables bounded retry topology when MaxAttempts is
// positive. Zero options preserve the historical immediate-requeue behavior.
func NewSubscriberWithOptions(url string, opts messaging.SubscriberOptions) (messaging.Subscriber, error) {
	if opts.MaxAttempts < 0 {
		return nil, fmt.Errorf("max attempts cannot be negative")
	}
	if opts.MaxAttempts > 0 && opts.FailedMessageHandler == nil {
		return nil, fmt.Errorf("failed-message handler is required when max attempts is configured")
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("创建 channel 失败: %w", err)
	}
	prefetch := opts.MaxInFlight
	if prefetch <= 0 {
		prefetch = 200
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("设置 QoS 失败: %w", err)
	}
	return &subscriber{conn: conn, channel: ch, consumers: make(map[string]*consumer), stopCh: make(chan struct{}), options: opts}, nil
}

func (s *subscriber) Subscribe(topic, channel string, handler messaging.Handler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := topic + ":" + channel
	if _, exists := s.consumers[key]; exists {
		return fmt.Errorf("已经订阅了 %s", key)
	}
	if err := s.channel.ExchangeDeclare(topic, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明 exchange %s 失败: %w", topic, err)
	}
	if s.options.MaxAttempts > 0 {
		if err := s.declareRetryTopology(topic, channel); err != nil {
			return err
		}
	}
	queue, err := s.channel.QueueDeclare(channel, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("声明 queue %s 失败: %w", channel, err)
	}
	if err := s.channel.QueueBind(queue.Name, "", topic, false, nil); err != nil {
		return fmt.Errorf("绑定 queue %s 到 exchange %s 失败: %w", channel, topic, err)
	}
	deliveries, err := s.channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("开始消费 queue %s 失败: %w", channel, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	item := &consumer{cancel: cancel, done: make(chan struct{})}
	s.consumers[key] = item
	go s.consume(ctx, item.done, topic, channel, deliveries, handler)
	return nil
}

func (s *subscriber) SubscribeWithMiddleware(topic, channel string, handler messaging.Handler, middlewares ...messaging.Middleware) error {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return s.Subscribe(topic, channel, handler)
}

func (s *subscriber) declareRetryTopology(topic, channel string) error {
	if err := s.channel.ExchangeDeclare(topic+".retry", "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare retry exchange: %w", err)
	}
	if err := s.channel.ExchangeDeclare(topic+".dlx", "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}
	dlq := channel + ".dlq"
	if _, err := s.channel.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := s.channel.QueueBind(dlq, channel, topic+".dlx", false, nil); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}
	firstRetryAttempt := 2
	if s.options.MaxAttempts == 1 {
		firstRetryAttempt = 1
	}
	for attempt := firstRetryAttempt; attempt <= s.options.MaxAttempts; attempt++ {
		queue := fmt.Sprintf("%s.retry.%d", channel, attempt)
		args := amqp.Table{"x-dead-letter-exchange": topic}
		if _, err := s.channel.QueueDeclare(queue, true, false, false, false, args); err != nil {
			return fmt.Errorf("declare retry queue %d: %w", attempt, err)
		}
		if err := s.channel.QueueBind(queue, rabbitRetryRoutingKey(channel, attempt), topic+".retry", false, nil); err != nil {
			return fmt.Errorf("bind retry queue %d: %w", attempt, err)
		}
	}
	return nil
}

func (s *subscriber) consume(ctx context.Context, done chan struct{}, topic, channel string, deliveries <-chan amqp.Delivery, handler messaging.Handler) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			s.handleDelivery(ctx, topic, channel, delivery, handler)
		}
	}
}

func (s *subscriber) handleDelivery(ctx context.Context, topic, channel string, delivery amqp.Delivery, handler messaging.Handler) {
	attempt, failedOnly, metadataErr := rabbitDeliveryMetadata(delivery.Headers)
	effectiveAttempt := attempt
	if s.options.MaxAttempts > 0 && effectiveAttempt > s.options.MaxAttempts {
		effectiveAttempt = s.options.MaxAttempts
	}
	message := &messaging.Message{UUID: delivery.MessageId, Payload: delivery.Body, Metadata: map[string]string{}, Timestamp: delivery.Timestamp.UnixNano(), Topic: topic, Channel: channel, Attempts: uint16(effectiveAttempt)}
	if message.UUID == "" {
		message.UUID = strconv.FormatUint(delivery.DeliveryTag, 10)
	}
	for key, value := range delivery.Headers {
		message.Metadata[key] = fmt.Sprint(value)
	}
	if s.options.MaxAttempts > 0 && (metadataErr != nil || failedOnly || attempt > s.options.MaxAttempts) {
		cause := metadataErr
		if cause == nil {
			cause = errors.New(rabbitHeaderString(delivery.Headers, rabbitLastErrorHeader, "transport delivery exhausted"))
		}
		if attempt > s.options.MaxAttempts {
			attempt = s.options.MaxAttempts
		}
		_ = s.failDelivery(ctx, topic, channel, delivery, message, attempt, cause, true)
		return
	}
	var handlerErr error
	message.SetAckFunc(func() error { return delivery.Ack(false) })
	message.SetNackFunc(func() error {
		if handlerErr == nil {
			handlerErr = errors.New("message nacked by handler")
		}
		return s.failDelivery(ctx, topic, channel, delivery, message, attempt, handlerErr, false)
	})
	if err := handler(ctx, message); err != nil {
		handlerErr = err
		if !message.IsSettled() {
			_ = message.Nack()
		}
		return
	}
	if !message.IsSettled() {
		_ = message.Ack()
	}
}

func (s *subscriber) failDelivery(ctx context.Context, topic, channel string, delivery amqp.Delivery, message *messaging.Message, attempt int, cause error, failedOnly bool) error {
	if s.options.MaxAttempts <= 0 {
		return delivery.Nack(false, true)
	}
	if attempt < s.options.MaxAttempts {
		return s.publishRetry(ctx, topic, channel, delivery, message.UUID, attempt+1, cause, failedOnly)
	}
	failed := messaging.FailedMessage{Provider: "rabbitmq", Topic: topic, Channel: channel, Message: message, Attempts: s.options.MaxAttempts, Cause: cause}
	if s.options.FailedMessageHandler == nil {
		return s.publishRetry(ctx, topic, channel, delivery, message.UUID, s.options.MaxAttempts, cause, true)
	}
	if err := s.options.FailedMessageHandler(ctx, failed); err != nil {
		return s.publishRetry(ctx, topic, channel, delivery, message.UUID, s.options.MaxAttempts, cause, true)
	}
	publishing := cloneRabbitPublishing(delivery)
	publishing.Headers[rabbitDeliveryAttemptHeader] = int32(s.options.MaxAttempts)
	delete(publishing.Headers, rabbitFailedOnlyHeader)
	delete(publishing.Headers, "expiration")
	if err := s.publish(ctx, topic+".dlx", channel, publishing); err != nil {
		return s.publishRetry(ctx, topic, channel, delivery, message.UUID, s.options.MaxAttempts, cause, true)
	}
	return delivery.Ack(false)
}

func (s *subscriber) publishRetry(ctx context.Context, topic, channel string, delivery amqp.Delivery, messageID string, attempt int, cause error, failedOnly bool) error {
	publishing := cloneRabbitPublishing(delivery)
	publishing.Headers[rabbitDeliveryAttemptHeader] = int32(attempt)
	publishing.Headers[rabbitFailedOnlyHeader] = failedOnly
	publishing.Headers[rabbitLastErrorHeader] = cause.Error()
	publishing.Expiration = strconv.FormatInt(messaging.RetryDelay(s.options.RetryBackoff, max(attempt-1, 1), messageID).Milliseconds(), 10)
	if err := s.publish(ctx, topic+".retry", rabbitRetryRoutingKey(channel, attempt), publishing); err != nil {
		return delivery.Nack(false, true)
	}
	return delivery.Ack(false)
}

func (s *subscriber) publish(ctx context.Context, exchange, routingKey string, publishing amqp.Publishing) error {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	return s.channel.PublishWithContext(ctx, exchange, routingKey, false, false, publishing)
}

func cloneRabbitPublishing(delivery amqp.Delivery) amqp.Publishing {
	headers := amqp.Table{}
	for key, value := range delivery.Headers {
		headers[key] = value
	}
	return amqp.Publishing{Headers: headers, ContentType: delivery.ContentType, ContentEncoding: delivery.ContentEncoding, DeliveryMode: amqp.Persistent, Priority: delivery.Priority, CorrelationId: delivery.CorrelationId, ReplyTo: delivery.ReplyTo, MessageId: delivery.MessageId, Timestamp: delivery.Timestamp, Type: delivery.Type, UserId: delivery.UserId, AppId: delivery.AppId, Body: delivery.Body}
}

func rabbitAttempt(headers amqp.Table) int {
	attempt, _ := parseRabbitAttempt(headers)
	return attempt
}

func parseRabbitAttempt(headers amqp.Table) (int, error) {
	if headers == nil {
		return 1, nil
	}
	value, exists := headers[rabbitDeliveryAttemptHeader]
	if !exists {
		return 1, nil
	}
	var attempt int
	switch typed := value.(type) {
	case int32:
		attempt = int(typed)
	case int64:
		attempt = int(typed)
	case int:
		attempt = typed
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 1, fmt.Errorf("invalid %s header %q", rabbitDeliveryAttemptHeader, typed)
		}
		attempt = parsed
	default:
		return 1, fmt.Errorf("invalid %s header type %T", rabbitDeliveryAttemptHeader, value)
	}
	if attempt < 1 {
		return 1, fmt.Errorf("invalid %s header %d", rabbitDeliveryAttemptHeader, attempt)
	}
	return attempt, nil
}

func rabbitDeliveryMetadata(headers amqp.Table) (int, bool, error) {
	attempt, attemptErr := parseRabbitAttempt(headers)
	failedOnly, failedOnlyErr := parseRabbitHeaderBool(headers, rabbitFailedOnlyHeader)
	if attemptErr != nil && failedOnlyErr != nil {
		return attempt, true, errors.Join(attemptErr, failedOnlyErr)
	}
	if attemptErr != nil {
		return attempt, true, attemptErr
	}
	if failedOnlyErr != nil {
		return attempt, true, failedOnlyErr
	}
	return attempt, failedOnly, nil
}

func parseRabbitHeaderBool(headers amqp.Table, key string) (bool, error) {
	value, ok := headers[key]
	if !ok {
		return false, nil
	}
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		parsed, err := strconv.ParseBool(typed)
		if err != nil {
			return false, fmt.Errorf("invalid %s header %q", key, typed)
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("invalid %s header type %T", key, value)
	}
}

func rabbitHeaderString(headers amqp.Table, key, fallback string) string {
	if value, ok := headers[key]; ok && fmt.Sprint(value) != "" {
		return fmt.Sprint(value)
	}
	return fallback
}

func rabbitRetryRoutingKey(channel string, attempt int) string {
	return fmt.Sprintf("%s.%d", channel, attempt)
}

func (s *subscriber) Stop() {
	s.mu.Lock()
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	consumers := make([]*consumer, 0, len(s.consumers))
	for _, item := range s.consumers {
		item.cancel()
		consumers = append(consumers, item)
	}
	s.mu.Unlock()
	for _, item := range consumers {
		<-item.done
	}
}

func (s *subscriber) Close() error {
	s.Stop()
	if s.channel != nil {
		_ = s.channel.Close()
	}
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

var _ messaging.Subscriber = (*subscriber)(nil)
