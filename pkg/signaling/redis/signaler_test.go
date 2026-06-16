package redis

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/signaling"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

type testSignal struct {
	AssessmentID string `json:"assessment_id"`
	Status       string `json:"status"`
}

func (testSignal) SignalName() string { return "report_status_changed" }
func (s testSignal) SignalKey() string {
	return s.AssessmentID
}

type emptyNameSignal struct {
	ID string `json:"id"`
}

func (emptyNameSignal) SignalName() string { return "" }
func (s emptyNameSignal) SignalKey() string {
	return s.ID
}

func TestChannelName(t *testing.T) {
	s := NewSignaler[testSignal](nil, DefaultOptions())
	if got := s.channel(testSignal{}.SignalName()); got != "signal:report_status_changed" {
		t.Fatalf("channel() = %q, want %q", got, "signal:report_status_changed")
	}
}

func TestNotifyAndWatch(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	s := NewSignaler[testSignal](client, Options{Prefix: "qs:signal"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan testSignal, 1)
	watchErr := make(chan error, 1)
	go func() {
		watchErr <- s.Watch(ctx, func(_ context.Context, signal testSignal) {
			received <- signal
		})
	}()

	time.Sleep(20 * time.Millisecond)
	if err := s.Notify(ctx, testSignal{
		AssessmentID: "assess-1",
		Status:       "completed",
	}); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	select {
	case got := <-received:
		if got.AssessmentID != "assess-1" || got.Status != "completed" {
			t.Fatalf("received signal = %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("wait signal timeout")
	}

	cancel()
	select {
	case err := <-watchErr:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Watch() after cancel error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("watch stop timeout")
	}
}

func TestWatchDecodeErrorDoesNotPanic(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	var decodeErrors atomic.Int32
	s := NewSignaler[testSignal](client, Options{
		Prefix: "qs:signal",
		ErrorHandler: func(err error) {
			if err != nil {
				decodeErrors.Add(1)
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.Watch(ctx, func(_ context.Context, _ testSignal) {})
	}()

	time.Sleep(20 * time.Millisecond)
	if err := client.Publish(ctx, "qs:signal:report_status_changed", "not-json").Err(); err != nil {
		t.Fatalf("Publish() invalid payload error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if decodeErrors.Load() == 0 {
		t.Fatalf("decode error handler was not called")
	}

	cancel()
	<-done
}

func TestWatchRequiresSignalName(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	s := NewSignaler[emptyNameSignal](client, DefaultOptions())
	err := s.Watch(context.Background(), func(_ context.Context, _ emptyNameSignal) {})
	if !errors.Is(err, signaling.ErrEmptySignalName) {
		t.Fatalf("Watch() error = %v, want %v", err, signaling.ErrEmptySignalName)
	}
}

func TestWatchReturnsErrorWhenRedisUnavailable(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	s := NewSignaler[testSignal](client, DefaultOptions())
	err := s.Watch(ctx, func(_ context.Context, _ testSignal) {})
	if err == nil {
		t.Fatalf("Watch() error = nil, want non-nil")
	}
}

func TestNotifyRequiresSignalName(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	s := NewSignaler[emptyNameSignal](client, DefaultOptions())
	err := s.Notify(context.Background(), emptyNameSignal{ID: "x"})
	if !errors.Is(err, signaling.ErrEmptySignalName) {
		t.Fatalf("Notify() error = %v, want %v", err, signaling.ErrEmptySignalName)
	}
}

func newRedisClient(t *testing.T) (*goredis.Client, func()) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return client, func() {
		_ = client.Close()
		mr.Close()
	}
}
