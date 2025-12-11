package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/FangcunMount/component-base/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoConfig MongoDB 数据库配置
type MongoConfig struct {
	// 分离的连接参数（推荐使用，便于通过环境变量配置）
	Host     string `json:"host,omitempty"     mapstructure:"host"`     // 主机地址，格式: host:port
	Username string `json:"username,omitempty" mapstructure:"username"` // 用户名
	Password string `json:"-"                  mapstructure:"password"` // 密码（不输出到JSON）
	Database string `json:"database,omitempty" mapstructure:"database"` // 数据库名

	UseSSL                   bool   `json:"use-ssl" mapstructure:"use-ssl"`
	SSLInsecureSkipVerify    bool   `json:"ssl-insecure-skip-verify" mapstructure:"ssl-insecure-skip-verify"`
	SSLAllowInvalidHostnames bool   `json:"ssl-allow-invalid-hostnames" mapstructure:"ssl-allow-invalid-hostnames"`
	SSLCAFile                string `json:"ssl-ca-file" mapstructure:"ssl-ca-file"`
	SSLPEMKeyfile            string `json:"ssl-pem-keyfile" mapstructure:"ssl-pem-keyfile"`

	// 日志配置
	EnableLogger  bool          `json:"enable-logger"  mapstructure:"enable-logger"`  // 是否启用日志
	SlowThreshold time.Duration `json:"slow-threshold" mapstructure:"slow-threshold"` // 慢查询阈值
}

// BuildURL 根据配置参数构建 MongoDB 连接 URL
func (c *MongoConfig) BuildURL() string {
	// 构建基础 URL
	scheme := "mongodb"
	if c.UseSSL {
		scheme = "mongodb+srv"
	}

	// 构建认证信息
	var userInfo string
	if c.Username != "" {
		if c.Password != "" {
			userInfo = fmt.Sprintf("%s:%s@", url.QueryEscape(c.Username), url.QueryEscape(c.Password))
		} else {
			userInfo = fmt.Sprintf("%s@", url.QueryEscape(c.Username))
		}
	}

	// 构建数据库路径
	dbPath := ""
	if c.Database != "" {
		dbPath = "/" + c.Database
	}

	return fmt.Sprintf("%s://%s%s%s", scheme, userInfo, c.Host, dbPath)
}

// MongoDBConnection MongoDB 连接实现
type MongoDBConnection struct {
	config *MongoConfig
	client *mongo.Client
}

// NewMongoDBConnection 创建 MongoDB 连接
func NewMongoDBConnection(config *MongoConfig) *MongoDBConnection {
	return &MongoDBConnection{
		config: config,
	}
}

// Type 返回数据库类型
func (m *MongoDBConnection) Type() DatabaseType {
	return MongoDB
}

// Connect 连接 MongoDB 数据库
func (m *MongoDBConnection) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 根据配置构建连接 URL
	mongoURL := m.config.BuildURL()

	// 创建连接选项
	clientOptions := options.Client().ApplyURI(mongoURL)

	// 设置连接超时
	clientOptions.SetConnectTimeout(5 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	// 如果启用日志，添加日志钩子
	if m.config.EnableLogger {
		slowThreshold := m.config.SlowThreshold
		if slowThreshold <= 0 {
			slowThreshold = 200 * time.Millisecond // 默认 200ms
		}

		mongoHook := logger.NewMongoHook(true, slowThreshold)
		clientOptions.SetMonitor(mongoHook.CommandMonitor())
		clientOptions.SetPoolMonitor(mongoHook.PoolMonitor())
		clientOptions.SetServerMonitor(mongoHook.ServerMonitor())
	}

	// 连接到MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// 测试连接
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	log.Printf("MongoDB connected successfully")
	return nil
}

// Close 关闭 MongoDB 连接
func (m *MongoDBConnection) Close() error {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return m.client.Disconnect(ctx)
	}
	return nil
}

// HealthCheck 检查 MongoDB 连接是否健康
func (m *MongoDBConnection) HealthCheck(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client is nil")
	}

	return m.client.Ping(ctx, readpref.Primary())
}

// GetClient 获取 MongoDB 客户端
func (m *MongoDBConnection) GetClient() interface{} {
	return m.client
}
