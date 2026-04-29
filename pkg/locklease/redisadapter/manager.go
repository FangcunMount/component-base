package redisadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/locklease"
	redislease "github.com/FangcunMount/component-base/pkg/redis/lease"
	"github.com/FangcunMount/component-base/pkg/redis/observability"
	redis "github.com/redis/go-redis/v9"
)

// KeyFunc 将逻辑锁身份 key 映射为具体 Redis key。
type KeyFunc func(string) string

// Manager 将通用 locklease.Manager 端口适配到 Redis 租约锁基础能力。
type Manager struct {
	service *redislease.Service
	keyFunc KeyFunc
}

// Option 配置 Redis 租约锁管理器。
type Option func(*managerOptions)

type managerOptions struct {
	observer    observability.LeaseObserver
	tokenSource redislease.TokenSource
}

// WithObserver 配置底层 Redis lease 观测器。
func WithObserver(observer observability.LeaseObserver) Option {
	return func(opts *managerOptions) {
		opts.observer = observer
	}
}

// WithTokenSource 配置自定义 token 来源，主要用于确定性测试。
func WithTokenSource(source redislease.TokenSource) Option {
	return func(opts *managerOptions) {
		opts.tokenSource = source
	}
}

// NewManager 创建一个基于 Redis 的 locklease.Manager。
func NewManager(client redis.UniversalClient, keyFunc KeyFunc, opts ...Option) *Manager {
	options := managerOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	serviceOptions := []redislease.Option{}
	if options.observer != nil {
		serviceOptions = append(serviceOptions, redislease.WithObserver(options.observer))
	}
	if options.tokenSource != nil {
		serviceOptions = append(serviceOptions, redislease.WithTokenSource(options.tokenSource))
	}
	return &Manager{
		service: redislease.NewService(client, serviceOptions...),
		keyFunc: keyFunc,
	}
}

// Acquire 尝试获取一个具体锁身份。
func (m *Manager) Acquire(ctx context.Context, identity locklease.Identity, ttl time.Duration) (*locklease.Lease, bool, error) {
	if identity.Name == "" {
		return nil, false, fmt.Errorf("lock identity name is empty")
	}
	if ttl <= 0 {
		return nil, false, fmt.Errorf("lock ttl must be greater than 0")
	}
	if m == nil || m.service == nil {
		return nil, false, fmt.Errorf("lock manager is unavailable")
	}

	key, err := redislease.NewLeaseKey(m.redisKey(identity))
	if err != nil {
		return nil, false, err
	}
	attempt, err := m.service.Acquire(ctx, key, ttl, &redislease.LeaseOwner{Label: identity.Name})
	if err != nil {
		return nil, false, err
	}
	if !attempt.Acquired {
		return nil, false, nil
	}
	return &locklease.Lease{
		Key:   attempt.Lease.Key.String(),
		Token: attempt.Lease.Token.String(),
	}, true, nil
}

// AcquireSpec 根据语义锁规格获取租约。
func (m *Manager) AcquireSpec(ctx context.Context, spec locklease.Spec, key string, ttlOverride ...time.Duration) (*locklease.Lease, bool, error) {
	ttl := spec.DefaultTTL
	if len(ttlOverride) > 0 && ttlOverride[0] > 0 {
		ttl = ttlOverride[0]
	}
	if spec.Name == "" {
		return nil, false, fmt.Errorf("lock spec name is empty")
	}
	if ttl <= 0 {
		return nil, false, fmt.Errorf("lock spec ttl must be greater than 0")
	}
	return m.Acquire(ctx, spec.Identity(key), ttl)
}

// Release 释放一个已获取的具体锁租约。
func (m *Manager) Release(ctx context.Context, identity locklease.Identity, lease *locklease.Lease) error {
	if m == nil || lease == nil || lease.Key == "" || lease.Token == "" {
		return nil
	}
	leaseKey, err := redislease.NewLeaseKey(lease.Key)
	if err != nil {
		return err
	}
	token, err := redislease.NewLeaseToken(lease.Token)
	if err != nil {
		return err
	}
	return m.service.Release(ctx, redislease.Lease{
		Key:   leaseKey,
		Token: token,
		Owner: &redislease.LeaseOwner{Label: identity.Name},
	})
}

// ReleaseSpec 根据语义锁规格释放租约。
func (m *Manager) ReleaseSpec(ctx context.Context, spec locklease.Spec, key string, lease *locklease.Lease) error {
	if spec.Name == "" {
		return fmt.Errorf("lock spec name is empty")
	}
	return m.Release(ctx, spec.Identity(key), lease)
}

func (m *Manager) redisKey(identity locklease.Identity) string {
	key := identity.Name
	if identity.Key != "" {
		key = identity.Key
	}
	if m != nil && m.keyFunc != nil {
		key = m.keyFunc(key)
	}
	return key
}
