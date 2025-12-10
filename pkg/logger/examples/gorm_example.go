package examples

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// SetupGormWithLogger 演示如何在 GORM 中配置日志适配器
func SetupGormWithLogger() (*gorm.DB, error) {
	dsn := "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"

	// 方式一：使用日志级别创建（推荐）
	// 日志级别：1=Silent, 2=Error, 3=Warn, 4=Info
	gormLogger := logger.NewGormLogger(4)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// SetupGormWithCustomConfig 演示使用自定义配置
func SetupGormWithCustomConfig() (*gorm.DB, error) {
	dsn := "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"

	// 方式二：使用自定义配置
	config := logger.GormConfig{
		// 慢查询阈值（超过此时间会记录警告）
		SlowThreshold: 200 * time.Millisecond,
		// 是否使用彩色输出（仅开发环境）
		Colorful: false,
		// 日志级别
		LogLevel: gormlogger.Info,
	}
	gormLogger := logger.NewGormLoggerWithConfig(config)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// SetupGormDynamicLogLevel 演示动态调整日志级别
func SetupGormDynamicLogLevel(db *gorm.DB) {
	// 在调试时临时启用详细日志
	db.Logger = db.Logger.LogMode(gormlogger.Info)

	// 在生产环境只记录错误
	db.Logger = db.Logger.LogMode(gormlogger.Error)

	// 完全禁用日志
	db.Logger = db.Logger.LogMode(gormlogger.Silent)
}

// GormLoggerOutput 说明 GORM 日志适配器产生的日志格式
//
// 正常 SQL 执行日志（Debug 级别）：
//
//	{
//	    "level": "debug",
//	    "ts": "2025-12-10T10:00:00.100Z",
//	    "msg": "GORM trace",
//	    "caller": "repository/user_repo.go:45",
//	    "sql": "SELECT * FROM users WHERE id = ?",
//	    "elapsed_ms": 5.2,
//	    "rows": 1,
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
//
// 慢查询警告日志：
//
//	{
//	    "level": "warn",
//	    "ts": "2025-12-10T10:00:00.500Z",
//	    "msg": "GORM slow query",
//	    "caller": "repository/user_repo.go:78",
//	    "sql": "SELECT * FROM orders WHERE user_id = ? AND status IN (?)",
//	    "elapsed_ms": 350.5,
//	    "rows": 156,
//	    "event": "slow_query",
//	    "slow_threshold": "200ms",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
//
// SQL 执行错误日志：
//
//	{
//	    "level": "error",
//	    "ts": "2025-12-10T10:00:00.200Z",
//	    "msg": "GORM trace failed",
//	    "caller": "repository/user_repo.go:92",
//	    "sql": "INSERT INTO users (username, email) VALUES (?, ?)",
//	    "elapsed_ms": 10.5,
//	    "rows": 0,
//	    "error": "Error 1062: Duplicate entry 'john@example.com' for key 'email'",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001"
//	}
func GormLoggerOutput() {}
