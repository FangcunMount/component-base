package rediskit

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	redislease "github.com/FangcunMount/component-base/pkg/redis/lease"
)

// AcquireLease 使用 SET NX EX 语义尝试获取租约锁。
func AcquireLease(ctx context.Context, client goredis.UniversalClient, key string, ttl time.Duration) (string, bool, error) {
	leaseKey, err := redislease.NewLeaseKey(key)
	if err != nil {
		return "", false, err
	}
	attempt, err := redislease.NewService(client).Acquire(ctx, leaseKey, ttl, nil)
	if err != nil {
		return "", false, err
	}
	if !attempt.Acquired {
		return "", false, nil
	}
	return attempt.Lease.Token.String(), true, nil
}

// ReleaseLease 仅在租约 token 仍匹配时释放锁。
func ReleaseLease(ctx context.Context, client goredis.UniversalClient, key, token string) error {
	if key == "" || token == "" {
		return nil
	}
	return redislease.NewService(client).Release(ctx, redislease.Lease{
		Key:   redislease.MustLeaseKey(key),
		Token: redislease.MustLeaseToken(token),
	})
}
