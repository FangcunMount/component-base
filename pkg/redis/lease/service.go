package lease

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/redis/observability"
)

var (
	releaseScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end
`)
	renewScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
	return 0
end
`)
)

type TokenSource interface {
	NewToken(ctx context.Context) (LeaseToken, error)
}

type randomTokenSource struct{}

func (randomTokenSource) NewToken(context.Context) (LeaseToken, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate lease token: %w", err)
	}
	return LeaseToken(hex.EncodeToString(buf)), nil
}

type Service struct {
	client      goredis.UniversalClient
	observer    observability.LeaseObserver
	tokenSource TokenSource
}

type Option func(*Service)

// WithObserver 配置租约锁观测器。
func WithObserver(observer observability.LeaseObserver) Option {
	return func(s *Service) {
		if observer != nil {
			s.observer = observer
		}
	}
}

// WithTokenSource 配置自定义 token 生成器。
func WithTokenSource(source TokenSource) Option {
	return func(s *Service) {
		if source != nil {
			s.tokenSource = source
		}
	}
}

// NewService 基于 Redis 客户端创建租约锁服务。
func NewService(client goredis.UniversalClient, opts ...Option) *Service {
	svc := &Service{
		client:      client,
		observer:    observability.NopLeaseObserver{},
		tokenSource: randomTokenSource{},
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// Acquire 使用 SET NX 语义尝试获取租约锁。
func (s *Service) Acquire(ctx context.Context, key LeaseKey, ttl time.Duration, owner *LeaseOwner) (LeaseAttempt, error) {
	if s == nil || s.client == nil {
		return LeaseAttempt{}, fmt.Errorf("redis client is nil")
	}
	if key.String() == "" {
		return LeaseAttempt{}, fmt.Errorf("lock key is empty")
	}
	if ttl <= 0 {
		return LeaseAttempt{}, fmt.Errorf("lock ttl must be positive")
	}

	token, err := s.tokenSource.NewToken(ctx)
	if err != nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "acquire",
			Key:       key.String(),
			Owner:     owner.Identity(),
			TTL:       ttl,
			Err:       err,
		})
		return LeaseAttempt{}, err
	}

	ok, err := s.client.SetNX(ctx, key.String(), token.String(), ttl).Result()
	if err != nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "acquire",
			Key:       key.String(),
			Token:     token.String(),
			Owner:     owner.Identity(),
			TTL:       ttl,
			Err:       err,
		})
		return LeaseAttempt{}, err
	}

	attempt := LeaseAttempt{Acquired: ok}
	if ok {
		attempt.Lease = Lease{
			Key:        key,
			Token:      token,
			TTL:        ttl,
			Owner:      owner,
			AcquiredAt: time.Now(),
		}
	}
	s.observer.OnLease(ctx, observability.LeaseEvent{
		Operation: "acquire",
		Key:       key.String(),
		Token:     token.String(),
		Owner:     owner.Identity(),
		TTL:       ttl,
		Acquired:  ok,
	})
	return attempt, nil
}

// Renew 使用 compare-and-expire 语义续约当前租约。
func (s *Service) Renew(ctx context.Context, lease Lease, ttl time.Duration) (Lease, bool, error) {
	if s == nil || s.client == nil {
		return Lease{}, false, fmt.Errorf("redis client is nil")
	}
	if lease.Key.String() == "" || lease.Token.String() == "" {
		return Lease{}, false, fmt.Errorf("lease key/token is empty")
	}
	if ttl <= 0 {
		return Lease{}, false, fmt.Errorf("lock ttl must be positive")
	}

	result, err := renewScript.Run(ctx, s.client, []string{lease.Key.String()}, lease.Token.String(), ttl.Milliseconds()).Int64()
	if err == goredis.Nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "renew",
			Key:       lease.Key.String(),
			Token:     lease.Token.String(),
			Owner:     lease.Owner.Identity(),
			TTL:       ttl,
			Owned:     false,
		})
		return Lease{}, false, nil
	}
	if err != nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "renew",
			Key:       lease.Key.String(),
			Token:     lease.Token.String(),
			Owner:     lease.Owner.Identity(),
			TTL:       ttl,
			Err:       err,
		})
		return Lease{}, false, err
	}
	if result == 0 {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "renew",
			Key:       lease.Key.String(),
			Token:     lease.Token.String(),
			Owner:     lease.Owner.Identity(),
			TTL:       ttl,
			Owned:     false,
		})
		return Lease{}, false, nil
	}

	lease.TTL = ttl
	lease.AcquiredAt = time.Now()
	s.observer.OnLease(ctx, observability.LeaseEvent{
		Operation: "renew",
		Key:       lease.Key.String(),
		Token:     lease.Token.String(),
		Owner:     lease.Owner.Identity(),
		TTL:       ttl,
		Owned:     true,
	})
	return lease, true, nil
}

// Release 仅在 token 仍匹配时释放租约锁。
func (s *Service) Release(ctx context.Context, lease Lease) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if lease.Key.String() == "" || lease.Token.String() == "" {
		return nil
	}

	_, err := releaseScript.Run(ctx, s.client, []string{lease.Key.String()}, lease.Token.String()).Result()
	if err == goredis.Nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "release",
			Key:       lease.Key.String(),
			Token:     lease.Token.String(),
			Owner:     lease.Owner.Identity(),
			Owned:     false,
		})
		return nil
	}
	s.observer.OnLease(ctx, observability.LeaseEvent{
		Operation: "release",
		Key:       lease.Key.String(),
		Token:     lease.Token.String(),
		Owner:     lease.Owner.Identity(),
		Owned:     err == nil,
		Err:       err,
	})
	return err
}

// CheckOwnership 检查给定租约是否仍然持有该锁键。
func (s *Service) CheckOwnership(ctx context.Context, lease Lease) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	if lease.Key.String() == "" || lease.Token.String() == "" {
		return false, fmt.Errorf("lease key/token is empty")
	}

	result, err := s.client.Get(ctx, lease.Key.String()).Result()
	if err == goredis.Nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "check",
			Key:       lease.Key.String(),
			Token:     lease.Token.String(),
			Owner:     lease.Owner.Identity(),
			Owned:     false,
		})
		return false, nil
	}
	if err != nil {
		s.observer.OnLease(ctx, observability.LeaseEvent{
			Operation: "check",
			Key:       lease.Key.String(),
			Token:     lease.Token.String(),
			Owner:     lease.Owner.Identity(),
			Err:       err,
		})
		return false, err
	}
	owned := result == lease.Token.String()
	s.observer.OnLease(ctx, observability.LeaseEvent{
		Operation: "check",
		Key:       lease.Key.String(),
		Token:     lease.Token.String(),
		Owner:     lease.Owner.Identity(),
		Owned:     owned,
	})
	return owned, nil
}
