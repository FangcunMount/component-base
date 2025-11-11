// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// 演示类型化日志的使用
package main

import (
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
)

func main() {
	// 初始化日志
	opts := log.NewOptions()
	opts.Level = "debug"
	opts.Format = "json" // JSON 格式便于类型过滤
	opts.EnableColor = false
	log.Init(opts)
	defer log.Flush()

	fmt.Println("========================================")
	fmt.Println("类型化日志示例")
	fmt.Println("========================================")
	fmt.Println()

	fmt.Println("展示不同类型的日志记录方式...")
	fmt.Println()

	// 1. HTTP 请求日志
	fmt.Println("1️⃣  HTTP 请求日志")
	fmt.Println("----------------------------------------")

	log.HTTP("GET /api/users",
		log.String("method", "GET"),
		log.String("path", "/api/users"),
		log.Int("status", 200),
		log.Float64("duration_ms", 45.6),
		log.String("client_ip", "192.168.1.100"),
	)

	log.HTTPDebug("请求头详情",
		log.String("user_agent", "Mozilla/5.0"),
		log.String("content_type", "application/json"),
	)

	log.HTTPWarn("响应慢",
		log.String("path", "/api/orders"),
		log.Float64("duration_ms", 1523.4),
		log.Float64("threshold_ms", 1000.0),
	)

	log.HTTPError("请求失败",
		log.String("path", "/api/payment"),
		log.Int("status", 500),
		log.String("error", "internal server error"),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 2. SQL 查询日志
	fmt.Println("2️⃣  SQL 查询日志")
	fmt.Println("----------------------------------------")

	log.SQL("查询用户信息",
		log.String("query", "SELECT * FROM users WHERE id = ?"),
		log.Int("rows", 1),
		log.Float64("duration_ms", 12.3),
	)

	log.SQLDebug("SQL 执行计划",
		log.String("explain", "Using index; Using where"),
		log.Int("cost", 150),
	)

	log.SQLWarn("慢查询",
		log.String("query", "SELECT * FROM orders WHERE created_at > ?"),
		log.Float64("duration_ms", 2500.0),
		log.Int("rows", 10000),
	)

	log.SQLError("查询失败",
		log.String("query", "INSERT INTO users (name) VALUES (?)"),
		log.String("error", "Duplicate entry 'alice' for key 'name'"),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 3. gRPC 调用日志
	fmt.Println("3️⃣  gRPC 调用日志")
	fmt.Println("----------------------------------------")

	log.GRPC("UserService.GetUser",
		log.String("service", "UserService"),
		log.String("method", "GetUser"),
		log.String("code", "OK"),
		log.Float64("duration_ms", 23.5),
	)

	log.GRPCDebug("请求详情",
		log.String("request", `{"user_id": "10086"}`),
		log.Int("request_size", 20),
	)

	log.GRPCWarn("响应超时告警",
		log.String("service", "OrderService"),
		log.String("method", "CreateOrder"),
		log.Float64("duration_ms", 4500.0),
	)

	log.GRPCError("调用失败",
		log.String("service", "PaymentService"),
		log.String("method", "Charge"),
		log.String("code", "UNAVAILABLE"),
		log.String("error", "connection refused"),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 4. Redis 操作日志
	fmt.Println("4️⃣  Redis 操作日志")
	fmt.Println("----------------------------------------")

	log.Redis("GET user:10086",
		log.String("command", "GET"),
		log.String("key", "user:10086"),
		log.Bool("hit", true),
		log.Float64("duration_ms", 1.2),
	)

	log.RedisDebug("连接池状态",
		log.Int("active", 10),
		log.Int("idle", 5),
		log.Int("max", 100),
	)

	log.RedisWarn("缓存未命中",
		log.String("key", "user:99999"),
		log.String("fallback", "query from database"),
	)

	log.RedisError("连接失败",
		log.String("host", "redis.example.com:6379"),
		log.String("error", "dial timeout"),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 5. 消息队列日志
	fmt.Println("5️⃣  消息队列日志")
	fmt.Println("----------------------------------------")

	log.MQ("发送消息",
		log.String("topic", "order-events"),
		log.String("message_id", "MSG-001"),
		log.Int("size", 1024),
	)

	log.MQDebug("消费者配置",
		log.String("group", "order-processor"),
		log.Int("workers", 5),
	)

	log.MQWarn("消息积压",
		log.String("topic", "user-events"),
		log.Int("pending", 5000),
		log.Int("threshold", 1000),
	)

	log.MQError("消息发送失败",
		log.String("topic", "payment-events"),
		log.String("error", "broker not available"),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 6. 缓存操作日志
	fmt.Println("6️⃣  缓存操作日志")
	fmt.Println("----------------------------------------")

	log.Cache("缓存命中",
		log.String("key", "product:12345"),
		log.Int("ttl", 3600),
	)

	log.CacheDebug("缓存统计",
		log.Float64("hit_rate", 0.95),
		log.Int64("total_keys", 10000),
	)

	log.CacheWarn("缓存即将过期",
		log.String("key", "session:abc123"),
		log.Int("remaining_ttl", 60),
	)

	log.CacheError("缓存删除失败",
		log.String("key", "temp:xyz789"),
		log.String("error", "key not found"),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 7. RPC 调用日志（通用）
	fmt.Println("7️⃣  RPC 调用日志")
	fmt.Println("----------------------------------------")

	log.RPC("调用订单服务",
		log.String("service", "order-service"),
		log.String("endpoint", "http://order.example.com/api"),
		log.Int("status", 200),
	)

	log.RPCError("服务调用失败",
		log.String("service", "inventory-service"),
		log.String("error", "connection timeout"),
		log.Int("retry_count", 3),
	)

	time.Sleep(10 * time.Millisecond)
	fmt.Println()

	// 8. 混合使用
	fmt.Println("8️⃣  实际业务场景：处理订单")
	fmt.Println("----------------------------------------")

	// 模拟一个完整的订单处理流程
	orderID := "ORD-20251111-001"

	log.HTTP("接收订单请求",
		log.String("method", "POST"),
		log.String("path", "/api/v1/orders"),
		log.String("order_id", orderID),
	)

	log.SQL("查询用户信息",
		log.String("query", "SELECT * FROM users WHERE id = ?"),
		log.String("user_id", "10086"),
		log.Float64("duration_ms", 5.2),
	)

	log.Redis("检查库存缓存",
		log.String("key", "inventory:PROD-12345"),
		log.Bool("hit", true),
	)

	log.SQL("扣减库存",
		log.String("query", "UPDATE inventory SET quantity = quantity - ? WHERE product_id = ?"),
		log.Int("affected_rows", 1),
	)

	log.GRPC("调用支付服务",
		log.String("service", "PaymentService"),
		log.String("method", "Charge"),
		log.String("code", "OK"),
	)

	log.MQ("发送订单事件",
		log.String("topic", "order-created"),
		log.String("order_id", orderID),
	)

	log.HTTP("订单创建成功",
		log.String("order_id", orderID),
		log.Int("status", 200),
	)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("日志优势说明")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("✅ 类型标记的优势:")
	fmt.Println("1. 便于日志过滤和分析")
	fmt.Println("   - 只查看 HTTP 日志: grep '\"type\":\"HTTP\"'")
	fmt.Println("   - 只查看 SQL 日志: grep '\"type\":\"SQL\"'")
	fmt.Println()
	fmt.Println("2. 便于性能监控")
	fmt.Println("   - 统计 HTTP 平均响应时间")
	fmt.Println("   - 统计 SQL 慢查询数量")
	fmt.Println("   - 监控 gRPC 调用失败率")
	fmt.Println()
	fmt.Println("3. 便于日志聚合分析")
	fmt.Println("   - ELK: 按 type 字段创建仪表盘")
	fmt.Println("   - Grafana: 按类型创建图表")
	fmt.Println("   - 自定义分析工具更容易处理")
	fmt.Println()
	fmt.Println("4. 代码可读性更好")
	fmt.Println("   - log.HTTP() 比 log.Info() 意图更明确")
	fmt.Println("   - 自动添加 type 字段，减少重复代码")
	fmt.Println()
}
