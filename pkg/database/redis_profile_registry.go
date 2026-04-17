package database

import (
	"context"
	"fmt"
	"sync"

	goredis "github.com/redis/go-redis/v9"
)

// NamedRedisRegistry manages a default Redis connection plus optional named profiles.
type NamedRedisRegistry struct {
	defaultConn *RedisConnection
	profiles    map[string]*RedisConnection
	mutex       sync.RWMutex
}

// NewNamedRedisRegistry creates a Redis registry with an optional default config
// and a set of optional named profile configs.
func NewNamedRedisRegistry(defaultConfig *RedisConfig, profiles map[string]*RedisConfig) *NamedRedisRegistry {
	r := &NamedRedisRegistry{
		profiles: make(map[string]*RedisConnection),
	}
	if defaultConfig != nil {
		r.defaultConn = NewRedisConnection(cloneRedisConfig(defaultConfig))
	}
	for name, cfg := range profiles {
		if name == "" || cfg == nil {
			continue
		}
		r.profiles[name] = NewRedisConnection(cloneRedisConfig(cfg))
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
	profiles := make(map[string]*RedisConnection, len(r.profiles))
	for name, conn := range r.profiles {
		profiles[name] = conn
	}
	r.mutex.RUnlock()

	if defaultConn != nil {
		if err := defaultConn.Connect(); err != nil {
			return fmt.Errorf("connect default redis: %w", err)
		}
	}
	for name, conn := range profiles {
		if err := conn.Connect(); err != nil {
			return fmt.Errorf("connect redis profile %q: %w", name, err)
		}
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
	profiles := make(map[string]*RedisConnection, len(r.profiles))
	for name, conn := range r.profiles {
		profiles[name] = conn
	}
	r.mutex.RUnlock()

	var lastErr error
	if defaultConn != nil {
		if err := defaultConn.Close(); err != nil {
			lastErr = fmt.Errorf("close default redis: %w", err)
		}
	}
	for name, conn := range profiles {
		if err := conn.Close(); err != nil {
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
	profiles := make(map[string]*RedisConnection, len(r.profiles))
	for name, conn := range r.profiles {
		profiles[name] = conn
	}
	r.mutex.RUnlock()

	if defaultConn != nil {
		if err := defaultConn.HealthCheck(ctx); err != nil {
			return fmt.Errorf("health check default redis: %w", err)
		}
	}
	for name, conn := range profiles {
		if err := conn.HealthCheck(ctx); err != nil {
			return fmt.Errorf("health check redis profile %q: %w", name, err)
		}
	}
	return nil
}

// GetConnection returns the named profile connection, falling back to default.
func (r *NamedRedisRegistry) GetConnection(name string) (*RedisConnection, error) {
	if r == nil {
		return nil, fmt.Errorf("redis registry is nil")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if name != "" {
		if conn, ok := r.profiles[name]; ok && conn != nil {
			return conn, nil
		}
	}
	if r.defaultConn != nil {
		return r.defaultConn, nil
	}
	return nil, fmt.Errorf("redis connection for profile %q not found", name)
}

// GetClient returns the named profile Redis client, falling back to default.
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
