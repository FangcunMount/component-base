package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RedisProfileState string

const (
	RedisProfileStateMissing     RedisProfileState = "missing"
	RedisProfileStateAvailable   RedisProfileState = "available"
	RedisProfileStateUnavailable RedisProfileState = "unavailable"
)

type RedisProfileStatus struct {
	Name        string
	State       RedisProfileState
	Err         error
	NextRetryAt time.Time
}

type namedRedisProfile struct {
	conn        *RedisConnection
	lastErr     error
	nextRetryAt time.Time
	retryDelay  time.Duration
}

const (
	redisProfileInitialRetryDelay = 30 * time.Second
	redisProfileMaxRetryDelay     = 5 * time.Minute
)

// NamedRedisRegistry manages a default Redis connection plus optional named profiles.
type NamedRedisRegistry struct {
	defaultConn *RedisConnection
	profiles    map[string]*namedRedisProfile
	mutex       sync.RWMutex
}

// NewNamedRedisRegistry creates a Redis registry with an optional default config
// and a set of optional named profile configs.
func NewNamedRedisRegistry(defaultConfig *RedisConfig, profiles map[string]*RedisConfig) *NamedRedisRegistry {
	r := &NamedRedisRegistry{
		profiles: make(map[string]*namedRedisProfile),
	}
	if defaultConfig != nil {
		r.defaultConn = NewRedisConnection(cloneRedisConfig(defaultConfig))
	}
	for name, cfg := range profiles {
		if name == "" || cfg == nil {
			continue
		}
		r.profiles[name] = &namedRedisProfile{
			conn: NewRedisConnection(mergeRedisConfig(defaultConfig, cfg)),
		}
	}
	return r
}

// HasConnections reports whether the registry has at least one configured connection.
func (r *NamedRedisRegistry) HasConnections() bool {
	if r == nil {
		return false
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.defaultConn != nil || len(r.profiles) > 0
}

// Connect initializes the default and named Redis connections.
func (r *NamedRedisRegistry) Connect() error {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defaultConn := r.defaultConn
	profiles := make(map[string]*namedRedisProfile, len(r.profiles))
	for name, profile := range r.profiles {
		profiles[name] = profile
	}
	r.mutex.RUnlock()

	if defaultConn != nil {
		if err := defaultConn.Connect(); err != nil {
			return fmt.Errorf("connect default redis: %w", err)
		}
	}
	for name, profile := range profiles {
		if profile == nil || profile.conn == nil {
			continue
		}
		if err := profile.conn.Connect(); err != nil {
			r.mutex.Lock()
			if current := r.profiles[name]; current != nil {
				current.markUnavailable(fmt.Errorf("connect redis profile %q: %w", name, err))
			}
			r.mutex.Unlock()
			continue
		}
		r.mutex.Lock()
		if current := r.profiles[name]; current != nil {
			current.markAvailable()
		}
		r.mutex.Unlock()
	}
	return nil
}

// Close closes every initialized Redis connection.
func (r *NamedRedisRegistry) Close() error {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defaultConn := r.defaultConn
	profiles := make(map[string]*namedRedisProfile, len(r.profiles))
	for name, profile := range r.profiles {
		profiles[name] = profile
	}
	r.mutex.RUnlock()

	var lastErr error
	if defaultConn != nil {
		if err := defaultConn.Close(); err != nil {
			lastErr = fmt.Errorf("close default redis: %w", err)
		}
	}
	for name, profile := range profiles {
		if profile == nil || profile.conn == nil {
			continue
		}
		if err := profile.conn.Close(); err != nil {
			lastErr = fmt.Errorf("close redis profile %q: %w", name, err)
		}
	}
	return lastErr
}

// HealthCheck validates the default and named Redis connections.
func (r *NamedRedisRegistry) HealthCheck(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defaultConn := r.defaultConn
	profiles := make(map[string]*namedRedisProfile, len(r.profiles))
	for name, profile := range r.profiles {
		profiles[name] = profile
	}
	r.mutex.RUnlock()

	if defaultConn != nil {
		if err := defaultConn.HealthCheck(ctx); err != nil {
			return fmt.Errorf("health check default redis: %w", err)
		}
	}
	for name, profile := range profiles {
		if profile == nil || profile.conn == nil {
			continue
		}
		if profile.lastErr != nil {
			if !profile.retryDue(time.Now()) {
				continue
			}
			if err := profile.reconnect(); err != nil {
				r.mutex.Lock()
				if current := r.profiles[name]; current != nil {
					current.markUnavailable(fmt.Errorf("reconnect redis profile %q: %w", name, err))
				}
				r.mutex.Unlock()
				continue
			}
			r.mutex.Lock()
			if current := r.profiles[name]; current != nil {
				current.markAvailable()
			}
			r.mutex.Unlock()
			continue
		}
		if err := profile.conn.HealthCheck(ctx); err != nil {
			r.mutex.Lock()
			if current := r.profiles[name]; current != nil {
				current.markUnavailable(fmt.Errorf("health check redis profile %q: %w", name, err))
			}
			r.mutex.Unlock()
		}
	}
	return nil
}

// ProfileStatus reports whether a named profile is missing, available, or unavailable.
func (r *NamedRedisRegistry) ProfileStatus(name string) RedisProfileStatus {
	status := RedisProfileStatus{Name: name, State: RedisProfileStateMissing}
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
		return RedisProfileStatus{
			Name:        name,
			State:       RedisProfileStateUnavailable,
			Err:         profile.lastErr,
			NextRetryAt: profile.nextRetryAt,
		}
	}
	return RedisProfileStatus{
		Name:  name,
		State: RedisProfileStateAvailable,
	}
}

