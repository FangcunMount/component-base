// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// 对比三种日志分级输出模式
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("日志分级输出模式对比")
	fmt.Println("========================================")
	fmt.Println()

	// 测试三种模式
	modes := []string{"duplicate", "above", "exact"}

	for _, mode := range modes {
		testMode(mode)
		fmt.Println()
	}

	fmt.Println("========================================")
	fmt.Println("总结")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("📊 Duplicate 模式（推荐）:")
	fmt.Println("   - app.log 记录所有日志（完整）")
	fmt.Println("   - error.log 额外记录错误（快速定位）")
	fmt.Println("   - 适合生产环境")
	fmt.Println()
	fmt.Println("📊 Above 模式:")
	fmt.Println("   - info.log 包含 INFO + WARN + ERROR")
	fmt.Println("   - error.log 包含 ERROR")
	fmt.Println("   - 日志有重复")
	fmt.Println()
	fmt.Println("📊 Exact 模式:")
	fmt.Println("   - 每个文件只包含对应级别")
	fmt.Println("   - 没有重复，严格分离")
	fmt.Println("   - 需要查看多个文件才能了解全貌")
}

func testMode(mode string) {
	logDir := filepath.Join("./logs/comparison", mode)
	os.MkdirAll(logDir, 0755)

	fmt.Printf("========================================\n")
	fmt.Printf("测试模式: %s\n", mode)
	fmt.Printf("========================================\n")

	// 配置日志
	opts := log.NewOptions()
	opts.Level = "debug"
	opts.Format = "console"
	opts.EnableColor = false
	opts.EnableLevelOutput = true
	opts.LevelOutputMode = mode

	// 根据模式配置输出路径
	if mode == "duplicate" {
		opts.LevelOutputPaths = map[string][]string{
			"all":   {filepath.Join(logDir, "app.log")},
			"error": {filepath.Join(logDir, "error.log")},
		}
	} else {
		opts.LevelOutputPaths = map[string][]string{
			"info":  {filepath.Join(logDir, "info.log")},
			"error": {filepath.Join(logDir, "error.log")},
		}
	}

	// 初始化日志
	log.Init(opts)

	// 写入测试日志
	log.Debug("这是 DEBUG 日志")
	log.Info("这是 INFO 日志")
	log.Warn("这是 WARN 日志")
	log.Error("这是 ERROR 日志")

	log.Flush()

	// 统计结果
	fmt.Println("\n文件统计:")
	files, _ := os.ReadDir(logDir)
	for _, file := range files {
		path := filepath.Join(logDir, file.Name())
		data, _ := os.ReadFile(path)
		lines := 0
		for _, b := range data {
			if b == '\n' {
				lines++
			}
		}
		fmt.Printf("  %-12s: %d 行\n", file.Name(), lines)
	}
}
