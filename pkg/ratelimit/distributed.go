package ratelimit

import (
	"context"
	"time"
)

// Backend 是 DistributedLimiter 使用的共享 token bucket 存储端口。
type Backend interface {
	Allow(ctx context.Context, key string, ratePerSecond float64, burst int) (bool, time.Duration, error)
}

// DistributedLimiter 将共享 token bucket backend 适配为限流决策器。
type DistributedLimiter struct {
	backend Backend
	policy  Policy
}

func NewDistributedLimiter(backend Backend, policy Policy) *DistributedLimiter {
	return &DistributedLimiter{backend: backend, policy: policy}
}

func (l *DistributedLimiter) Decide(ctx context.Context, key string) Decision {
	if l == nil || l.backend == nil || key == "" || !l.policy.Valid() {
		return allowedDecision(l.policy, OutcomeDegradedOpen)
	}
	allowed, retryAfter, err := l.backend.Allow(ctx, key, l.policy.RatePerSecond, l.policy.Burst)
	if err != nil {
		return allowedDecision(l.policy, OutcomeDegradedOpen)
	}
	if allowed {
		return allowedDecision(l.policy, OutcomeAllowed)
	}
	seconds := int(retryAfter.Seconds()) + 1
	if seconds < 1 {
		seconds = 1
	}
	return limitedDecision(l.policy, retryAfter, seconds)
}
