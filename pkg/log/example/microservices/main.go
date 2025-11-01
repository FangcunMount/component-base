// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package main 展示微服务间的链路追踪
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
	// 初始化日志
	opts := log.NewOptions()
	opts.Level = "debug"
	opts.EnableColor = true
	opts.Format = "console"
	log.Init(opts)
	defer log.Flush()

	fmt.Println("========================================")
	fmt.Println("微服务链路追踪示例")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("场景: 电商订单创建流程")
	fmt.Println("  API Gateway -> Order Service -> Inventory Service")
	fmt.Println("               -> Payment Service -> Notification Service")
	fmt.Println()

	// 模拟 API Gateway 接收请求
	simulateAPIGateway()
}

// simulateAPIGateway 模拟 API 网关
func simulateAPIGateway() {
	fmt.Println("========================================")
	fmt.Println("[API Gateway] 接收客户端请求")
	fmt.Println("========================================")
	fmt.Println()

	// 创建 HTTP 请求
	req, _ := http.NewRequest("POST", "/api/v1/orders", nil)
	req.Header.Set("User-Agent", "Mobile-App/1.0")

	// 应用追踪中间件
	handler := http.HandlerFunc(handleCreateOrder)
	tracedHandler := middleware.TracingMiddleware(handler)

	// 处理请求
	rr := &mockResponseWriter{header: make(http.Header)}
	tracedHandler.ServeHTTP(rr, req)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("请求处理完成")
	fmt.Println("========================================")
	fmt.Printf("响应头 X-Trace-Id: %s\n", rr.Header().Get("X-Trace-Id"))
	fmt.Printf("响应头 X-Request-Id: %s\n", rr.Header().Get("X-Request-Id"))
	fmt.Println()
}

