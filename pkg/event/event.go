package event

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const SourceDefault = "application"

type DomainEvent interface {
	EventID() string
	EventType() string
	OccurredAt() time.Time
	AggregateType() string
	AggregateID() string
}

type Stager interface {
	Stage(ctx context.Context, events ...DomainEvent) error
}

type Publisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}

type EventPublisher = Publisher

type EventSubscriber interface {
	Subscribe(eventType string, handler EventHandler) error
	Start(ctx context.Context) error
	Stop() error
}

type EventHandler func(ctx context.Context, event DomainEvent) error

type EventStore interface {
	Save(ctx context.Context, events []DomainEvent) error
	Load(ctx context.Context, aggregateType, aggregateID string) ([]DomainEvent, error)
	LoadFrom(ctx context.Context, aggregateType, aggregateID string, fromVersion int64) ([]DomainEvent, error)
}

type BaseEvent struct {
	ID                 string    `json:"id"`
	EventTypeValue     string    `json:"eventType"`
	OccurredAtValue    time.Time `json:"occurredAt"`
	AggregateTypeValue string    `json:"aggregateType"`
	AggregateIDValue   string    `json:"aggregateID"`
}

func NewBaseEvent(eventType, aggregateType, aggregateID string) BaseEvent {
	return BaseEvent{
		ID:                 uuid.New().String(),
		EventTypeValue:     eventType,
		OccurredAtValue:    time.Now(),
		AggregateTypeValue: aggregateType,
		AggregateIDValue:   aggregateID,
	}
}

func (e BaseEvent) EventID() string       { return e.ID }
func (e BaseEvent) EventType() string     { return e.EventTypeValue }
func (e BaseEvent) OccurredAt() time.Time { return e.OccurredAtValue }
func (e BaseEvent) AggregateType() string { return e.AggregateTypeValue }
func (e BaseEvent) AggregateID() string   { return e.AggregateIDValue }

type Event[T any] struct {
	BaseEvent
	Data T `json:"data"`
}

func New[T any](eventType, aggregateType, aggregateID string, data T) Event[T] {
	return Event[T]{
		BaseEvent: NewBaseEvent(eventType, aggregateType, aggregateID),
		Data:      data,
	}
}

func (e Event[T]) Payload() T { return e.Data }

type EventRaiser interface {
	Events() []DomainEvent
	ClearEvents()
}

type EventCollector struct {
	events []DomainEvent
}

func NewEventCollector() *EventCollector {
	return &EventCollector{events: make([]DomainEvent, 0)}
}

func (c *EventCollector) AddEvent(event DomainEvent) {
	c.events = append(c.events, event)
}

func (c *EventCollector) Events() []DomainEvent {
	return c.events
}

func (c *EventCollector) ClearEvents() {
	c.events = make([]DomainEvent, 0)
}

func (c *EventCollector) HasEvents() bool {
	return len(c.events) > 0
}

type NopEventPublisher struct{}

func NewNopEventPublisher() *NopEventPublisher {
	return &NopEventPublisher{}
}

func (p *NopEventPublisher) Publish(_ context.Context, _ DomainEvent) error {
	return nil
}

func (p *NopEventPublisher) PublishAll(_ context.Context, _ []DomainEvent) error {
	return nil
}
