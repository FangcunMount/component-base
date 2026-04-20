package database

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	redisruntime "github.com/FangcunMount/component-base/pkg/redis/runtime"
)

// RedisConfig 是 Foundation 层运行时配置的兼容别名。
type RedisConfig = redisruntime.Config

// RedisConnection Redis 连接实现
type RedisConnection struct {
	handle *redisruntime.Handle
}

// NewRedisConnection 创建 Redis 连接
func NewRedisConnection(config *RedisConfig) *RedisConnection {
	return &RedisConnection{
		handle: redisruntime.NewHandle("", config),
	}
}

// Type 返回数据库类型
func (r *RedisConnection) Type() DatabaseType {
	return Redis
}

// Connect 连接 Redis 数据库
func (r *RedisConnection) Connect() error {
	if r == nil || r.handle == nil {
		return fmt.Errorf("redis connection is nil")
	}
	return r.handle.Connect()
}

// Close 关闭 Redis 连接
func (r *RedisConnection) Close() error {
	if r == nil || r.handle == nil {
		return nil
	}
	return r.handle.Close()
}

// HealthCheck 检查 Redis 连接是否健康
func (r *RedisConnection) HealthCheck(ctx context.Context) error {
	if r == nil || r.handle == nil {
		return fmt.Errorf("redis connection is nil")
	}
	return r.handle.HealthCheck(ctx)
}

// GetClient 获取 Redis 客户端
func (r *RedisConnection) GetClient() interface{} {
	if r == nil || r.handle == nil {
		return nil
	}
	return r.handle.Client()
}

// Client 返回强类型 Redis 客户端。
func (r *RedisConnection) Client() goredis.UniversalClient {
	if r == nil || r.handle == nil {
		return nil
	}
	return r.handle.Client()
}

// Config 返回解析后的 Redis 配置。
func (r *RedisConnection) Config() *RedisConfig {
	if r == nil || r.handle == nil {
		return nil
	}
	return r.handle.Config()
}
