package examples

import (
	"context"
	"time"

	"github.com/FangcunMount/component-base/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// SetupRedisWithLogger 演示如何在 Redis 客户端中配置日志钩子
func SetupRedisWithLogger() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// 方式一：使用简单参数创建
	// enabled: 是否启用日志
	// slowThreshold: 慢命令阈值
	redisHook := logger.NewRedisHook(true, 200*time.Millisecond)
	client.AddHook(redisHook)

	return client
}

// SetupRedisWithCustomConfig 演示使用自定义配置
func SetupRedisWithCustomConfig() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// 方式二：使用配置结构体
	config := logger.RedisHookConfig{
		// 是否启用日志记录
		Enabled: true,
		// 慢命令阈值（超过此时间会记录警告）
		SlowThreshold: 100 * time.Millisecond,
	}
	redisHook := logger.NewRedisHookWithConfig(config)
	client.AddHook(redisHook)

	return client
}

// SetupRedisCluster 演示集群模式下的配置
func SetupRedisCluster() *redis.ClusterClient {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7000",
			"localhost:7001",
			"localhost:7002",
		},
	})

	// 集群模式同样支持添加钩子
	redisHook := logger.NewRedisHook(true, 200*time.Millisecond)
	client.AddHook(redisHook)

	return client
}

// DisableRedisLogging 演示如何禁用日志记录
func DisableRedisLogging() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 设置 enabled=false 禁用日志
	redisHook := logger.NewRedisHook(false, 0)
	client.AddHook(redisHook)

	return client
}

// RedisUsageExample 演示 Redis 操作时日志的自动记录
func RedisUsageExample(client *redis.Client) {
	ctx := context.Background()

	// 所有 Redis 操作都会自动记录日志
	// 无需在业务代码中手动添加

	// SET 命令
	client.Set(ctx, "user:1:name", "John", time.Hour)

	// GET 命令
	client.Get(ctx, "user:1:name")

	// 管道操作
	pipe := client.Pipeline()
	pipe.Set(ctx, "user:1:age", 25, time.Hour)
	pipe.Set(ctx, "user:1:email", "john@example.com", time.Hour)
	pipe.Exec(ctx)
}

// RedisLoggerOutput 说明 Redis 日志钩子产生的日志格式
//
// 正常命令执行日志（Debug 级别）：
//
//	{
//	    "level": "debug",
//	    "ts": "2025-12-10T10:00:00.010Z",
//	    "msg": "Redis command executed",
//	    "command": "SET user:1:name John",
//	    "elapsed_ms": 2.5,
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
//
// 慢命令警告日志：
//
//	{
//	    "level": "warn",
//	    "ts": "2025-12-10T10:00:00.300Z",
//	    "msg": "Redis slow command",
//	    "command": "KEYS user:*",
//	    "elapsed_ms": 250.5,
//	    "event": "slow_command",
//	    "slow_threshold": "200ms",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
//
// 命令执行错误日志：
//
//	{
//	    "level": "error",
//	    "ts": "2025-12-10T10:00:00.020Z",
//	    "msg": "Redis command failed",
//	    "command": "GET user:1:name",
//	    "elapsed_ms": 5.0,
//	    "error": "WRONGTYPE Operation against a key holding the wrong kind of value",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
//
// 管道执行日志：
//
//	{
//	    "level": "debug",
//	    "ts": "2025-12-10T10:00:00.015Z",
//	    "msg": "Redis pipeline executed",
//	    "command_count": 3,
//	    "commands": "SET, SET, GET",
//	    "elapsed_ms": 8.0,
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
//
// 注意事项：
//   - AUTH 和 HELLO 命令的参数会自动脱敏为 ***
//   - 超过 100 字符的参数会被截断
//   - 超过 500 字符的完整命令会被截断
//   - redis.Nil 错误（key 不存在）不会记录为错误
func RedisLoggerOutput() {}
