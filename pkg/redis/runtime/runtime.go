package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/redis/observability"
)

// ProfileState 表示已配置 profile 的健康状态。
type ProfileState string

const (
	ProfileStateMissing     ProfileState = "missing"
	ProfileStateAvailable   ProfileState = "available"
	ProfileStateUnavailable ProfileState = "unavailable"
)

// ProfileStatus 描述 profile 当前的可用性状态。
type ProfileStatus struct {
	Name        string
	State       ProfileState
	Err         error
	NextRetryAt time.Time
}

// Binding 描述资源应绑定到哪个 Redis profile。
type Binding struct {
	Profile string
}

// NewBinding 创建 profile 绑定值对象。
func NewBinding(profile string) Binding {
	return Binding{Profile: profile}
}

// RedisProfile 描述一个已配置的命名 profile。
type RedisProfile struct {
	Name   string
	Config Config
	Status ProfileStatus
}

// Runtime 定义稳定的 Redis 运行时门面接口。
type Runtime interface {
	HasConnections() bool
	Connect() error
	Close() error
	HealthCheck(ctx context.Context) error
	Default() (*Handle, error)
	ByProfile(name string) (*Handle, error)
	Bind(name string) (*Handle, error)
	Client(name string) (goredis.UniversalClient, error)
	Status(name string) ProfileStatus
	Statuses() map[string]ProfileStatus
	Profiles() []RedisProfile
}

type Option func(*Registry)

// WithFallbackOnMissing 控制缺失命名 profile 时是否回退到默认 profile。
func WithFallbackOnMissing(enabled bool) Option {
	return func(r *Registry) {
		r.fallbackOnMissing = enabled
	}
}

// WithProfileObserver 注册 profile 状态变化观测器。
func WithProfileObserver(observer observability.ProfileObserver) Option {
	return func(r *Registry) {
		if observer != nil {
			r.profileObserver = observer
		}
	}
}

type namedProfile struct {
	handle      *Handle
	lastErr     error
	nextRetryAt time.Time
	retryDelay  time.Duration
}

const (
	profileInitialRetryDelay = 30 * time.Second
	profileMaxRetryDelay     = 5 * time.Minute
)

// Registry 基于默认 profile 与命名 profile 实现 Runtime 门面。
type Registry struct {
	defaultHandle     *Handle
	profiles          map[string]*namedProfile
	fallbackOnMissing bool
	profileObserver   observability.ProfileObserver
	mutex             sync.RWMutex
}

// New 创建带可选默认 profile 与命名 profile 的运行时注册表。
func New(defaultConfig *Config, profiles map[string]*Config, opts ...Option) *Registry {
	r := &Registry{
		profiles:          make(map[string]*namedProfile),
		fallbackOnMissing: true,
		profileObserver:   observability.NopProfileObserver{},
	}
	for _, opt := range opts {
		opt(r)
	}
	if defaultConfig != nil {
		r.defaultHandle = NewHandle("", defaultConfig)
	}
	for name, cfg := range profiles {
		if name == "" || cfg == nil {
			continue
		}
		r.profiles[name] = &namedProfile{
			handle: NewHandle(name, MergeConfig(defaultConfig, cfg)),
		}
	}
	return r
}