// handleCreateOrder 处理创建订单请求
func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.InfoContext(ctx, "[Order Service] 开始创建订单",
		log.String("user_id", "U-10086"),
		log.String("product_id", "P-88888"),
	)

	// 1. 验证库存
	if !checkInventory(ctx, "P-88888", 2) {
		log.ErrorContext(ctx, "[Order Service] 库存不足")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. 创建订单记录
	orderID := createOrderRecord(ctx)

	// 3. 调用支付服务
	if !processPayment(ctx, orderID, 299.99) {
		log.ErrorContext(ctx, "[Order Service] 支付失败")
		w.WriteHeader(http.StatusPaymentRequired)
		return
	}

	// 4. 发送通知
	sendNotification(ctx, orderID)

	log.InfoContext(ctx, "[Order Service] 订单创建成功",
		log.String("order_id", orderID),
	)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"order_id":"` + orderID + `"}`))
}

// checkInventory 检查库存（模拟调用库存服务）
func checkInventory(ctx context.Context, productID string, quantity int) bool {
	// 创建新的 span 表示调用库存服务
	spanID := idutil.NewSpanID()
	invCtx := log.WithSpanID(ctx, spanID)

	log.InfoContext(invCtx, "[Inventory Service] 收到库存查询请求",
		log.String("product_id", productID),
		log.Int("quantity", quantity),
	)

	// 模拟数据库查询
	time.Sleep(20 * time.Millisecond)
	log.DebugContext(invCtx, "[Inventory Service] 查询数据库")

	currentStock := 100
	log.DebugContext(invCtx, "[Inventory Service] 当前库存",
		log.Int("current_stock", currentStock),
	)

	if currentStock >= quantity {
		// 锁定库存
		time.Sleep(10 * time.Millisecond)
		log.InfoContext(invCtx, "[Inventory Service] 库存锁定成功",
			log.Int("locked_quantity", quantity),
		)
		return true
	}

	log.WarnContext(invCtx, "[Inventory Service] 库存不足",
		log.Int("required", quantity),
		log.Int("available", currentStock),
	)
	return false
}

// createOrderRecord 创建订单记录
func createOrderRecord(ctx context.Context) string {
	spanID := idutil.NewSpanID()
	orderCtx := log.WithSpanID(ctx, spanID)

	orderID := fmt.Sprintf("ORD-%d", time.Now().UnixNano())

	log.DebugContext(orderCtx, "[Order Service] 生成订单 ID",
		log.String("order_id", orderID),
	)

	// 模拟数据库插入
	time.Sleep(30 * time.Millisecond)
	log.DebugContext(orderCtx, "[Order Service] 写入订单数据库")

	log.InfoContext(orderCtx, "[Order Service] 订单记录创建成功",
		log.String("order_id", orderID),
	)

	return orderID
}

// processPayment 处理支付（模拟调用支付服务）
func processPayment(ctx context.Context, orderID string, amount float64) bool {
	spanID := idutil.NewSpanID()
	payCtx := log.WithSpanID(ctx, spanID)

	log.InfoContext(payCtx, "[Payment Service] 收到支付请求",
		log.String("order_id", orderID),
		log.Float64("amount", amount),
	)

	// 验证支付信息
	time.Sleep(15 * time.Millisecond)
	log.DebugContext(payCtx, "[Payment Service] 验证支付信息")

	// 调用第三方支付网关
	transactionID := callPaymentGateway(payCtx, amount)
	if transactionID == "" {
		log.ErrorContext(payCtx, "[Payment Service] 第三方支付失败")
		return false
	}

	// 更新订单支付状态
	time.Sleep(10 * time.Millisecond)
	log.DebugContext(payCtx, "[Payment Service] 更新支付状态")

	log.InfoContext(payCtx, "[Payment Service] 支付成功",
		log.String("transaction_id", transactionID),
	)

	return true
}

// callPaymentGateway 调用支付网关
func callPaymentGateway(ctx context.Context, amount float64) string {
	spanID := idutil.NewSpanID()
	gwCtx := log.WithSpanID(ctx, spanID)

	log.DebugContext(gwCtx, "[Payment Gateway] 创建支付交易",
		log.Float64("amount", amount),
	)

	// 模拟网络延迟
	time.Sleep(80 * time.Millisecond)

	transactionID := fmt.Sprintf("TXN-%d", time.Now().UnixNano())

	log.InfoContext(gwCtx, "[Payment Gateway] 支付网关响应",
		log.String("transaction_id", transactionID),
		log.String("status", "SUCCESS"),
	)

	return transactionID
}

// sendNotification 发送通知（模拟调用通知服务）
func sendNotification(ctx context.Context, orderID string) {
	spanID := idutil.NewSpanID()
	notifyCtx := log.WithSpanID(ctx, spanID)

	log.InfoContext(notifyCtx, "[Notification Service] 发送订单通知",
		log.String("order_id", orderID),
	)

	// 发送短信
	sendSMS(notifyCtx, orderID)

	// 发送邮件
	sendEmail(notifyCtx, orderID)

	log.InfoContext(notifyCtx, "[Notification Service] 通知发送完成")
}

// sendSMS 发送短信
func sendSMS(ctx context.Context, orderID string) {
	spanID := idutil.NewSpanID()
	smsCtx := log.WithSpanID(ctx, spanID)

	log.DebugContext(smsCtx, "[SMS Provider] 发送短信",
		log.String("phone", "+86-138****8888"),
	)

	time.Sleep(40 * time.Millisecond)

	log.InfoContext(smsCtx, "[SMS Provider] 短信发送成功",
		log.String("message_id", fmt.Sprintf("SMS-%d", time.Now().UnixNano())),
	)
}

// sendEmail 发送邮件
func sendEmail(ctx context.Context, orderID string) {
	spanID := idutil.NewSpanID()
	emailCtx := log.WithSpanID(ctx, spanID)

	log.DebugContext(emailCtx, "[Email Provider] 发送邮件",
		log.String("email", "user@example.com"),
	)

	time.Sleep(50 * time.Millisecond)

	log.InfoContext(emailCtx, "[Email Provider] 邮件发送成功",
		log.String("message_id", fmt.Sprintf("EMAIL-%d", time.Now().UnixNano())),
	)
}

// mockResponseWriter 模拟 HTTP 响应写入器
type mockResponseWriter struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (w *mockResponseWriter) Header() http.Header {
	return w.header
}

func (w *mockResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *mockResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
