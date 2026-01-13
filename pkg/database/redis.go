package database

import (
	"context"
	"fmt"
	"log"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// RedisConfig Redis 数据库配置
type RedisConfig struct {
	Host                  string   `json:"host" mapstructure:"host"`
	Port                  int      `json:"port" mapstructure:"port"`
	Addrs                 []string `json:"addrs" mapstructure:"addrs"`
	Username              string   `json:"username" mapstructure:"username"`
	Password              string   `json:"password" mapstructure:"password"`
	Database              int      `json:"database" mapstructure:"database"`
	MaxIdle               int      `json:"max-idle" mapstructure:"max-idle"`
	MaxActive             int      `json:"max-active" mapstructure:"max-active"`
	Timeout               int      `json:"timeout" mapstructure:"timeout"`
	MinIdleConns          int      `json:"min-idle-conns" mapstructure:"min-idle-conns"`
	PoolTimeout           int      `json:"pool-timeout" mapstructure:"pool-timeout"`
	DialTimeout           int      `json:"dial-timeout" mapstructure:"dial-timeout"`
	ReadTimeout           int      `json:"read-timeout" mapstructure:"read-timeout"`
	WriteTimeout          int      `json:"write-timeout" mapstructure:"write-timeout"`
	EnableCluster         bool     `json:"enable-cluster" mapstructure:"enable-cluster"`
	UseSSL                bool     `json:"use-ssl" mapstructure:"use-ssl"`
	SSLInsecureSkipVerify bool     `json:"ssl-insecure-skip-verify" mapstructure:"ssl-insecure-skip-verify"`
}

// RedisConnection Redis 连接实现
type RedisConnection struct {
	config *RedisConfig
	client redis.UniversalClient
}

// NewRedisConnection 创建 Redis 连接
func NewRedisConnection(config *RedisConfig) *RedisConnection {
	return &RedisConnection{
		config: config,
	}
}

// Type 返回数据库类型
func (r *RedisConnection) Type() DatabaseType {
	return Redis
}

// Connect 连接 Redis 数据库
func (r *RedisConnection) Connect() error {
	var addrs []string
	if len(r.config.Addrs) > 0 {
		addrs = r.config.Addrs
	} else {
		addr := fmt.Sprintf("%s:%d", r.config.Host, r.config.Port)
		addrs = []string{addr}
	}

	// 打印连接信息
	log.Printf("Connecting to Redis at %v", addrs)

	poolTimeout := secondsToDuration(r.config.PoolTimeout)
	dialTimeout := secondsToDuration(r.config.DialTimeout)
	readTimeout := secondsToDuration(r.config.ReadTimeout)
	writeTimeout := secondsToDuration(r.config.WriteTimeout)
	connMaxIdleTime := secondsToDuration(r.config.Timeout)

	// 创建 Redis 客户端
	var client redis.UniversalClient

	if r.config.EnableCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           addrs,
			Username:        r.config.Username,
			Password:        r.config.Password,
			PoolSize:        r.config.MaxActive,
			MaxIdleConns:    r.config.MaxIdle,
			MinIdleConns:    r.config.MinIdleConns,
			PoolTimeout:     poolTimeout,
			DialTimeout:     dialTimeout,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ConnMaxIdleTime: connMaxIdleTime,
			MaxActiveConns:  r.config.MaxActive,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:            addrs[0],
			Username:        r.config.Username,
			Password:        r.config.Password,
			DB:              r.config.Database,
			PoolSize:        r.config.MaxActive,
			MaxIdleConns:    r.config.MaxIdle,
			MinIdleConns:    r.config.MinIdleConns,
			PoolTimeout:     poolTimeout,
			DialTimeout:     dialTimeout,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			MaxActiveConns:  r.config.MaxActive,
			ConnMaxIdleTime: connMaxIdleTime,
		})
	}

	// 测试连接
	if err := client.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	r.client = client
	log.Printf("Redis connected successfully to %v", addrs)
	return nil
}

// Close 关闭 Redis 连接
func (r *RedisConnection) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// HealthCheck 检查 Redis 连接是否健康
func (r *RedisConnection) HealthCheck(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}
	return nil
}

// GetClient 获取 Redis 客户端
func (r *RedisConnection) GetClient() interface{} {
	return r.client
}

func secondsToDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}
