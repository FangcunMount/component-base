package outboxcore

import (
	"strings"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/event"
	"github.com/FangcunMount/component-base/pkg/eventcatalog"
)

type fakeResolver struct {
	topic    string
	delivery eventcatalog.DeliveryClass
}

func (r fakeResolver) GetTopicForEvent(string) (string, bool) {
	return r.topic, r.topic != ""
}

func (r fakeResolver) GetDeliveryClass(string) (eventcatalog.DeliveryClass, bool) {
	return r.delivery, r.delivery != ""
}

func TestBuildRecordsPreservesEnvelopeAndRejectsBestEffort(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 1, 2, 3, 0, time.UTC)
	evt := event.New("sample.created", "Sample", "sample-1", map[string]string{"id": "sample-1"})
	records, err := BuildRecords(BuildRecordsOptions{
		Events:   []event.DomainEvent{evt},
		Resolver: fakeResolver{topic: "sample.topic", delivery: eventcatalog.DeliveryClassDurableOutbox},
		Now:      now,
	})
	if err != nil {
		t.Fatalf("BuildRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].TopicName != "sample.topic" || records[0].Status != StatusPending || records[0].CreatedAt != now {
		t.Fatalf("record = %#v", records[0])
	}
	if !strings.Contains(records[0].PayloadJSON, `"eventType":"sample.created"`) {
		t.Fatalf("payload = %s", records[0].PayloadJSON)
	}

	_, err = BuildRecords(BuildRecordsOptions{
		Events:   []event.DomainEvent{evt},
		Resolver: fakeResolver{topic: "sample.topic", delivery: eventcatalog.DeliveryClassBestEffort},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be staged") {
		t.Fatalf("BuildRecords() error = %v, want best-effort rejection", err)
	}
}

func TestBuildStatusSnapshotFillsMissingUnfinishedStatuses(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 1, 2, 3, 0, time.UTC)
	oldest := now.Add(-2 * time.Second)
	snapshot := BuildStatusSnapshot("mysql", now, []StatusObservation{
		{Status: StatusPending, Count: 2, OldestCreatedAt: &oldest},
		{Status: StatusPublished, Count: 99, OldestCreatedAt: &oldest},
	})
	if len(snapshot.Buckets) != 3 {
		t.Fatalf("bucket count = %d, want 3", len(snapshot.Buckets))
	}
	if snapshot.Buckets[0].Status != StatusPending || snapshot.Buckets[0].OldestAgeSeconds != 2 {
		t.Fatalf("pending bucket = %#v", snapshot.Buckets[0])
	}
	if snapshot.Buckets[1].Status != StatusFailed || snapshot.Buckets[1].Count != 0 {
		t.Fatalf("failed bucket = %#v", snapshot.Buckets[1])
	}
}
