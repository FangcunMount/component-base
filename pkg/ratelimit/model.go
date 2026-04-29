package ratelimit

import (
	"context"
	"time"
)

// Outcome 描述一次限流决策的通用结果。
type Outcome string

const (
	OutcomeAllowed      Outcome = "allowed"
	OutcomeRateLimited  Outcome = "rate_limited"
	OutcomeDegradedOpen Outcome = "degraded_open"
)

// Policy 描述一个有边界的限流控制点。
type Policy struct {
	Component     string
	Scope         string
	Resource      string
	Strategy      string
	RatePerSecond float64
	Burst         int
}

// Valid 判断策略是否能构造正数 token bucket。
func (p Policy) Valid() bool {
	return p.RatePerSecond > 0 && p.Burst > 0
}

// Decision 表示一次与传输协议无关的限流检查结果。
type Decision struct {
	Allowed           bool
	RetryAfter        time.Duration
	RetryAfterSeconds int
	Policy            Policy
	Outcome           Outcome
}

// Limiter 决定一个请求 key 是否允许通过。
type Limiter interface {
	Decide(ctx context.Context, key string) Decision
}

func allowedDecision(policy Policy, outcome Outcome) Decision {
	return Decision{
		Allowed: true,
		Policy:  policy,
		Outcome: outcome,
	}
}

func limitedDecision(policy Policy, retryAfter time.Duration, retryAfterSeconds int) Decision {
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}
	return Decision{
		Allowed:           false,
		RetryAfter:        retryAfter,
		RetryAfterSeconds: retryAfterSeconds,
		Policy:            policy,
		Outcome:           OutcomeRateLimited,
	}
}
