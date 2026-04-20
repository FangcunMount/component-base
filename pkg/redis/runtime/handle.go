package runtime

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"

	goredis "github.com/redis/go-redis/v9"
)

// Handle 封装了 Redis 客户端及其解析后的 profile 元数据。
type Handle struct {
	profile string
	config  *Config
	client  goredis.UniversalClient
}

// NewHandle 为指定 profile 创建运行时句柄。
func NewHandle(profile string, config *Config) *Handle {
	return &Handle{
		profile: profile,
		config:  CloneConfig(config),
	}
}

// Profile 返回句柄绑定的 profile 名称。
func (h *Handle) Profile() string {
	if h == nil {
		return ""
	}
	return h.profile
}

// Config 返回当前 profile 配置的克隆副本。
func (h *Handle) Config() *Config {
	if h == nil {
		return nil
	}
	return CloneConfig(h.config)
}

// Client 返回底层 go-redis 客户端。
func (h *Handle) Client() goredis.UniversalClient {
	if h == nil {
		return nil
	}
	return h.client
}

// Connect 初始化底层 Redis 客户端。
func (h *Handle) Connect() error {
	if h == nil {
		return fmt.Errorf("redis handle is nil")
	}
	if h.config == nil {
		return fmt.Errorf("redis config is nil")
	}

	addrs := h.addresses()
	log.Printf("Connecting to Redis at %v", addrs)

	poolTimeout := secondsToDuration(h.config.PoolTimeout)
	dialTimeout := secondsToDuration(h.config.DialTimeout)
	readTimeout := secondsToDuration(h.config.ReadTimeout)
	writeTimeout := secondsToDuration(h.config.WriteTimeout)
	connMaxIdleTime := secondsToDuration(h.config.Timeout)

	var tlsConfig *tls.Config
	if h.config.UseSSL {
		tlsConfig = &tls.Config{InsecureSkipVerify: h.config.SSLInsecureSkipVerify}
	}

	var client goredis.UniversalClient
	if h.config.EnableCluster {
		client = goredis.NewClusterClient(&goredis.ClusterOptions{
			Addrs:           addrs,
			Username:        h.config.Username,
			Password:        h.config.Password,
			PoolSize:        h.config.MaxActive,
			MaxIdleConns:    h.config.MaxIdle,
			MinIdleConns:    h.config.MinIdleConns,
			PoolTimeout:     poolTimeout,
			DialTimeout:     dialTimeout,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ConnMaxIdleTime: connMaxIdleTime,
			MaxActiveConns:  h.config.MaxActive,
			TLSConfig:       tlsConfig,
		})
	} else {
		client = goredis.NewClient(&goredis.Options{
			Addr:            addrs[0],
			Username:        h.config.Username,
			Password:        h.config.Password,
			DB:              h.config.Database,
			PoolSize:        h.config.MaxActive,
			MaxIdleConns:    h.config.MaxIdle,
			MinIdleConns:    h.config.MinIdleConns,
			PoolTimeout:     poolTimeout,
			DialTimeout:     dialTimeout,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			MaxActiveConns:  h.config.MaxActive,
			ConnMaxIdleTime: connMaxIdleTime,
			TLSConfig:       tlsConfig,
		})
	}

	if err := client.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	h.client = client
	log.Printf("Redis connected successfully to %v", addrs)
	return nil
}

// Close 关闭底层 Redis 客户端。
func (h *Handle) Close() error {
	if h == nil || h.client == nil {
		return nil
	}
	return h.client.Close()
}

// HealthCheck 检查底层 Redis 客户端健康状态。
func (h *Handle) HealthCheck(ctx context.Context) error {
	if h == nil {
		return fmt.Errorf("redis handle is nil")
	}
	if h.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if err := h.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}
	return nil
}

func (h *Handle) addresses() []string {
	if h == nil || h.config == nil {
		return nil
	}
	if len(h.config.Addrs) > 0 {
		return append([]string(nil), h.config.Addrs...)
	}
	return []string{fmt.Sprintf("%s:%d", h.config.Host, h.config.Port)}
}
