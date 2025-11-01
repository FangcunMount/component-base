// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示按天轮转日志功能
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
	logDir := "/tmp/component-base-daily-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("按天轮转日志示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置按天轮转
	opts := log.NewOptions()
	opts.EnableTimeRotation = true         // 启用按时间轮转
	opts.TimeRotationFormat = "2006-01-02" // 按天轮转（年-月-日）
	opts.MaxAge = 7                        // 保留7天的日志
	opts.EnableColor = true
	opts.Level = "debug"

	// 配置日志输出
	opts.OutputPaths = []string{
		"stdout",
		filepath.Join(logDir, "app.log"),
	}

	log.Init(opts)
	defer log.Flush()

	fmt.Println("配置说明:")
	fmt.Printf("- 启用按时间轮转: %v\n", opts.EnableTimeRotation)
	fmt.Printf("- 轮转格式: %s (按天)\n", opts.TimeRotationFormat)
	fmt.Printf("- 保留天数: %d 天\n", opts.MaxAge)
	fmt.Printf("- 日志目录: %s\n", logDir)
	fmt.Println()

	// 当前时间
	today := time.Now().Format("2006-01-02")
	fmt.Printf("当前日期: %s\n", today)
	fmt.Printf("日志文件将生成为: app.log.%s.log\n", today)
	fmt.Println()

	fmt.Println("开始记录日志...")
	fmt.Println("========================================")
	fmt.Println()

	// 模拟一天中不同时间的日志
	for hour := 0; hour < 24; hour += 3 {
		log.Infof("[%02d:00] 系统正常运行", hour)
		log.Debugf("[%02d:00] 处理了 %d 个请求", hour, hour*100)

		if hour == 12 {
			log.Warn("[12:00] CPU 使用率较高")
		}

		if hour == 18 {
			log.Error("[18:00] 发现异常连接")
		}
	}

	// 记录一些业务日志
	log.Info("用户登录", log.String("user", "alice"), log.String("ip", "192.168.1.100"))
	log.Info("订单创建", log.String("order_id", "ORDER-2024-001"), log.Int("amount", 1000))
	log.Warn("库存预警", log.String("product", "iPhone 15"), log.Int("stock", 5))
	log.Error("支付失败", log.String("order_id", "ORDER-2024-002"), log.String("reason", "余额不足"))

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("日志记录完成！")
	fmt.Println("========================================")
	fmt.Println()

	// 列出生成的日志文件
	fmt.Println("生成的日志文件:")
	files, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, _ := file.Info()
		fmt.Printf("  - %s (大小: %d 字节)\n", file.Name(), info.Size())
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("说明：")
	fmt.Println("1. 日志文件按天自动创建：app.log.YYYY-MM-DD.log")
	fmt.Println("2. 每天 00:00 会自动切换到新的日志文件")
	fmt.Println("3. 超过 MaxAge 天数的日志会被自动清理")
	fmt.Println()
	fmt.Println("其他支持的时间格式：")
	fmt.Println("  - '2006-01-02'      : 按天轮转")
	fmt.Println("  - '2006-01-02-15'   : 按小时轮转")
	fmt.Println("  - '2006-01'         : 按月轮转")
	fmt.Println("  - '2006-W01'        : 按周轮转")
	fmt.Println("========================================")
}
