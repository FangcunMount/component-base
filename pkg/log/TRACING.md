# 链路追踪集成说明

## 概述

链路追踪（Distributed Tracing）是微服务架构中的重要观测手段，通过在服务调用链路中传递唯一的追踪 ID，可以：

1. **关联分布式日志**：通过 trace_id 将分散在不同服务的日志关联起来
2. **定位性能瓶颈**：通过 span 记录每个操作的耗时
3. **分析调用链路**：理解服务间的依赖关系
4. **问题排查**：快速定位故障发生的位置

## 核心概念

### Trace ID
- **长度**：32 字符十六进制字符串
- **用途**：唯一标识一次完整的请求调用链
- **传递**：在整个调用链中保持不变
- **示例**：`51c89d3c4c18d317915cae78f4221908`

### Span ID
- **长度**：16 字符十六进制字符串
- **用途**：标识调用链中的一个操作片段
- **传递**：每次调用子服务时会创建新的 Span ID
- **示例**：`2cc53d25e033223e`

### Request ID
- **格式**：`req-{timestamp}-{random}`
- **用途**：唯一标识一个 HTTP 请求
- **示例**：`req-1761965855567-63dg7rsf`

## 实现架构

### 1. ID 生成（idutil 包）

```go
// pkg/util/idutil/idutil.go

// 使用加密安全的随机数生成器
func NewTraceID() string {
    buf := make([]byte, 16)
    rand.Read(buf)
    return fmt.Sprintf("%032x", buf)
}

func NewSpanID() string {
    buf := make([]byte, 8)
    rand.Read(buf)
    return fmt.Sprintf("%016x", buf)
}

func NewRequestID() string {
    timestamp := time.Now().UnixMilli()
    random := randomString(8)
    return fmt.Sprintf("req-%d-%s", timestamp, random)
}
```

### 2. 上下文传递（trace.go）

```go
// pkg/log/trace.go

// Context Key 定义
type contextKey string

const (
    TraceIDKey   contextKey = "trace_id"
    SpanIDKey    contextKey = "span_id"
    RequestIDKey contextKey = "request_id"
)

// 注入追踪信息
func WithTraceContext(ctx context.Context, traceID, spanID, requestID string) context.Context {
    ctx = context.WithValue(ctx, TraceIDKey, traceID)
    ctx = context.WithValue(ctx, SpanIDKey, spanID)
    ctx = context.WithValue(ctx, RequestIDKey, requestID)
    return ctx
}

// 提取追踪信息
func TraceID(ctx context.Context) string {
    if v := ctx.Value(TraceIDKey); v != nil {
        return v.(string)
    }
    return ""
}
```

### 3. 日志集成（trace.go）

```go
// 带追踪信息的日志方法
func InfoContext(ctx context.Context, msg string, fields ...Field) {
    traceID := TraceID(ctx)
    spanID := SpanID(ctx)
    requestID := RequestID(ctx)
    
    // 自动添加追踪字段
    allFields := []Field{
        String("trace_id", traceID),
        String("span_id", spanID),
        String("request_id", requestID),
    }
    allFields = append(allFields, fields...)
    
    std.Info(msg, allFields...)
}
```

### 4. HTTP 中间件（middleware/tracing.go）

```go
// pkg/log/middleware/tracing.go

func TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. 提取或生成 Trace ID
        traceID := r.Header.Get("X-Trace-Id")
        if traceID == "" {
            traceID = idutil.NewTraceID()
        }
        
        // 2. 生成 Span ID 和 Request ID
        spanID := idutil.NewSpanID()
        requestID := idutil.NewRequestID()
        
        // 3. 注入到 Context
        ctx := log.WithTraceContext(r.Context(), traceID, spanID, requestID)
        
        // 4. 设置响应头
        w.Header().Set("X-Trace-Id", traceID)
        w.Header().Set("X-Request-Id", requestID)
        
        // 5. 记录请求开始
        start := time.Now()
        log.InfoContext(ctx, "HTTP请求开始", ...)
        
        // 6. 执行处理器
        next.ServeHTTP(w, r.WithContext(ctx))
        
        // 7. 记录请求结束
        log.InfoContext(ctx, "HTTP请求完成", 
            log.Float64("duration", time.Since(start).Seconds()),
        )
    })
}
```

## 使用场景

### 场景 1：单体应用内追踪

```go
func handleOrder(ctx context.Context, orderID string) error {
    log.InfoContext(ctx, "开始处理订单", log.String("order_id", orderID))
    
    // 验证库存 - 创建子 span
    spanID := idutil.NewSpanID()
    childCtx := log.WithSpanID(ctx, spanID)
    log.DebugContext(childCtx, "验证库存")
    
    // 处理支付 - 创建另一个子 span
    spanID2 := idutil.NewSpanID()
    childCtx2 := log.WithSpanID(ctx, spanID2)
    log.DebugContext(childCtx2, "处理支付")
    
    log.InfoContext(ctx, "订单处理完成")
    return nil
}
```

日志输出：
```
[INFO] 开始处理订单 {"trace_id": "xxx", "span_id": "parent", ...}
[DEBUG] 验证库存 {"trace_id": "xxx", "span_id": "child1", ...}
[DEBUG] 处理支付 {"trace_id": "xxx", "span_id": "child2", ...}
[INFO] 订单处理完成 {"trace_id": "xxx", "span_id": "parent", ...}
```

