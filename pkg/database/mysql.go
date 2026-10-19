package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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
	TLSMode         string `json:"tls-mode" mapstructure:"tls-mode"`
	TLSCAFile       string `json:"tls-ca-file" mapstructure:"tls-ca-file"`
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
	connector, err := NewMySQLConnector(m.config)
	if err != nil {
		return err
	}

	// 打印 dsn
	log.Printf("Connecting to MySQL at %s/%s as user %s", m.config.Host, m.config.Database, m.config.Username)

	pool := sql.OpenDB(connector)
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: pool}), &gorm.Config{
		Logger: m.config.Logger,
	})
	if err != nil {
		_ = pool.Close()
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
	dsn, err := mysqlDriverConfig(config)
	if err != nil {
		return "", err
	}
	if dsn.TLS != nil {
		return "", fmt.Errorf("TLS configuration cannot be serialized as a DSN; use NewMySQLConnector")
	}
	return dsn.FormatDSN(), nil
}

// NewMySQLConnector preserves TLS verification on every pooled connection.
// Passing a typed Config avoids losing the CA context during DSN serialization.
func NewMySQLConnector(config *MySQLConfig) (driver.Connector, error) {
	dsn, err := mysqlDriverConfig(config)
	if err != nil {
		return nil, err
	}
	return mysqldriver.NewConnector(dsn)
}

func mysqlDriverConfig(config *MySQLConfig) (*mysqldriver.Config, error) {
	if config == nil {
		return nil, fmt.Errorf("mysql config is nil")
	}
	locationName := config.Location
	if locationName == "" {
		locationName = "Local"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql location %q: %w", locationName, err)
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
	dsn.Params = map[string]string{}
	if err := dsn.Apply(mysqldriver.Charset("utf8", "")); err != nil {
		return nil, err
	}
	if config.SessionTimeZone != "" {
		dsn.Params["time_zone"] = "'" + config.SessionTimeZone + "'"
	}
	if err := ConfigureMySQLTLS(dsn, config.TLSMode, config.TLSCAFile); err != nil {
		return nil, err
	}
	return dsn, nil
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

// ConfigureMySQLTLS applies the same certificate policy to maintenance clients
// that already own a driver Config (including their timeouts and DSN options).
func ConfigureMySQLTLS(config *mysqldriver.Config, mode, caFile string) error {
	if config == nil {
		return fmt.Errorf("mysql driver config is nil")
	}
	config.TLS = nil
	config.TLSConfig = ""
	mode = strings.ToUpper(strings.TrimSpace(mode))
	switch mode {
	case "", "DISABLED":
	case "VERIFY_IDENTITY":
		if caFile == "" {
			return fmt.Errorf("mysql TLS CA file is required for VERIFY_IDENTITY")
		}
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return fmt.Errorf("read mysql TLS CA: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return fmt.Errorf("mysql TLS CA file contains no certificates")
		}
		host, _, err := net.SplitHostPort(config.Addr)
		if err != nil || host == "" {
			return fmt.Errorf("mysql VERIFY_IDENTITY requires host:port")
		}
		config.TLS = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: host}
	default:
		return fmt.Errorf("unsupported mysql TLS mode %q", mode)
	}
	return nil
}
