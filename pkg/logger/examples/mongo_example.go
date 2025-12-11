package examples

import (
	"context"
	"time"

	"github.com/FangcunMount/component-base/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SetupMongoWithLogger 演示如何在 MongoDB 客户端中配置日志钩子
func SetupMongoWithLogger() (*mongo.Client, error) {
	// 方式一：使用简单参数创建
	// enabled: 是否启用日志
	// slowThreshold: 慢查询阈值
	mongoHook := logger.NewMongoHook(true, 200*time.Millisecond)

	// 配置 MongoDB 客户端选项
	clientOptions := options.Client().
		ApplyURI("mongodb://localhost:27017").
		SetMonitor(mongoHook.CommandMonitor()).     // 命令监控
		SetPoolMonitor(mongoHook.PoolMonitor()).    // 连接池监控
		SetServerMonitor(mongoHook.ServerMonitor()) // 服务器监控

	// 连接到 MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// SetupMongoWithCustomConfig 演示使用自定义配置
func SetupMongoWithCustomConfig() (*mongo.Client, error) {
	// 方式二：使用配置结构体
	config := logger.MongoHookConfig{
		// 是否启用日志记录
		Enabled: true,
		// 慢查询阈值（超过此时间会记录警告）
		SlowThreshold: 100 * time.Millisecond,
	}
	mongoHook := logger.NewMongoHookWithConfig(config)

	clientOptions := options.Client().
		ApplyURI("mongodb://localhost:27017").
		SetMonitor(mongoHook.CommandMonitor()).
		SetPoolMonitor(mongoHook.PoolMonitor()).
		SetServerMonitor(mongoHook.ServerMonitor())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// SetupMongoWithAuth 演示带认证的 MongoDB 连接
func SetupMongoWithAuth(username, password, host, database string) (*mongo.Client, error) {
	mongoHook := logger.NewMongoHookWithConfig(logger.DefaultMongoHookConfig())

	// 构建连接 URI
	uri := "mongodb://" + username + ":" + password + "@" + host + "/" + database

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(mongoHook.CommandMonitor()).
		SetPoolMonitor(mongoHook.PoolMonitor()).
		SetServerMonitor(mongoHook.ServerMonitor())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// PerformDatabaseOperations 演示数据库操作示例
func PerformDatabaseOperations(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := client.Database("testdb").Collection("users")

	// 插入文档
	doc := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	// 查询文档
	var result map[string]interface{}
	err = collection.FindOne(ctx, map[string]interface{}{"name": "John Doe"}).Decode(&result)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	return nil
}
