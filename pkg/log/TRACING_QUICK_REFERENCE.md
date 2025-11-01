# 链路追踪快速参考

## 基础使用

### 1. 手动创建追踪上下文

```go
import (
    "context"
    "github.com/FangcunMount/component-base/pkg/log"
    "github.com/FangcunMount/component-base/pkg/util/idutil"
)

// 创建追踪上下文
ctx := context.Background()
traceID := idutil.NewTraceID()
spanID := idutil.NewSpanID()
requestID := idutil.NewRequestID()

ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

// 使用带追踪的日志
log.InfoContext(ctx, "订单创建", log.String("order_id", "123"))
```

### 2. HTTP 服务自动追踪

```go
import (
    "net/http"
    "github.com/FangcunMount/component-base/pkg/log"
    "github.com/FangcunMount/component-base/pkg/log/middleware"
)

func main() {
    log.Init(log.NewOptions())
    defer log.Flush()
    
    handler := http.HandlerFunc(yourHandler)
    
    // 应用追踪中间件
    http.ListenAndServe(":8080", middleware.TracingMiddleware(handler))
}

func yourHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // 已包含追踪信息
    
    log.InfoContext(ctx, "处理请求")
}
```

### 3. 创建子 Span

```go
func processOrder(ctx context.Context) {
    log.InfoContext(ctx, "开始处理订单")
    
    // 验证库存 - 创建子 span
    spanID := idutil.NewSpanID()
    childCtx := log.WithSpanID(ctx, spanID)
    log.DebugContext(childCtx, "验证库存")
    
    // 处理支付 - 另一个子 span
    spanID2 := idutil.NewSpanID()
    paymentCtx := log.WithSpanID(ctx, spanID2)
    log.DebugContext(paymentCtx, "处理支付")
}
```

### 4. 跨服务传递追踪信息

```go
// Service A
func callServiceB(ctx context.Context) {
    traceID := log.TraceID(ctx)
    spanID := log.SpanID(ctx)
    
    req, _ := http.NewRequest("POST", "http://service-b/api", nil)
    req.Header.Set("X-Trace-Id", traceID)
    req.Header.Set("X-Span-Id", spanID)
    
    // 发送请求...
}

// Service B (使用中间件会自动提取)
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // 包含从 Header 提取的 trace_id
    log.InfoContext(ctx, "收到请求")
}
```

## API 参考

### Context 操作

```go
// 注入完整追踪信息
ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

// 只更新 Span ID
ctx = log.WithSpanID(ctx, newSpanID)

// 获取追踪信息
traceID := log.TraceID(ctx)
spanID := log.SpanID(ctx)
requestID := log.RequestID(ctx)
```

### 日志方法

```go
log.InfoContext(ctx, "message", fields...)
log.DebugContext(ctx, "message", fields...)
log.WarnContext(ctx, "message", fields...)
log.ErrorContext(ctx, "message", fields...)
```

### ID 生成

```go
import "github.com/FangcunMount/component-base/pkg/util/idutil"

traceID := idutil.NewTraceID()    // 32 字符十六进制
spanID := idutil.NewSpanID()      // 16 字符十六进制
requestID := idutil.NewRequestID() // req-{timestamp}-{random}
```

## HTTP Header

| Header | 说明 | 示例 |
|--------|------|------|
| X-Trace-Id | 追踪 ID，贯穿整个调用链 | 51c89d3c4c18d317915cae78f4221908 |
| X-Span-Id | 当前操作的 Span ID | 2cc53d25e033223e |
| X-Request-Id | 请求 ID | req-1761965855567-63dg7rsf |

## 日志输出示例

```json
{
  "level": "info",
  "ts": "2025-11-01T10:57:35.567Z",
  "msg": "处理订单",
  "trace_id": "51c89d3c4c18d317915cae78f4221908",
  "span_id": "2cc53d25e033223e",
  "request_id": "req-1761965855567-63dg7rsf",
  "order_id": "ORD-12345"
}
```

## 常见模式

### 模式 1: API 端点
```go
func CreateOrder(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log.InfoContext(ctx, "创建订单 API")
    
    order := parseRequest(r)
    
    if err := validateOrder(ctx, order); err != nil {
        log.ErrorContext(ctx, "订单验证失败", log.Error(err))
        return
    }
    
    orderID := saveOrder(ctx, order)
    log.InfoContext(ctx, "订单创建成功", log.String("order_id", orderID))
}
```

### 模式 2: 数据库操作
```go
func saveOrder(ctx context.Context, order *Order) string {
    spanID := idutil.NewSpanID()
    dbCtx := log.WithSpanID(ctx, spanID)
    
    log.DebugContext(dbCtx, "插入订单数据")
    
    // 执行数据库操作...
    
    log.DebugContext(dbCtx, "订单数据已保存")
    return orderID
}
```

### 模式 3: 外部 API 调用
```go
func callPaymentAPI(ctx context.Context, amount float64) error {
    spanID := idutil.NewSpanID()
    payCtx := log.WithSpanID(ctx, spanID)
    
    traceID := log.TraceID(ctx)
    
    req, _ := http.NewRequest("POST", paymentURL, nil)
    req.Header.Set("X-Trace-Id", traceID)
    req.Header.Set("X-Span-Id", spanID)
    
    log.InfoContext(payCtx, "调用支付 API", log.Float64("amount", amount))
    
    // 发送请求...
    
    log.InfoContext(payCtx, "支付 API 响应")
    return nil
}
```

## 运行示例

```bash
# 基础示例
go run pkg/log/example/tracing/main.go

# 微服务示例
go run pkg/log/example/microservices/main.go
```

## 查询日志

```bash
# 按 trace_id 查询
grep "trace_id.*51c89d3c" /var/log/app.log

# 按 request_id 查询
grep "request_id.*req-1761965855567" /var/log/app.log

# Elasticsearch 查询
GET /logs/_search
{
  "query": {
    "term": {
      "trace_id": "51c89d3c4c18d317915cae78f4221908"
    }
  }
}
```

## 最佳实践

✅ **DO**
- 所有函数接收 `context.Context` 参数
- 关键操作创建新的 Span ID
- 跨服务调用传递 Trace ID
- 使用 `*Context` 日志方法

❌ **DON'T**
- 不要修改已有的 Trace ID
- 不要在循环中创建 Span
- 不要忘记传递 Context
- 不要使用普通日志方法（无追踪信息）

## 性能影响

- ID 生成: ~1-2 μs
- Context 传递: 可忽略
- 额外字段: ~100 bytes/log
- HTTP Header: ~150 bytes/request

**总体 < 1% 性能影响**
