// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示链路追踪集成功能
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/log/middleware"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("链路追踪集成示例")
	fmt.Println("========================================")
	fmt.Println()

	// 配置日志
	opts := log.NewOptions()
	opts.Level = "debug"
	opts.EnableColor = true
	opts.Format = "console"
	log.Init(opts)
	defer log.Flush()

	fmt.Println("功能演示:")
	fmt.Println("1. 手动设置追踪信息")
	fmt.Println("2. 模拟 HTTP 请求处理")
	fmt.Println("3. 跨服务调用追踪")
	fmt.Println()

	// === 示例 1: 手动设置追踪信息 ===
	fmt.Println("========================================")
	fmt.Println("示例 1: 手动设置追踪信息")
	fmt.Println("========================================")
	fmt.Println()

	ctx := context.Background()

	// 生成追踪 ID
	traceID := idutil.NewTraceID()
	spanID := idutil.NewSpanID()
	requestID := idutil.NewRequestID()

	fmt.Printf("生成的追踪信息:\n")
	fmt.Printf("  Trace ID:   %s\n", traceID)
	fmt.Printf("  Span ID:    %s\n", spanID)
	fmt.Printf("  Request ID: %s\n", requestID)
	fmt.Println()

	// 将追踪信息注入 context
	ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

	// 使用带追踪信息的日志
	log.InfoContext(ctx, "用户登录", log.String("username", "alice"))
	log.DebugContext(ctx, "验证用户凭证")
	log.InfoContext(ctx, "生成会话令牌", log.String("token", "tk-xxxxx"))

	fmt.Println()

	// === 示例 2: 模拟业务流程 ===
	fmt.Println("========================================")
	fmt.Println("示例 2: 模拟订单处理流程（带追踪）")
	fmt.Println("========================================")
	fmt.Println()

	// 模拟处理订单
	processOrder(ctx)

	fmt.Println()

	// === 示例 3: 模拟 HTTP 服务器 ===
	fmt.Println("========================================")
	fmt.Println("示例 3: HTTP 服务器示例")
	fmt.Println("========================================")
	fmt.Println()

	startHTTPServerExample()
}

// processOrder 模拟订单处理流程
func processOrder(ctx context.Context) {
	log.InfoContext(ctx, "开始处理订单", log.String("order_id", "ORD-12345"))

	// 创建子 span
	spanID := idutil.NewSpanID()
	childCtx := log.WithSpanID(ctx, spanID)

	// 验证库存
	log.DebugContext(childCtx, "验证库存", log.Int("product_id", 100), log.Int("quantity", 2))
	time.Sleep(50 * time.Millisecond)
	log.InfoContext(childCtx, "库存验证通过")

	// 扣减库存
	spanID2 := idutil.NewSpanID()
	childCtx2 := log.WithSpanID(ctx, spanID2)
	log.DebugContext(childCtx2, "扣减库存")
	time.Sleep(30 * time.Millisecond)
	log.InfoContext(childCtx2, "库存扣减成功")

	// 调用支付服务
	callPaymentService(ctx)

	// 订单完成
	log.InfoContext(ctx, "订单处理完成", log.String("order_id", "ORD-12345"))
}

// callPaymentService 模拟调用支付服务
func callPaymentService(ctx context.Context) {
	// 新的 span
	spanID := idutil.NewSpanID()
	paymentCtx := log.WithSpanID(ctx, spanID)

	log.InfoContext(paymentCtx, "调用支付服务", log.String("service", "payment-api"))
	time.Sleep(100 * time.Millisecond)

	// 模拟支付处理
	log.DebugContext(paymentCtx, "验证支付信息")
	log.DebugContext(paymentCtx, "调用第三方支付接口")
	log.InfoContext(paymentCtx, "支付成功", log.String("transaction_id", "TXN-98765"))
}

// startHTTPServerExample 演示 HTTP 服务器集成
func startHTTPServerExample() {
	// 创建一个简单的 handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 日志会自动包含 trace_id, span_id, request_id
		log.InfoContext(ctx, "处理 API 请求",
			log.String("method", r.Method),
			log.String("path", r.URL.Path),
		)

		// 模拟业务逻辑
		time.Sleep(50 * time.Millisecond)

		// 调用其他服务
		callUserService(ctx)

		log.InfoContext(ctx, "API 请求处理完成")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 使用追踪中间件包装 handler
	tracedHandler := middleware.TracingMiddleware(handler)

	// 创建测试请求
	req, _ := http.NewRequest("GET", "/api/v1/users/123", nil)
	req.Header.Set("X-Trace-Id", "existing-trace-id-123456")

	// 创建响应记录器
	rr := &responseRecorder{header: make(http.Header)}

	fmt.Println("模拟 HTTP 请求:")
	fmt.Printf("  Method: %s\n", req.Method)
	fmt.Printf("  Path:   %s\n", req.URL.Path)
	fmt.Printf("  Header: X-Trace-Id = %s\n", req.Header.Get("X-Trace-Id"))
	fmt.Println()
	fmt.Println("请求处理日志:")

	// 处理请求
	tracedHandler.ServeHTTP(rr, req)

	fmt.Println()
	fmt.Printf("响应头:\n")
	fmt.Printf("  X-Trace-Id:   %s\n", rr.Header().Get("X-Trace-Id"))
	fmt.Printf("  X-Request-Id: %s\n", rr.Header().Get("X-Request-Id"))
	fmt.Println()
}

// callUserService 模拟调用用户服务
func callUserService(ctx context.Context) {
	spanID := idutil.NewSpanID()
	userCtx := log.WithSpanID(ctx, spanID)

	log.DebugContext(userCtx, "调用用户服务", log.String("service", "user-api"))
	time.Sleep(30 * time.Millisecond)
	log.InfoContext(userCtx, "获取用户信息成功",
		log.Int("user_id", 123),
		log.String("username", "alice"),
	)
}

// responseRecorder 简单的响应记录器（用于测试）
type responseRecorder struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return len(data), nil
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}
