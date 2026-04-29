package outboxcore

import (
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/event"
	"github.com/FangcunMount/component-base/pkg/eventcatalog"
	"github.com/FangcunMount/component-base/pkg/eventcodec"
	"github.com/FangcunMount/component-base/pkg/outbox"
)

const (
	StatusPending    = "pending"
	StatusPublishing = "publishing"
	StatusPublished  = "published"
	StatusFailed     = "failed"

	DefaultPublishingStaleFor       = time.Minute
	DefaultRelayRetryDelay          = 10 * time.Second
	DefaultDecodeFailureRetryDelay  = 10 * time.Second
	DefaultFailedTransitionAttempts = 1
)

var unfinishedStatuses = []string{StatusPending, StatusFailed, StatusPublishing}

func UnfinishedStatuses() []string {
	return append([]string(nil), unfinishedStatuses...)
}

type Record struct {
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	TopicName     string
	PayloadJSON   string
	Status        string
	AttemptCount  int
	NextAttemptAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StatusObservation struct {
	Status          string
	Count           int64
	OldestCreatedAt *time.Time
}

type BuildRecordsOptions struct {
	Events   []event.DomainEvent
	Resolver eventcatalog.TopicResolver
	Encoder  eventcodec.PayloadEncoder
	Now      time.Time
}

func BuildRecords(opts BuildRecordsOptions) ([]Record, error) {
	if len(opts.Events) == 0 {
		return nil, nil
	}
	resolver := opts.Resolver
	if resolver == nil {
		resolver = eventcatalog.NewCatalog(nil)
	}
	encoder := opts.Encoder
	if encoder == nil {
		encoder = eventcodec.EncodeDomainEvent
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	records := make([]Record, 0, len(opts.Events))
	for _, evt := range opts.Events {
		topicName, ok := resolver.GetTopicForEvent(evt.EventType())
		if !ok {
			return nil, fmt.Errorf("event %q not found in event config", evt.EventType())
		}
		if deliveryResolver, ok := resolver.(eventcatalog.DeliveryClassResolver); ok {
			delivery, ok := deliveryResolver.GetDeliveryClass(evt.EventType())
			if !ok {
				return nil, fmt.Errorf("event %q has no delivery class", evt.EventType())
			}
			if delivery != eventcatalog.DeliveryClassDurableOutbox {
				return nil, fmt.Errorf("event %q delivery class %q cannot be staged to outbox", evt.EventType(), delivery)
			}
		}
		payload, err := encoder(evt)
		if err != nil {
			return nil, err
		}
		records = append(records, Record{
			EventID:       evt.EventID(),
			EventType:     evt.EventType(),
			AggregateType: evt.AggregateType(),
			AggregateID:   evt.AggregateID(),
			TopicName:     topicName,
			PayloadJSON:   string(payload),
			Status:        StatusPending,
			AttemptCount:  0,
			NextAttemptAt: now,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return records, nil
}

func BuildStatusSnapshot(store string, now time.Time, observations []StatusObservation) outbox.StatusSnapshot {
	if now.IsZero() {
		now = time.Now()
	}
	byStatus := make(map[string]StatusObservation, len(observations))
	for _, observation := range observations {
		if !isUnfinishedStatus(observation.Status) {
			continue
		}
		byStatus[observation.Status] = observation
	}

	buckets := make([]outbox.StatusBucket, 0, len(unfinishedStatuses))
	for _, status := range unfinishedStatuses {
		observation := byStatus[status]
		ageSeconds := 0.0
		if observation.Count > 0 && observation.OldestCreatedAt != nil {
			ageSeconds = now.Sub(*observation.OldestCreatedAt).Seconds()
			if ageSeconds < 0 {
				ageSeconds = 0
			}
		}
		buckets = append(buckets, outbox.StatusBucket{
			Status:           status,
			Count:            observation.Count,
			OldestCreatedAt:  observation.OldestCreatedAt,
			OldestAgeSeconds: ageSeconds,
		})
	}
	return outbox.StatusSnapshot{
		Store:       store,
		GeneratedAt: now,
		Buckets:     buckets,
	}
}

func DecodePendingEvent(eventID, payloadJSON string, decoders ...eventcodec.PayloadDecoder) (outbox.PendingEvent, error) {
	decoder := eventcodec.DecodeDomainEvent
	if len(decoders) > 0 && decoders[0] != nil {
		decoder = decoders[0]
	}
	evt, err := decoder([]byte(payloadJSON))
	if err != nil {
		return outbox.PendingEvent{}, err
	}
	return outbox.PendingEvent{EventID: eventID, Event: evt}, nil
}

type PublishedTransition struct {
	Status      string
	PublishedAt time.Time
	UpdatedAt   time.Time
}

func NewPublishedTransition(publishedAt time.Time) PublishedTransition {
	return PublishedTransition{
		Status:      StatusPublished,
		PublishedAt: publishedAt,
		UpdatedAt:   publishedAt,
	}
}

type FailedTransition struct {
	Status           string
	LastError        string
	NextAttemptAt    time.Time
	UpdatedAt        time.Time
	AttemptIncrement int
}

func NewFailedTransition(lastError string, nextAttemptAt, updatedAt time.Time) FailedTransition {
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	return FailedTransition{
		Status:           StatusFailed,
		LastError:        lastError,
		NextAttemptAt:    nextAttemptAt,
		UpdatedAt:        updatedAt,
		AttemptIncrement: DefaultFailedTransitionAttempts,
	}
}

func NewDecodeFailureTransition(decodeErr error, now time.Time) FailedTransition {
	if now.IsZero() {
		now = time.Now()
	}
	return NewFailedTransition(
		fmt.Sprintf("decode outbox payload: %v", decodeErr),
		now.Add(DefaultDecodeFailureRetryDelay),
		now,
	)
}

func isUnfinishedStatus(status string) bool {
	for _, known := range unfinishedStatuses {
		if status == known {
			return true
		}
	}
	return false
}
