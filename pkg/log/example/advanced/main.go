// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示日志模块的高级功能组合使用
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	// 创建日志目录
	logDir := "/tmp/component-base-advanced-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("日志模块高级功能综合示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置日志：结合时间轮转 + 分级输出 + 彩色显示
	opts := log.NewOptions()

	// 基础配置
	opts.Level = "debug"
	opts.EnableColor = true
	opts.Format = "console"

	// 启用按时间轮转（按天）
	opts.EnableTimeRotation = true
	opts.TimeRotationFormat = "2006-01-02"
	opts.MaxAge = 7

	// 启用分级输出
	opts.EnableLevelOutput = true
	opts.LevelOutputMode = "above"

	// 为不同级别配置不同的输出
	opts.LevelOutputPaths = map[string][]string{
		"debug": []string{"stdout"}, // DEBUG 只输出到控制台
		"info": []string{
			"stdout",
			filepath.Join(logDir, "app.log"), // INFO 及以上输出到 app.log
		},
		"warn": []string{
			filepath.Join(logDir, "warn.log"), // WARN 及以上输出到 warn.log
		},
		"error": []string{
			filepath.Join(logDir, "error.log"), // ERROR 单独输出
		},
	}

	log.Init(opts)
	defer log.Flush()

	fmt.Println("配置总结:")
	fmt.Println("1. 基础设置:")
	fmt.Printf("   - 日志级别: %s\n", opts.Level)
	fmt.Printf("   - 彩色输出: %v\n", opts.EnableColor)
	fmt.Printf("   - 日志格式: %s\n", opts.Format)
	fmt.Println()
	fmt.Println("2. 时间轮转:")
	fmt.Printf("   - 启用: %v\n", opts.EnableTimeRotation)
	fmt.Printf("   - 格式: %s (按天)\n", opts.TimeRotationFormat)
	fmt.Printf("   - 保留: %d 天\n", opts.MaxAge)
	fmt.Println()
	fmt.Println("3. 分级输出:")
	fmt.Printf("   - 启用: %v\n", opts.EnableLevelOutput)
	fmt.Printf("   - 模式: %s\n", opts.LevelOutputMode)
	fmt.Println("   - DEBUG -> 控制台")
	fmt.Println("   - INFO  -> 控制台 + app.YYYY-MM-DD.log")
	fmt.Println("   - WARN  -> warn.YYYY-MM-DD.log")
	fmt.Println("   - ERROR -> error.YYYY-MM-DD.log")
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("开始模拟应用运行...")
	fmt.Println("========================================")
	fmt.Println()

	// 模拟应用启动
	log.Info("应用启动",
		log.String("version", "v2.0.0"),
		log.String("env", "production"),
		log.String("host", "server-01"),
	)

	// 模拟调试信息（只在控制台显示）
	log.Debug("加载配置文件",
		log.String("path", "/etc/app/config.yaml"),
		log.Int("items", 25),
	)

	log.Debug("初始化数据库连接池",
		log.Int("max_connections", 100),
		log.Int("idle_connections", 10),
	)

	// 模拟正常业务日志（控制台 + app.log）
	for i := 1; i <= 5; i++ {
		log.Info("处理用户请求",
			log.Int("request_id", 1000+i),
			log.String("user", fmt.Sprintf("user_%d", i)),
			log.String("action", "create_order"),
		)
	}

	// 模拟警告（warn.log）
	log.Warn("数据库连接池使用率较高",
		log.Float64("usage", 0.85),
		log.Int("active", 85),
		log.Int("max", 100),
	)

	log.Warn("API 响应时间超过阈值",
		log.String("endpoint", "/api/v1/products"),
		log.Int("response_time_ms", 1200),
		log.Int("threshold_ms", 1000),
	)

	// 模拟错误（error.log）
	log.Error("Redis 连接失败",
		log.String("host", "redis-cluster:6379"),
		log.String("error", "connection refused"),
		log.Int("retry_count", 3),
	)

	log.Error("支付接口调用失败",
		log.String("order_id", "ORD-2025-11-01-001"),
		log.String("payment_method", "alipay"),
		log.String("error", "timeout"),
	)

	// 更多业务日志
	log.Info("缓存更新",
		log.String("key", "user:sessions"),
		log.Int("count", 1523),
	)

	log.Info("定时任务执行",
		log.String("task", "cleanup_expired_data"),
		log.String("status", "success"),
		log.Int("deleted_rows", 345),
	)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("应用运行模拟完成")
	fmt.Println("========================================")
	fmt.Println()

	// 等待日志完全写入
	time.Sleep(100 * time.Millisecond)

	// 显示生成的日志文件
	fmt.Println("生成的日志文件:")
	today := time.Now().Format("2006-01-02")

	expectedFiles := []string{
		fmt.Sprintf("app.%s.log", today),
		fmt.Sprintf("warn.%s.log", today),
		fmt.Sprintf("error.%s.log", today),
	}

	for _, filename := range expectedFiles {
		fullPath := filepath.Join(logDir, filename)
		if info, err := os.Stat(fullPath); err == nil {
			content, _ := os.ReadFile(fullPath)
			lines := 0
			for _, b := range content {
				if b == '\n' {
					lines++
				}
			}
			fmt.Printf("  ✓ %s\n", filename)
			fmt.Printf("    - 大小: %d 字节\n", info.Size())
			fmt.Printf("    - 行数: %d\n", lines)
		} else {
			fmt.Printf("  ✗ %s (未生成)\n", filename)
		}
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("功能说明")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("✨ 已启用的功能:")
	fmt.Println()
	fmt.Println("1. 彩色日志输出")
	fmt.Println("   - 控制台输出带颜色，便于快速识别日志级别")
	fmt.Println()
	fmt.Println("2. 按天轮转")
	fmt.Println("   - 每天自动创建新的日志文件")
	fmt.Println("   - 文件名包含日期：app.YYYY-MM-DD.log")
	fmt.Println("   - 自动清理过期日志（7天前）")
	fmt.Println()
	fmt.Println("3. 日志分级输出")
	fmt.Println("   - DEBUG: 仅控制台，不保存到文件")
	fmt.Println("   - INFO:  控制台 + app 日志文件")
	fmt.Println("   - WARN:  warn 专用文件（包含 ERROR）")
	fmt.Println("   - ERROR: error 专用文件")
	fmt.Println()
	fmt.Println("4. 结构化日志")
	fmt.Println("   - 使用键值对记录上下文信息")
	fmt.Println("   - 便于日志分析和查询")
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("适用场景")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("这种配置特别适合：")
	fmt.Println("• 生产环境的 Web 服务")
	fmt.Println("• 需要长期保留日志的应用")
	fmt.Println("• 需要快速定位错误的场景")
	fmt.Println("• 需要按天统计和分析的业务")
	fmt.Println("========================================")
}
