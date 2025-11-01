// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示日志分级输出功能
package main

import (
	"fmt"
	"os"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	// 创建日志目录
	logDir := "/tmp/component-base-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("日志分级输出示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置分级输出
	opts := log.NewOptions()
	opts.EnableLevelOutput = true
	opts.LevelOutputMode = "above" // 输出该级别及以上的日志
	opts.EnableColor = true
	opts.Level = "debug"

	// 为不同级别配置不同的输出路径
	opts.LevelOutputPaths = map[string][]string{
		"debug": []string{"stdout"},                                     // debug 日志输出到控制台
		"info":  []string{"stdout", fmt.Sprintf("%s/info.log", logDir)}, // info 日志输出到控制台和文件
		"warn":  []string{fmt.Sprintf("%s/warn.log", logDir)},           // warn 日志只输出到文件
		"error": []string{fmt.Sprintf("%s/error.log", logDir)},          // error 日志只输出到文件
	}

	log.Init(opts)
	defer log.Flush()

	fmt.Println("配置说明:")
	fmt.Printf("- DEBUG 日志: 输出到控制台\n")
	fmt.Printf("- INFO 日志:  输出到控制台 + %s/info.log\n", logDir)
	fmt.Printf("- WARN 日志:  输出到 %s/warn.log\n", logDir)
	fmt.Printf("- ERROR 日志: 输出到 %s/error.log\n", logDir)
	fmt.Printf("- 输出模式: %s (该级别及以上)\n\n", opts.LevelOutputMode)

	fmt.Println("开始记录日志...")
	fmt.Println("========================================")
	fmt.Println()

	// 记录不同级别的日志
	log.Debug("这是一条 DEBUG 日志", log.String("level", "debug"))
	log.Info("这是一条 INFO 日志", log.String("level", "info"))
	log.Warn("这是一条 WARN 日志", log.String("level", "warn"))
	log.Error("这是一条 ERROR 日志", log.String("level", "error"))

	// 记录一些业务日志
	log.Debugf("用户 %s 发起请求", "alice")
	log.Infof("订单 %s 创建成功", "ORDER-12345")
	log.Warnf("库存不足，当前库存: %d", 5)
	log.Errorf("支付失败: %s", "余额不足")

	fmt.Println("\n========================================")
	fmt.Println("日志记录完成！")
	fmt.Println("========================================")
	fmt.Println()

	// 显示日志文件内容
	fmt.Println("查看日志文件内容:")
	fmt.Println("========================================")
	fmt.Println()

	files := []string{
		fmt.Sprintf("%s/info.log", logDir),
		fmt.Sprintf("%s/warn.log", logDir),
		fmt.Sprintf("%s/error.log", logDir),
	}

	for _, file := range files {
		fmt.Printf("\n>>> %s <<<\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("读取失败: %v\n", err)
			continue
		}
		if len(content) == 0 {
			fmt.Println("(文件为空)")
		} else {
			fmt.Print(string(content))
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("提示：由于使用 'above' 模式：")
	fmt.Println("- info.log 包含: INFO, WARN, ERROR 级别的日志")
	fmt.Println("- warn.log 包含: WARN, ERROR 级别的日志")
	fmt.Println("- error.log 包含: ERROR 级别的日志")
	fmt.Println("========================================")
}
