package backpressure

import (
	"context"
	"time"
)

// Outcome 描述一次背压限流事件的结果。
type Outcome string

const (
	OutcomeAcquired Outcome = "backpressure_acquired"
	OutcomeTimeout  Outcome = "backpressure_timeout"
	OutcomeReleased Outcome = "backpressure_released"
)

// Event 描述一次限流器状态变化。
type Event struct {
	Component   string
	Dependency  string
	Resource    string
	Strategy    string
	Outcome     Outcome
	Wait        time.Duration
	InFlight    int
	MaxInflight int
	Err         error
}

// Observer 接收背压限流事件。
type Observer interface {
	OnBackpressure(ctx context.Context, event Event)
}

// NopObserver 忽略所有限流事件。
type NopObserver struct{}

func (NopObserver) OnBackpressure(context.Context, Event) {}

// Acquirer 等待一个执行槽位，并返回 release 函数。
type Acquirer interface {
	Acquire(context.Context) (context.Context, func(), error)
}

// Options 配置一个背压限流器。
type Options struct {
	Component  string
	Dependency string
	Observer   Observer
}

// Stats 是一个限流器的机制级快照。
type Stats struct {
	Component     string
	Name          string
	Dependency    string
	Strategy      string
	Enabled       bool
	MaxInflight   int
	InFlight      int
	TimeoutMillis int64
	Degraded      bool
	Reason        string
}

// Limiter 提供一个可选等待超时的 in-flight 限流器。
type Limiter struct {
	sem         chan struct{}
	maxInflight int
	timeout     time.Duration
	component   string
	dependency  string
	observer    Observer
}

// NewLimiter 根据最大 in-flight 数和可选等待超时创建限流器。
func NewLimiter(maxInflight int, timeout time.Duration) *Limiter {
	return NewLimiterWithOptions(maxInflight, timeout, Options{})
}

func NewLimiterWithOptions(maxInflight int, timeout time.Duration, opts Options) *Limiter {
	if maxInflight <= 0 {
		return nil
	}
	observer := opts.Observer
	if observer == nil {
		observer = NopObserver{}
	}
	return &Limiter{
		sem:         make(chan struct{}, maxInflight),
		maxInflight: maxInflight,
		timeout:     timeout,
		component:   opts.Component,
		dependency:  opts.Dependency,
		observer:    observer,
	}
}

// Acquire 等待一个槽位，直到成功、context 取消或等待超时。
// 返回的 context 保持为原始请求 context，release 用于释放已获取的槽位。
func (l *Limiter) Acquire(ctx context.Context) (context.Context, func(), error) {
	if l == nil {
		return ctx, func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	waitStartedAt := time.Now()
	waitCtx := ctx
	var cancel context.CancelFunc
	if l.timeout > 0 {
		if deadline, ok := waitCtx.Deadline(); !ok || time.Until(deadline) > l.timeout {
			waitCtx, cancel = context.WithTimeout(waitCtx, l.timeout)
		}
	}

	select {
	case l.sem <- struct{}{}:
		l.observe(ctx, OutcomeAcquired, time.Since(waitStartedAt), len(l.sem), nil)
		release := func() {
			<-l.sem
			l.observe(ctx, OutcomeReleased, 0, len(l.sem), nil)
			if cancel != nil {
				cancel()
			}
		}
		return ctx, release, nil
	case <-waitCtx.Done():
		if cancel != nil {
			cancel()
		}
		err := waitCtx.Err()
		l.observe(ctx, OutcomeTimeout, time.Since(waitStartedAt), len(l.sem), err)
		return ctx, func() {}, err
	}
}

// Stats 返回机制级限流器快照。
func (l *Limiter) Stats(name string) Stats {
	if l == nil {
		return Stats{
			Name:     name,
			Enabled:  false,
			Degraded: true,
			Reason:   "backpressure limiter disabled",
		}
	}
	if name == "" {
		name = l.dependency
	}
	return Stats{
		Component:     l.component,
		Name:          name,
		Dependency:    l.dependency,
		Strategy:      "semaphore",
		Enabled:       true,
		MaxInflight:   l.maxInflight,
		InFlight:      len(l.sem),
		TimeoutMillis: l.timeout.Milliseconds(),
	}
}

func (l *Limiter) observe(ctx context.Context, outcome Outcome, wait time.Duration, inFlight int, err error) {
	if l == nil || l.observer == nil {
		return
	}
	l.observer.OnBackpressure(ctx, Event{
		Component:   l.component,
		Dependency:  l.dependency,
		Resource:    "downstream",
		Strategy:    "semaphore",
		Outcome:     outcome,
		Wait:        wait,
		InFlight:    inFlight,
		MaxInflight: l.maxInflight,
		Err:         err,
	})
}
