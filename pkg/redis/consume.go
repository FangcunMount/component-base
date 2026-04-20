package rediskit

import (
	"context"

	goredis "github.com/redis/go-redis/v9"

	redisops "github.com/FangcunMount/component-base/pkg/redis/ops"
)

// ConsumeIfExists 原子地检查并删除单个键。
func ConsumeIfExists(ctx context.Context, client goredis.UniversalClient, key string) (bool, error) {
	return redisops.ConsumeIfExists(ctx, client, key)
}
