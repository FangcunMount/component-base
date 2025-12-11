package examples

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/logger"
)

// DemoProgramStartup 演示程序启动时的日志记录
func DemoProgramStartup() {
	// 场景 1: 程序启动日志
	logger.Default().Info("Application starting",
		log.String("version", "1.0.0"),
		log.String("env", "production"),
	)

	// 场景 2: 使用 WithField 链式调用
	appLogger := logger.Default().
		WithField("app", "user-service").
		WithField("instance", "server-01")

	appLogger.Info("Service initialized")
	appLogger.Info("Database connected")
}

// DemoBackgroundTask 演示后台任务/定时任务的日志记录
func DemoBackgroundTask() {
	taskLogger := logger.New(
		log.String("task", "data_sync"),
		log.String("module", "background"),
	)

	taskLogger.Info("Task started")
	time.Sleep(100 * time.Millisecond)
	taskLogger.Info("Processing data", log.Int("records", 1000))
	taskLogger.Info("Task completed", log.Duration("duration", 100*time.Millisecond))
}

// DemoAsyncWorker 演示独立的 goroutine（无 context）
func DemoAsyncWorker() {
	go func() {
		workerLogger := logger.New(
			log.String("worker", "background-worker-1"),
			log.String("type", "async"),
		)
		workerLogger.Info("Worker started")
		time.Sleep(50 * time.Millisecond)
		workerLogger.Info("Worker processing job", log.String("job_id", "job-123"))
		workerLogger.Info("Worker completed")
	}()

	// 等待 goroutine 完成
	time.Sleep(100 * time.Millisecond)
}

// DemoModuleLogger 演示模块级别的日志（可以声明为全局变量）
func DemoModuleLogger() {
	// 模块级别的 Logger
	dbLogger := logger.New(log.String("component", "database"))
	dbLogger.Info("Connection pool initialized", log.Int("max_connections", 100))
	dbLogger.Warn("Connection pool usage high", log.Float64("usage_percent", 85.5))
}

// DemoErrorHandling 演示错误处理（无 context）
func DemoErrorHandling() {
	err := performOperation()
	if err != nil {
		logger.Default().Error("Operation failed",
			log.String("operation", "data_processing"),
			log.Err(err),
		)
	}
}

// DemoSystemEvents 演示系统事件日志
func DemoSystemEvents() {
	systemLogger := logger.New(log.String("source", "system"))
	systemLogger.Info("Configuration loaded", log.String("config_file", "/etc/app/config.yaml"))
	systemLogger.Warn("Memory usage high", log.Float64("usage_mb", 1024.5))
}

// DemoAllScenarios 演示所有无 context 场景
func DemoAllScenarios() {
	// 程序启动
	DemoProgramStartup()

	// 后台任务
	DemoBackgroundTask()

	// 异步 Worker
	DemoAsyncWorker()

	// 模块日志
	DemoModuleLogger()

	// 错误处理
	DemoErrorHandling()

	// 系统事件
	DemoSystemEvents()

	logger.Default().Info("All scenarios completed")
}

func performOperation() error {
	// 模拟操作
	return nil
}
