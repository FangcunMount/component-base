// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// 演示实际生产环境的日志配置：app.log 记录所有，error.log 额外记录错误，按天轮转
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	// 创建日志目录
	logDir := "./logs/production"
	os.MkdirAll(logDir, 0755)

	fmt.Println("========================================")
	fmt.Println("生产环境日志配置示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置日志（生产环境推荐配置）
	opts := log.NewOptions()
	opts.Level = "info"  // 生产环境建议使用 info 级别
	opts.Format = "json" // 生产环境建议使用 JSON 格式（便于日志分析）
	opts.EnableColor = false
	opts.EnableLevelOutput = true
	opts.LevelOutputMode = "duplicate"

	// 启用按天轮转
	opts.EnableTimeRotation = true
	opts.TimeRotationFormat = "2006-01-02" // 按天轮转
	opts.MaxAge = 30                       // 保留 30 天
	opts.Compress = true                   // 压缩旧日志

	// 配置输出路径
	opts.LevelOutputPaths = map[string][]string{
		"all":   {filepath.Join(logDir, "app.log")},   // 所有日志
		"error": {filepath.Join(logDir, "error.log")}, // 额外记录错误
		"warn":  {filepath.Join(logDir, "warn.log")},  // 可选：额外记录警告
	}

	// 初始化日志
	log.Init(opts)
	defer log.Flush()

	fmt.Println("生产环境配置:")
	fmt.Println("✓ 日志级别: INFO（过滤 DEBUG 日志）")
	fmt.Println("✓ 日志格式: JSON（便于解析和分析）")
	fmt.Println("✓ 轮转策略: 按天轮转，保留 30 天")
	fmt.Println("✓ 压缩策略: 自动压缩旧日志")
	fmt.Println()
	fmt.Println("输出配置:")
	fmt.Println("- app.log:   记录所有日志 (INFO, WARN, ERROR)")
	fmt.Println("- error.log: 只记录 ERROR 日志（便于快速定位问题）")
	fmt.Println("- warn.log:  只记录 WARN 日志（可选，便于发现潜在问题）")
	fmt.Println()
	fmt.Println("文件命名:")
	fmt.Println("- 当天日志: app.log, error.log, warn.log")
	fmt.Println("- 历史日志: app.2025-11-10.log, error.2025-11-10.log.gz")
	fmt.Println()

	// 模拟生产环境的日志
	fmt.Println("模拟生产环境日志输出...")
	fmt.Println()

	// 1. 应用启动
	log.Info("应用启动",
		log.String("app", "api-server"),
		log.String("version", "v1.2.3"),
		log.Int("port", 8080),
	)

	// 2. 数据库连接
	log.Info("数据库连接成功",
		log.String("host", "mysql.example.com"),
		log.String("database", "production"),
	)

	// 3. 正常请求
	log.Info("API 请求",
		log.String("method", "POST"),
		log.String("path", "/api/v1/orders"),
		log.String("user_id", "10086"),
		log.Int("status", 200),
		log.Float64("duration_ms", 45.6),
	)

	// 4. 警告：性能问题
	log.Warn("接口响应慢",
		log.String("path", "/api/v1/users"),
		log.Float64("duration_ms", 1523.4),
		log.Float64("threshold_ms", 1000.0),
	)

	// 5. 警告：资源使用
	log.Warn("内存使用率高",
		log.Float64("usage_percent", 85.5),
		log.Int64("used_mb", 6840),
		log.Int64("total_mb", 8000),
	)

	// 6. 错误：业务异常
	log.Error("订单创建失败",
		log.String("order_id", "ORD-20251111-001"),
		log.String("user_id", "10086"),
		log.String("error", "库存不足"),
		log.String("product_id", "PROD-12345"),
		log.Int("required_quantity", 10),
		log.Int("available_quantity", 3),
	)

	// 7. 错误：第三方服务
	log.Error("支付服务调用失败",
		log.String("service", "payment-gateway"),
		log.String("endpoint", "https://pay.example.com/api/charge"),
		log.String("error", "connection timeout after 5s"),
		log.Int("retry_count", 3),
		log.Bool("will_retry", true),
	)

	// 8. 错误：数据库
	log.Error("数据库查询失败",
		log.String("query", "SELECT * FROM orders WHERE user_id = ?"),
		log.String("error", "Error 1062: Duplicate entry '123' for key 'PRIMARY'"),
		log.String("user_id", "10086"),
	)

	// 9. 更多正常日志
	log.Info("缓存更新",
		log.String("key", "user:10086:profile"),
		log.Int("ttl_seconds", 3600),
	)

	log.Info("消息队列发送",
		log.String("topic", "order-events"),
		log.String("message_id", "MSG-202511110001"),
	)

	// 10. 警告：配置问题
	log.Warn("使用默认配置",
		log.String("config_key", "max_connections"),
		log.Int("default_value", 100),
		log.String("reason", "配置文件中未设置"),
	)

	log.Flush()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("日志记录完成！")
	fmt.Println("========================================")
	fmt.Println()

	// 显示文件统计
	showFileStats(logDir)

	fmt.Println()
	fmt.Println("使用建议:")
	fmt.Println("1. 监控 error.log 大小，设置告警阈值")
	fmt.Println("2. 定期分析 warn.log，发现潜在问题")
	fmt.Println("3. 使用日志收集工具（如 ELK）分析 JSON 格式日志")
	fmt.Println("4. 设置日志轮转避免磁盘占满")
	fmt.Println("5. 错误日志触发告警通知相关人员")
}

func showFileStats(logDir string) {
	fmt.Println("文件统计:")
	fmt.Println("========================================")

	files := []string{"app.log", "error.log", "warn.log"}
	for _, filename := range files {
		path := filepath.Join(logDir, filename)
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("%-12s: (不存在)\n", filename)
			continue
		}

		// 读取文件统计行数
		data, _ := os.ReadFile(path)
		lines := 0
		for _, b := range data {
			if b == '\n' {
				lines++
			}
		}

		fmt.Printf("%-12s: %d 行, %d 字节\n", filename, lines, info.Size())
	}

	fmt.Println()
	fmt.Println("文件说明:")
	fmt.Println("- app.log:   完整日志，用于问题追踪和审计")
	fmt.Println("- error.log: 错误日志，用于快速定位故障")
	fmt.Println("- warn.log:  警告日志，用于发现潜在问题")
}
