package database

import (
	"context"
	"fmt"
	"log"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLConfig MySQL 数据库配置
type MySQLConfig struct {
	Host                  string        `json:"host" mapstructure:"host"`
	Username              string        `json:"username" mapstructure:"username"`
	Password              string        `json:"password" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections" mapstructure:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
	// Location controls DATETIME parsing. Empty preserves the historical Local
	// behavior for existing callers.
	Location string `json:"location" mapstructure:"location"`
	// SessionTimeZone is applied by the MySQL driver whenever it opens a pooled
	// connection. Empty leaves the server/session default unchanged.
	SessionTimeZone string `json:"session-time-zone" mapstructure:"session-time-zone"`
	Logger          logger.Interface
}

// MySQLConnection MySQL 连接实现
type MySQLConnection struct {
	config *MySQLConfig
	client *gorm.DB
}

// NewMySQLConnection 创建 MySQL 连接
func NewMySQLConnection(config *MySQLConfig) *MySQLConnection {
	return &MySQLConnection{
		config: config,
	}
}

// Type 返回数据库类型
func (m *MySQLConnection) Type() DatabaseType {
	return MySQL
}

// Connect 连接 MySQL 数据库
func (m *MySQLConnection) Connect() error {
	dsn, err := buildMySQLDSN(m.config)
	if err != nil {
		return err
	}

	// 打印 dsn
	log.Printf("Connecting to MySQL at %s/%s as user %s", m.config.Host, m.config.Database, m.config.Username)

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: m.config.Logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(m.config.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(m.config.MaxConnectionLifeTime)
	sqlDB.SetMaxIdleConns(m.config.MaxIdleConnections)

	m.client = db
	log.Printf("MySQL connected successfully to %s/%s", m.config.Host, m.config.Database)
	return nil
}

func buildMySQLDSN(config *MySQLConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("mysql config is nil")
	}
	locationName := config.Location
	if locationName == "" {
		locationName = "Local"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return "", fmt.Errorf("invalid mysql location %q: %w", locationName, err)
	}
	dsn := mysqldriver.NewConfig()
	dsn.User = config.Username
	dsn.Passwd = config.Password
	dsn.Net = "tcp"
	dsn.Addr = config.Host
	dsn.DBName = config.Database
	dsn.ParseTime = true
	dsn.Loc = location
	dsn.MultiStatements = true
	dsn.Params = map[string]string{"charset": "utf8"}
	if config.SessionTimeZone != "" {
		dsn.Params["time_zone"] = "'" + config.SessionTimeZone + "'"
	}
	return dsn.FormatDSN(), nil
}

// Close 关闭 MySQL 连接
func (m *MySQLConnection) Close() error {
	if m.client != nil {
		if sqlDB, err := m.client.DB(); err == nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// HealthCheck 检查 MySQL 连接是否健康
func (m *MySQLConnection) HealthCheck(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("MySQL client is nil")
	}

	if sqlDB, err := m.client.DB(); err == nil {
		return sqlDB.PingContext(ctx)
	}

	return fmt.Errorf("failed to get MySQL sql.DB for health check")
}

// GetClient 获取 MySQL 客户端
func (m *MySQLConnection) GetClient() interface{} {
	return m.client
}
