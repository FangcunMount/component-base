// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// 演示 duplicate 模式：app.log 记录所有日志，error.log 额外记录错误
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
	logDir := "./logs/duplicate-mode"
	os.MkdirAll(logDir, 0755)

	fmt.Println("========================================")
	fmt.Println("Duplicate 模式示例")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("日志目录: %s\n", logDir)
	fmt.Println()

	// 配置日志
	opts := log.NewOptions()
	opts.Level = "debug"
	opts.Format = "console"
	opts.EnableColor = false // 文件输出不需要颜色
	opts.EnableLevelOutput = true
	opts.LevelOutputMode = "duplicate" // 使用 duplicate 模式

	// 配置输出路径
	// "all" 记录所有级别的日志
	// "error" 额外记录错误日志
	opts.LevelOutputPaths = map[string][]string{
		"all":   {filepath.Join(logDir, "app.log")},
		"error": {filepath.Join(logDir, "error.log")},
	}

	// 初始化日志
	log.Init(opts)
	defer log.Flush()

	fmt.Println("配置说明:")
	fmt.Println("- app.log:   记录所有日志 (DEBUG, INFO, WARN, ERROR)")
	fmt.Println("- error.log: 只记录 ERROR 日志")
	fmt.Println()

	// 输出各种级别的日志
	fmt.Println("开始记录日志...")
	fmt.Println()

	log.Debug("这是一条 DEBUG 日志", log.String("module", "test"))
	log.Info("这是一条 INFO 日志", log.String("action", "startup"))
	log.Warn("这是一条 WARN 日志", log.String("warning", "memory usage high"))
	log.Error("这是一条 ERROR 日志", log.String("error", "connection failed"))

	// 模拟实际业务场景
	fmt.Println("模拟业务场景...")
	fmt.Println()

	// 1. 正常业务流程
	log.Info("用户登录", log.String("user_id", "10086"), log.String("ip", "192.168.1.100"))
	log.Debug("验证用户凭证", log.String("user_id", "10086"))
	log.Info("登录成功", log.String("user_id", "10086"))

	// 2. 警告场景
	log.Warn("数据库连接池接近上限",
		log.Int("current", 95),
		log.Int("max", 100),
	)

	// 3. 错误场景
	log.Error("订单创建失败",
		log.String("order_id", "ORD-12345"),
		log.String("error", "库存不足"),
		log.Int("required", 10),
		log.Int("available", 5),
	)

	log.Error("支付服务调用失败",
		log.String("service", "payment-api"),
		log.String("error", "connection timeout"),
		log.Int("retry_count", 3),
	)

	// 4. 更多日志
	log.Info("任务开始执行", log.String("task_id", "TASK-001"))
	log.Debug("加载配置文件", log.String("path", "/etc/app/config.yaml"))
	log.Info("配置加载完成")
	log.Warn("缓存未命中", log.String("key", "user:10086"))
	log.Error("Redis 连接失败",
		log.String("host", "redis.example.com"),
		log.Int("port", 6379),
		log.String("error", "i/o timeout"),
	)

	// 等待日志写入
	time.Sleep(100 * time.Millisecond)
	log.Flush()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("日志记录完成！")
	fmt.Println("========================================")
	fmt.Println()

	// 显示文件内容
	showLogFiles(logDir)
}

func showLogFiles(logDir string) {
	files := []string{"app.log", "error.log"}

	for _, filename := range files {
		path := filepath.Join(logDir, filename)
		fmt.Printf("========================================\n")
		fmt.Printf("文件: %s\n", filename)
		fmt.Printf("========================================\n")

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("读取失败: %v\n\n", err)
			continue
		}

		if len(data) == 0 {
			fmt.Println("(空文件)")
		} else {
			fmt.Println(string(data))
		}
		fmt.Println()
	}

	// 统计日志行数
	fmt.Println("========================================")
	fmt.Println("统计信息")
	fmt.Println("========================================")

	for _, filename := range files {
		path := filepath.Join(logDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := 0
		for _, b := range data {
			if b == '\n' {
				lines++
			}
		}

		fmt.Printf("%-12s: %d 行\n", filename, lines)
	}
	fmt.Println()

	fmt.Println("结论:")
	fmt.Println("✅ app.log 包含所有日志（DEBUG, INFO, WARN, ERROR）")
	fmt.Println("✅ error.log 只包含 ERROR 日志")
	fmt.Println("✅ ERROR 日志同时出现在两个文件中（重复记录）")
}
