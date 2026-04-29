package backpressure

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAcquireDoesNotWrapOperationContextWithLimiterTimeout(t *testing.T) {
	limiter := NewLimiter(1, 50*time.Millisecond)

	ctx := context.Background()
	gotCtx, release, err := limiter.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer release()

	if gotCtx != ctx {
		t.Fatalf("Acquire() should preserve original context")
	}
	if _, ok := gotCtx.Deadline(); ok {
		t.Fatalf("Acquire() should not add a deadline to the downstream operation context")
	}
}

func TestAcquireTimeoutOnlyAppliesWhileWaitingForSlot(t *testing.T) {
	limiter := NewLimiter(1, 50*time.Millisecond)

	_, release, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	defer release()

	_, _, err = limiter.Acquire(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestAcquireReportsOutcomes(t *testing.T) {
	observer := &recordingObserver{}
	limiter := NewLimiterWithOptions(1, 10*time.Millisecond, Options{
		Component:  "apiserver",
		Dependency: "mysql",
		Observer:   observer,
	})

	_, release, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	release()

	if !observer.has(OutcomeAcquired) {
		t.Fatal("expected acquired outcome")
	}
	if !observer.has(OutcomeReleased) {
		t.Fatal("expected released outcome")
	}
}

func TestAcquireTimeoutReportsOutcome(t *testing.T) {
	observer := &recordingObserver{}
	limiter := NewLimiterWithOptions(1, 10*time.Millisecond, Options{Observer: observer})

	_, release, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	defer release()

	if _, _, err := limiter.Acquire(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Acquire() error = %v, want deadline exceeded", err)
	}
	if !observer.has(OutcomeTimeout) {
		t.Fatal("expected timeout outcome")
	}
}

func TestStatsReportsInFlightAndConfig(t *testing.T) {
	limiter := NewLimiterWithOptions(2, 150*time.Millisecond, Options{
		Component:  "apiserver",
		Dependency: "mysql",
	})

	stats := limiter.Stats("mysql")
	if !stats.Enabled || stats.MaxInflight != 2 || stats.TimeoutMillis != 150 {
		t.Fatalf("initial stats = %+v", stats)
	}
	if stats.InFlight != 0 {
		t.Fatalf("initial in-flight = %d, want 0", stats.InFlight)
	}

	_, release, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer release()

	stats = limiter.Stats("mysql")
	if stats.InFlight != 1 {
		t.Fatalf("in-flight = %d, want 1", stats.InFlight)
	}
}

func TestNilLimiterStatsIsDegraded(t *testing.T) {
	var limiter *Limiter
	stats := limiter.Stats("mysql")
	if stats.Enabled || !stats.Degraded {
		t.Fatalf("nil stats = %+v, want disabled degraded", stats)
	}
}

type recordingObserver struct {
	events []Event
}

func (r *recordingObserver) OnBackpressure(_ context.Context, event Event) {
	r.events = append(r.events, event)
}

func (r *recordingObserver) has(outcome Outcome) bool {
	for _, event := range r.events {
		if event.Outcome == outcome {
			return true
		}
	}
	return false
}
