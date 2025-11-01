// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示精确级别输出模式
package main

import (
	"fmt"
	"os"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	// 创建日志目录
	logDir := "/tmp/component-base-logs-exact"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("日志精确级别输出示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置精确级别输出
	opts := log.NewOptions()
	opts.EnableLevelOutput = true
	opts.LevelOutputMode = "exact" // 只输出精确级别的日志
	opts.EnableColor = true
	opts.Level = "debug"

	// 为每个级别配置独立的输出
	opts.LevelOutputPaths = map[string][]string{
		"debug": []string{"stdout", fmt.Sprintf("%s/debug.log", logDir)},
		"info":  []string{"stdout", fmt.Sprintf("%s/info.log", logDir)},
		"warn":  []string{"stdout", fmt.Sprintf("%s/warn.log", logDir)},
		"error": []string{"stdout", fmt.Sprintf("%s/error.log", logDir)},
	}

	log.Init(opts)
	defer log.Flush()

	fmt.Println("配置说明:")
	fmt.Printf("- 输出模式: %s (仅输出精确级别)\n", opts.LevelOutputMode)
	fmt.Printf("- DEBUG 日志: 控制台 + %s/debug.log\n", logDir)
	fmt.Printf("- INFO 日志:  控制台 + %s/info.log\n", logDir)
	fmt.Printf("- WARN 日志:  控制台 + %s/warn.log\n", logDir)
	fmt.Printf("- ERROR 日志: 控制台 + %s/error.log\n", logDir)
	fmt.Println()

	fmt.Println("开始记录日志...")
	fmt.Println("========================================")
	fmt.Println()

	// 记录不同级别的日志
	log.Debug("这是一条 DEBUG 日志")
	log.Info("这是一条 INFO 日志")
	log.Warn("这是一条 WARN 日志")
	log.Error("这是一条 ERROR 日志")

	// 记录更多日志以观察效果
	for i := 1; i <= 3; i++ {
		log.Debugf("Debug 消息 #%d", i)
		log.Infof("Info 消息 #%d", i)
		log.Warnf("Warn 消息 #%d", i)
		log.Errorf("Error 消息 #%d", i)
	}

	fmt.Println("\n========================================")
	fmt.Println("日志记录完成！")
	fmt.Println("========================================")
	fmt.Println()

	// 显示日志文件内容
	fmt.Println("查看日志文件内容:")
	fmt.Println("========================================")
	fmt.Println()

	files := map[string]string{
		"DEBUG": fmt.Sprintf("%s/debug.log", logDir),
		"INFO":  fmt.Sprintf("%s/info.log", logDir),
		"WARN":  fmt.Sprintf("%s/warn.log", logDir),
		"ERROR": fmt.Sprintf("%s/error.log", logDir),
	}

	for level, file := range files {
		fmt.Printf("\n>>> %s 级别日志 (%s) <<<\n", level, file)
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("读取失败: %v\n", err)
			continue
		}
		if len(content) == 0 {
			fmt.Println("(文件为空)")
		} else {
			lines := 0
			for _, b := range content {
				if b == '\n' {
					lines++
				}
			}
			fmt.Printf("共 %d 行日志\n", lines)
			fmt.Print(string(content[:min(len(content), 500)]))
			if len(content) > 500 {
				fmt.Println("...")
			}
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("提示：由于使用 'exact' 模式：")
	fmt.Println("- debug.log 仅包含 DEBUG 级别的日志")
	fmt.Println("- info.log 仅包含 INFO 级别的日志")
	fmt.Println("- warn.log 仅包含 WARN 级别的日志")
	fmt.Println("- error.log 仅包含 ERROR 级别的日志")
	fmt.Println("========================================")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