### 场景 2：HTTP API 服务

```go
func main() {
    log.Init(log.NewOptions())
    defer log.Flush()
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // 日志自动包含追踪信息
        log.InfoContext(ctx, "处理 API 请求", 
            log.String("path", r.URL.Path),
        )
        
        // 业务逻辑...
        processRequest(ctx)
        
        w.WriteHeader(http.StatusOK)
    })
    
    // 应用追踪中间件
    tracedHandler := middleware.TracingMiddleware(handler)
    
    http.ListenAndServe(":8080", tracedHandler)
}
```

### 场景 3：微服务调用链

```
Client -> API Gateway -> Order Service -> Payment Service
                                      -> Inventory Service
```

**API Gateway**：
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    log.InfoContext(ctx, "[Gateway] 接收请求")
    
    // 调用 Order Service，传递 trace_id
    traceID := log.TraceID(ctx)
    req, _ := http.NewRequest("POST", "http://order-service/api/orders", nil)
    req.Header.Set("X-Trace-Id", traceID)
    
    // 发送请求...
}
```

**Order Service**：
```go
func createOrder(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // 中间件已注入追踪信息
    
    log.InfoContext(ctx, "[Order] 创建订单")
    
    // 调用 Payment Service
    spanID := idutil.NewSpanID()
    childCtx := log.WithSpanID(ctx, spanID)
    callPaymentService(childCtx)
}

func callPaymentService(ctx context.Context) {
    traceID := log.TraceID(ctx)
    spanID := log.SpanID(ctx)
    
    req, _ := http.NewRequest("POST", "http://payment-service/api/pay", nil)
    req.Header.Set("X-Trace-Id", traceID)
    req.Header.Set("X-Span-Id", spanID)
    
    log.InfoContext(ctx, "[Order] 调用支付服务")
    
    // 发送请求...
}
```

**Payment Service**：
```go
func processPayment(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // 继承了 trace_id
    
    log.InfoContext(ctx, "[Payment] 处理支付")
    
    // 所有服务的日志都包含相同的 trace_id
}
```

### 场景 4：日志查询与分析

有了追踪 ID 后，可以轻松查询完整调用链的日志：

```bash
# 查询某个请求的所有日志
grep "trace_id.*51c89d3c4c18d317915cae78f4221908" /var/log/app.log

# 在 Elasticsearch/Kibana 中查询
trace_id: "51c89d3c4c18d317915cae78f4221908"

# 分析调用耗时
{"trace_id": "xxx", "span_id": "parent", "duration": 0.5}
{"trace_id": "xxx", "span_id": "child1", "duration": 0.1}
{"trace_id": "xxx", "span_id": "child2", "duration": 0.3}
```

## 与其他追踪系统集成

### OpenTelemetry

如需与 OpenTelemetry 集成，可以：

1. 从 OTel Span 提取 Trace ID 和 Span ID
2. 注入到我们的 Context
3. 日志自动包含追踪信息

参考 `pkg/log/otel/otel.go` 示例。

### Jaeger/Zipkin

可以将日志中的 trace_id 和 span_id 上报到 Jaeger/Zipkin：

1. 使用相同的 trace_id 格式（符合 OpenTelemetry 规范）
2. 在上报 Span 时使用日志中的 trace_id
3. 实现可视化的调用链路图

## 性能考虑

1. **ID 生成开销**：使用 `crypto/rand`，单次生成约 1-2μs
2. **Context 传递**：零拷贝，性能影响可忽略
3. **日志额外字段**：每条日志增加 3 个字段（约 100 字节）
4. **HTTP Header**：每个请求增加 2-3 个 Header（约 150 字节）

总体性能影响 < 1%，可以放心使用。

## 最佳实践

1. **始终传递 Context**：函数签名包含 `ctx context.Context`
2. **创建子 Span**：关键操作创建新的 Span ID
3. **跨服务传递**：通过 HTTP Header 传递 trace_id
4. **日志使用 Context 方法**：使用 `InfoContext` 等方法
5. **统一 ID 格式**：遵循 OpenTelemetry 规范
6. **记录关键信息**：span 开始/结束时记录日志
7. **错误传播**：错误发生时保持 trace_id 传递

## 运行示例

```bash
# 基础追踪示例
cd /path/to/component-base
go run pkg/log/example/tracing/main.go

# 微服务调用链示例
go run pkg/log/example/microservices/main.go
```

观察日志输出，你会看到：
- 所有日志包含相同的 `trace_id`
- 不同操作有不同的 `span_id`
- 可以清晰追踪请求的完整生命周期

## 总结

链路追踪集成为日志系统增加了分布式环境下的可观测性：

✅ **自动化**：HTTP 中间件自动处理追踪信息  
✅ **轻量级**：无需引入复杂的追踪系统  
✅ **兼容性**：遵循 OpenTelemetry 规范  
✅ **易用性**：Context-based API，简单直观  
✅ **灵活性**：可选集成 OTel、Jaeger 等系统  

开始使用链路追踪，让你的微服务日志更有条理！
