// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示按小时轮转日志功能
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
	logDir := "/tmp/component-base-hourly-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("按小时轮转日志示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置按小时轮转
	opts := log.NewOptions()
	opts.EnableTimeRotation = true            // 启用按时间轮转
	opts.TimeRotationFormat = "2006-01-02-15" // 按小时轮转（年-月-日-时）
	opts.MaxAge = 24                          // 保留24小时的日志
	opts.EnableColor = true
	opts.Level = "info"

	// 配置日志输出
	opts.OutputPaths = []string{
		"stdout",
		filepath.Join(logDir, "app.log"),
	}

	log.Init(opts)
	defer log.Flush()

	fmt.Println("配置说明:")
	fmt.Printf("- 启用按时间轮转: %v\n", opts.EnableTimeRotation)
	fmt.Printf("- 轮转格式: %s (按小时)\n", opts.TimeRotationFormat)
	fmt.Printf("- 保留小时数: %d 小时\n", opts.MaxAge)
	fmt.Printf("- 日志目录: %s\n", logDir)
	fmt.Println()

	// 当前时间
	now := time.Now()
	currentHour := now.Format("2006-01-02-15")
	fmt.Printf("当前时间: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("日志文件将生成为: app.%s.log\n", currentHour)
	fmt.Println()

	fmt.Println("开始记录日志...")
	fmt.Println("========================================")
	fmt.Println()

	// 模拟一小时内不同时刻的日志
	for minute := 0; minute < 60; minute += 10 {
		log.Infof("[%02d:%02d] 处理请求批次 #%d", now.Hour(), minute, minute/10+1)
		log.Warnf("[%02d:%02d] 内存使用: %.1f%%", now.Hour(), minute, 60.0+(float64(minute)/2))
	}

	// 记录一些事件
	log.Info("服务启动", log.String("version", "v1.0.0"), log.String("env", "production"))
	log.Info("数据库连接成功", log.String("host", "localhost:3306"))
	log.Warn("缓存命中率较低", log.Float64("hit_rate", 0.45))
	log.Error("API 调用失败", log.String("api", "/users"), log.Int("status_code", 500))

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
		fmt.Printf("  - %s (大小: %d 字节, 修改时间: %s)\n",
			file.Name(),
			info.Size(),
			info.ModTime().Format("15:04:05"))
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("说明：")
	fmt.Println("1. 日志文件按小时自动创建：app.YYYY-MM-DD-HH.log")
	fmt.Println("2. 每小时 00:00 会自动切换到新的日志文件")
	fmt.Println("3. 超过 MaxAge 小时的日志会被自动清理")
	fmt.Println()
	fmt.Println("适用场景：")
	fmt.Println("  - 高流量应用，日志量大")
	fmt.Println("  - 需要细粒度的日志管理")
	fmt.Println("  - 便于按小时统计和分析")
	fmt.Println("========================================")
}