// HasConnections 返回运行时是否至少存在一个已配置连接。
func (r *Registry) HasConnections() bool {
	if r == nil {
		return false
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.defaultHandle != nil || len(r.profiles) > 0
}

// Connect 初始化默认 profile 与命名 profile。
func (r *Registry) Connect() error {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defaultHandle := r.defaultHandle
	profiles := make(map[string]*namedProfile, len(r.profiles))
	for name, profile := range r.profiles {
		profiles[name] = profile
	}
	r.mutex.RUnlock()

	ctx := context.Background()
	if defaultHandle != nil {
		if err := defaultHandle.Connect(); err != nil {
			return fmt.Errorf("connect default redis: %w", err)
		}
	}
	for name, profile := range profiles {
		if profile == nil || profile.handle == nil {
			continue
		}
		if err := profile.handle.Connect(); err != nil {
			r.mutex.Lock()
			if current := r.profiles[name]; current != nil {
				current.markUnavailable(ctx, r.profileObserver, fmt.Errorf("connect redis profile %q: %w", name, err))
			}
			r.mutex.Unlock()
			continue
		}
		r.mutex.Lock()
		if current := r.profiles[name]; current != nil {
			current.markAvailable(ctx, r.profileObserver)
		}
		r.mutex.Unlock()
	}
	return nil
}

// Close 关闭所有已初始化的 Redis 连接。
func (r *Registry) Close() error {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defaultHandle := r.defaultHandle
	profiles := make(map[string]*namedProfile, len(r.profiles))
	for name, profile := range r.profiles {
		profiles[name] = profile
	}
	r.mutex.RUnlock()

	var lastErr error
	if defaultHandle != nil {
		if err := defaultHandle.Close(); err != nil {
			lastErr = fmt.Errorf("close default redis: %w", err)
		}
	}
	for name, profile := range profiles {
		if profile == nil || profile.handle == nil {
			continue
		}
		if err := profile.handle.Close(); err != nil {
			lastErr = fmt.Errorf("close redis profile %q: %w", name, err)
		}
	}
	return lastErr
}

// HealthCheck 检查已配置 Redis 连接，并在可重试时尝试恢复。
func (r *Registry) HealthCheck(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defaultHandle := r.defaultHandle
	profiles := make(map[string]*namedProfile, len(r.profiles))
	for name, profile := range r.profiles {
		profiles[name] = profile
	}
	r.mutex.RUnlock()

	if defaultHandle != nil {
		if err := defaultHandle.HealthCheck(ctx); err != nil {
			return fmt.Errorf("health check default redis: %w", err)
		}
	}
	for name, profile := range profiles {
		if profile == nil || profile.handle == nil {
			continue
		}
		if profile.lastErr != nil {
			if !profile.retryDue(time.Now()) {
				continue
			}
			if err := profile.reconnect(); err != nil {
				r.mutex.Lock()
				if current := r.profiles[name]; current != nil {
					current.markUnavailable(ctx, r.profileObserver, fmt.Errorf("reconnect redis profile %q: %w", name, err))
				}
				r.mutex.Unlock()
				continue
			}
			r.mutex.Lock()
			if current := r.profiles[name]; current != nil {
				current.markAvailable(ctx, r.profileObserver)
			}
			r.mutex.Unlock()
			continue
		}
		if err := profile.handle.HealthCheck(ctx); err != nil {
			r.mutex.Lock()
			if current := r.profiles[name]; current != nil {
				current.markUnavailable(ctx, r.profileObserver, fmt.Errorf("health check redis profile %q: %w", name, err))
			}
			r.mutex.Unlock()
		}
	}
	return nil
}

// Default 返回默认 Redis 句柄。
func (r *Registry) Default() (*Handle, error) {
	if r == nil {
		return nil, fmt.Errorf("redis runtime is nil")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if r.defaultHandle == nil {
		return nil, fmt.Errorf("default redis profile is not configured")
	}
	return r.defaultHandle, nil
}

// ByProfile 返回精确匹配的命名 profile 句柄。
// 缺失或不可用的 profile 会直接返回错误。
func (r *Registry) ByProfile(name string) (*Handle, error) {
	if r == nil {
		return nil, fmt.Errorf("redis runtime is nil")
	}
	if name == "" {
		return r.Default()
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	profile, ok := r.profiles[name]
	if !ok || profile == nil || profile.handle == nil {
		return nil, fmt.Errorf("redis profile %q not found", name)
	}
	if profile.lastErr != nil {
		return nil, fmt.Errorf("redis profile %q is unavailable: %w", name, profile.lastErr)
	}
	return profile.handle, nil
}

// Bind 根据绑定策略解析到命名或默认 Redis 句柄。
func (r *Registry) Bind(name string) (*Handle, error) {
	if r == nil {
		return nil, fmt.Errorf("redis runtime is nil")
	}
	if name == "" {
		return r.Default()
	}
	handle, err := r.ByProfile(name)
	if err == nil {
		return handle, nil
	}
	if r.Status(name).State == ProfileStateMissing && r.fallbackOnMissing {
		return r.Default()
	}
	return nil, err
}

// Client 返回绑定解析后的 Redis 客户端。
func (r *Registry) Client(name string) (goredis.UniversalClient, error) {
	handle, err := r.Bind(name)
	if err != nil {
		return nil, err
	}
	if handle == nil || handle.Client() == nil {
		return nil, fmt.Errorf("redis client for profile %q is unavailable", name)
	}
	return handle.Client(), nil
}

// Status 返回指定命名 profile 的当前状态。
func (r *Registry) Status(name string) ProfileStatus {
	status := ProfileStatus{Name: name, State: ProfileStateMissing}
	if r == nil || name == "" {
		return status
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	profile, ok := r.profiles[name]
	if !ok || profile == nil {
		return status
	}
	if profile.lastErr != nil {
		return ProfileStatus{
			Name:        name,
			State:       ProfileStateUnavailable,
			Err:         profile.lastErr,
			NextRetryAt: profile.nextRetryAt,
		}
	}
	return ProfileStatus{Name: name, State: ProfileStateAvailable}
}

// Statuses 返回所有已配置命名 profile 的状态快照。
func (r *Registry) Statuses() map[string]ProfileStatus {
	if r == nil {
		return nil
	}
	r.mutex.RLock()
	names := make([]string, 0, len(r.profiles))
	for name := range r.profiles {
		names = append(names, name)
	}
	r.mutex.RUnlock()

	statuses := make(map[string]ProfileStatus, len(names))
	for _, name := range names {
		statuses[name] = r.Status(name)
	}
	return statuses
}

// Profiles 返回已配置命名 profile 及其状态快照。
func (r *Registry) Profiles() []RedisProfile {
	if r == nil {
		return nil
	}
	r.mutex.RLock()
	profiles := make([]RedisProfile, 0, len(r.profiles))
	for name, profile := range r.profiles {
		if profile == nil || profile.handle == nil {
			continue
		}
		cfg := profile.handle.Config()
		status := ProfileStatus{Name: name, State: ProfileStateAvailable}
		if profile.lastErr != nil {
			status.State = ProfileStateUnavailable
			status.Err = profile.lastErr
			status.NextRetryAt = profile.nextRetryAt
		}
		profiles = append(profiles, RedisProfile{
			Name:   name,
			Config: *cfg,
			Status: status,
		})
	}
	r.mutex.RUnlock()
	return profiles
}

func (p *namedProfile) markAvailable(ctx context.Context, observer observability.ProfileObserver) {
	if p == nil {
		return
	}
	p.lastErr = nil
	p.nextRetryAt = time.Time{}
	p.retryDelay = 0
	observer.OnProfileStatus(ctx, observability.ProfileEvent{
		Profile: p.handle.Profile(),
		State:   string(ProfileStateAvailable),
	})
}

func (p *namedProfile) markUnavailable(ctx context.Context, observer observability.ProfileObserver, err error) {
	if p == nil {
		return
	}
	p.lastErr = err
	if p.retryDelay <= 0 {
		p.retryDelay = profileInitialRetryDelay
	} else {
		p.retryDelay *= 2
		if p.retryDelay > profileMaxRetryDelay {
			p.retryDelay = profileMaxRetryDelay
		}
	}
	p.nextRetryAt = time.Now().Add(p.retryDelay)
	observer.OnProfileStatus(ctx, observability.ProfileEvent{
		Profile:     p.handle.Profile(),
		State:       string(ProfileStateUnavailable),
		Err:         err,
		NextRetryAt: p.nextRetryAt,
	})
}

func (p *namedProfile) retryDue(now time.Time) bool {
	if p == nil || p.lastErr == nil {
		return false
	}
	if p.nextRetryAt.IsZero() {
		return true
	}
	return !now.Before(p.nextRetryAt)
}

func (p *namedProfile) reconnect() error {
	if p == nil || p.handle == nil {
		return fmt.Errorf("redis profile handle is nil")
	}
	_ = p.handle.Close()
	return p.handle.Connect()
}
