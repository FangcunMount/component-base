// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示不带颜色但有方括号的日志输出
package main

import (
	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	// 初始化不带颜色的日志配置
	opts := log.NewOptions()
	opts.EnableColor = false // 禁用颜色
	opts.Level = "debug"     // 设置为 debug 级别以便看到所有日志
	log.Init(opts)

	// 测试不同级别的日志输出
	log.Debug("This is a DEBUG message", log.String("status", "testing"))
	log.Info("This is an INFO message", log.String("status", "running"))
	log.Warn("This is a WARN message", log.String("status", "warning"))
	log.Error("This is an ERROR message", log.String("status", "error"))

	// 测试格式化输出
	log.Debugf("Debug: Processing item %d of %d", 1, 10)
	log.Infof("Info: Server started on port %d", 8080)
	log.Warnf("Warn: Memory usage at %.2f%%", 85.5)
	log.Errorf("Error: Failed to connect to %s", "database")

	log.Flush()
}
