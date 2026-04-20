package database

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	redisruntime "github.com/FangcunMount/component-base/pkg/redis/runtime"
)

type RedisProfileState = redisruntime.ProfileState

const (
	RedisProfileStateMissing     = redisruntime.ProfileStateMissing
	RedisProfileStateAvailable   = redisruntime.ProfileStateAvailable
	RedisProfileStateUnavailable = redisruntime.ProfileStateUnavailable
)

type RedisProfileStatus = redisruntime.ProfileStatus

// NamedRedisRegistry 管理默认 Redis 连接与可选命名 profile。
type NamedRedisRegistry struct {
	runtime *redisruntime.Registry
}

// NewNamedRedisRegistry 使用可选默认配置与命名 profile 配置创建 Redis 注册表。
func NewNamedRedisRegistry(defaultConfig *RedisConfig, profiles map[string]*RedisConfig) *NamedRedisRegistry {
	return &NamedRedisRegistry{
		runtime: redisruntime.New(defaultConfig, profiles),
	}
}

// HasConnections 返回注册表是否至少存在一个已配置连接。
func (r *NamedRedisRegistry) HasConnections() bool {
	return r != nil && r.runtime != nil && r.runtime.HasConnections()
}

// Connect 初始化默认连接与命名 profile 连接。
func (r *NamedRedisRegistry) Connect() error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Connect()
}

// Close 关闭所有已初始化的 Redis 连接。
func (r *NamedRedisRegistry) Close() error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Close()
}

// HealthCheck 检查默认连接与命名 profile 连接的健康状态。
func (r *NamedRedisRegistry) HealthCheck(ctx context.Context) error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.HealthCheck(ctx)
}

// ProfileStatus 返回指定命名 profile 的状态。
func (r *NamedRedisRegistry) ProfileStatus(name string) RedisProfileStatus {
	status := RedisProfileStatus{Name: name, State: RedisProfileStateMissing}
	if r == nil || r.runtime == nil || name == "" {
		return status
	}
	return r.runtime.Status(name)
}

// ProfileStatuses 返回所有已配置命名 profile 的当前状态。
func (r *NamedRedisRegistry) ProfileStatuses() map[string]RedisProfileStatus {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Statuses()
}

// GetConnection 返回指定命名 profile 的连接。
// 缺失 profile 会回退到默认连接；已配置但不可用的 profile 会返回错误。
func (r *NamedRedisRegistry) GetConnection(name string) (*RedisConnection, error) {
	if r == nil || r.runtime == nil {
		return nil, fmt.Errorf("redis registry is nil")
	}
	handle, err := r.runtime.Bind(name)
	if err != nil {
		return nil, err
	}
	return &RedisConnection{handle: handle}, nil
}

// GetClient 返回指定命名 profile 的 Redis 客户端。
// 缺失 profile 会回退到默认连接；已配置但不可用的 profile 会返回错误。
func (r *NamedRedisRegistry) GetClient(name string) (goredis.UniversalClient, error) {
	if r == nil || r.runtime == nil {
		return nil, fmt.Errorf("redis registry is nil")
	}
	return r.runtime.Client(name)
}

// Profiles 返回已配置命名 profile 的快照信息。
func (r *NamedRedisRegistry) Profiles() []redisruntime.RedisProfile {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Profiles()
}

func cloneRedisConfig(cfg *RedisConfig) *RedisConfig {
	return redisruntime.CloneConfig(cfg)
}

func mergeRedisConfig(base, override *RedisConfig) *RedisConfig {
	return redisruntime.MergeConfig(base, override)
}
