// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 对比展示带颜色和不带颜色的日志输出
package main

import (
	"fmt"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("示例 1: 不带颜色的日志输出")
	fmt.Println("========================================")

	// 不带颜色的配置
	opts1 := log.NewOptions()
	opts1.EnableColor = false
	opts1.Level = "debug"
	log.Init(opts1)

	log.Debug("调试信息：系统正在处理请求")
	log.Info("提示信息：用户登录成功")
	log.Warn("警告信息：内存使用率达到 80%")
	log.Error("错误信息：数据库连接失败")

	fmt.Println("\n========================================")
	fmt.Println("示例 2: 带颜色的日志输出")
	fmt.Println("========================================")

	// 带颜色的配置
	opts2 := log.NewOptions()
	opts2.EnableColor = true
	opts2.Level = "debug"
	log.Init(opts2)

	log.Debug("调试信息：系统正在处理请求")
	log.Info("提示信息：用户登录成功")
	log.Warn("警告信息：内存使用率达到 80%")
	log.Error("错误信息：数据库连接失败")

	fmt.Println("\n========================================")
	fmt.Println("对比说明：")
	fmt.Println("1. 示例 1 的日志级别仅带方括号，无颜色")
	fmt.Println("2. 示例 2 的日志级别带方括号和颜色")
	fmt.Println("   - [DEBUG] 显示为青色")
	fmt.Println("   - [INFO]  显示为绿色")
	fmt.Println("   - [WARN]  显示为黄色")
	fmt.Println("   - [ERROR] 显示为红色")
	fmt.Println("========================================")

	log.Flush()
}
