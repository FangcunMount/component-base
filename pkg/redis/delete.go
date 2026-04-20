package rediskit

import (
	"context"

	goredis "github.com/redis/go-redis/v9"

	redisops "github.com/FangcunMount/component-base/pkg/redis/ops"
)

// DeleteBatch 是分批删除观测结果的兼容别名。
type DeleteBatch = redisops.DeleteBatch

// DeleteByPatternOptions 是按模式删除配置的兼容别名。
type DeleteByPatternOptions = redisops.DeleteByPatternOptions

// DefaultDeleteByPatternOptions 返回标准按模式删除默认配置。
func DefaultDeleteByPatternOptions() DeleteByPatternOptions {
	return redisops.DefaultDeleteByPatternOptions()
}

// ScanKeys 使用 SCAN 收集匹配模式的键。
func ScanKeys(ctx context.Context, client goredis.UniversalClient, pattern string, count int64) ([]string, error) {
	return redisops.ScanKeys(ctx, client, pattern, count)
}

// DeleteByPattern 按批次扫描并删除匹配模式的键。
func DeleteByPattern(ctx context.Context, client goredis.UniversalClient, pattern string, opts DeleteByPatternOptions) (int, error) {
	return redisops.DeleteByPattern(ctx, client, pattern, opts)
}