// ProfileStatuses returns the current status of every configured named profile.
func (r *NamedRedisRegistry) ProfileStatuses() map[string]RedisProfileStatus {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	names := make([]string, 0, len(r.profiles))
	for name := range r.profiles {
		names = append(names, name)
	}
	r.mutex.RUnlock()

	statuses := make(map[string]RedisProfileStatus, len(names))
	for _, name := range names {
		statuses[name] = r.ProfileStatus(name)
	}
	return statuses
}

// GetConnection returns the named profile connection. Missing profiles fall back to default,
// while configured-but-unavailable profiles return an error.
func (r *NamedRedisRegistry) GetConnection(name string) (*RedisConnection, error) {
	if r == nil {
		return nil, fmt.Errorf("redis registry is nil")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if name != "" {
		if profile, ok := r.profiles[name]; ok {
			if profile == nil || profile.conn == nil {
				return nil, fmt.Errorf("redis profile %q is unavailable", name)
			}
			if profile.lastErr != nil {
				return nil, fmt.Errorf("redis profile %q is unavailable: %w", name, profile.lastErr)
			}
			return profile.conn, nil
		}
	}
	if r.defaultConn != nil {
		return r.defaultConn, nil
	}
	return nil, fmt.Errorf("redis connection for profile %q not found", name)
}

// GetClient returns the named profile Redis client. Missing profiles fall back to default,
// while configured-but-unavailable profiles return an error.
func (r *NamedRedisRegistry) GetClient(name string) (goredis.UniversalClient, error) {
	conn, err := r.GetConnection(name)
	if err != nil {
		return nil, err
	}
	client, ok := conn.GetClient().(goredis.UniversalClient)
	if !ok || client == nil {
		return nil, fmt.Errorf("redis client for profile %q is unavailable", name)
	}
	return client, nil
}

func cloneRedisConfig(cfg *RedisConfig) *RedisConfig {
	if cfg == nil {
		return nil
	}
	copyCfg := *cfg
	if len(cfg.Addrs) > 0 {
		copyCfg.Addrs = append([]string(nil), cfg.Addrs...)
	}
	return &copyCfg
}

func mergeRedisConfig(base, override *RedisConfig) *RedisConfig {
	if override == nil {
		return cloneRedisConfig(base)
	}
	if base == nil {
		return cloneRedisConfig(override)
	}

	merged := cloneRedisConfig(base)
	if merged == nil {
		merged = &RedisConfig{}
	}

	if override.Host != "" {
		merged.Host = override.Host
	}
	if override.Port != 0 {
		merged.Port = override.Port
	}
	if len(override.Addrs) > 0 {
		merged.Addrs = append([]string(nil), override.Addrs...)
	}
	if override.Username != "" {
		merged.Username = override.Username
	}
	if override.Password != "" {
		merged.Password = override.Password
	}

	// Database 0 is valid, so always honor the named profile's DB selection.
	merged.Database = override.Database

	if override.MaxIdle != 0 {
		merged.MaxIdle = override.MaxIdle
	}
	if override.MaxActive != 0 {
		merged.MaxActive = override.MaxActive
	}
	if override.Timeout != 0 {
		merged.Timeout = override.Timeout
	}
	if override.MinIdleConns != 0 {
		merged.MinIdleConns = override.MinIdleConns
	}
	if override.PoolTimeout != 0 {
		merged.PoolTimeout = override.PoolTimeout
	}
	if override.DialTimeout != 0 {
		merged.DialTimeout = override.DialTimeout
	}
	if override.ReadTimeout != 0 {
		merged.ReadTimeout = override.ReadTimeout
	}
	if override.WriteTimeout != 0 {
		merged.WriteTimeout = override.WriteTimeout
	}

	if override.EnableCluster {
		merged.EnableCluster = true
	}
	if override.UseSSL {
		merged.UseSSL = true
	}
	if override.SSLInsecureSkipVerify {
		merged.SSLInsecureSkipVerify = true
	}

	return merged
}

func (p *namedRedisProfile) markAvailable() {
	if p == nil {
		return
	}
	p.lastErr = nil
	p.nextRetryAt = time.Time{}
	p.retryDelay = 0
}

func (p *namedRedisProfile) markUnavailable(err error) {
	if p == nil {
		return
	}
	p.lastErr = err
	if p.retryDelay <= 0 {
		p.retryDelay = redisProfileInitialRetryDelay
	} else {
		p.retryDelay *= 2
		if p.retryDelay > redisProfileMaxRetryDelay {
			p.retryDelay = redisProfileMaxRetryDelay
		}
	}
	p.nextRetryAt = time.Now().Add(p.retryDelay)
}

func (p *namedRedisProfile) retryDue(now time.Time) bool {
	if p == nil || p.lastErr == nil {
		return false
	}
	if p.nextRetryAt.IsZero() {
		return true
	}
	return !now.Before(p.nextRetryAt)
}

func (p *namedRedisProfile) reconnect() error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("redis profile connection is nil")
	}
	_ = p.conn.Close()
	return p.conn.Connect()
}
