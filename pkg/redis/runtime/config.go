package runtime

import "time"

// Config 描述单个 Redis profile 的连接策略。
type Config struct {
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

// CloneConfig 克隆一份 Redis 配置。
func CloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	copyCfg := *cfg
	if len(cfg.Addrs) > 0 {
		copyCfg.Addrs = append([]string(nil), cfg.Addrs...)
	}
	return &copyCfg
}

// MergeConfig 使用 override 配置覆盖 base 配置。
func MergeConfig(base, override *Config) *Config {
	if override == nil {
		return CloneConfig(base)
	}
	if base == nil {
		return CloneConfig(override)
	}

	merged := CloneConfig(base)
	if merged == nil {
		merged = &Config{}
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

	// Database 为 0 也是合法值，因此始终尊重 override 的 DB 选择。
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

func secondsToDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
