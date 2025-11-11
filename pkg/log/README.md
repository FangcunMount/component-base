# Log Package

一个功能强大的日志包，支持多种日志格式和输出方式。

## 特性

- 支持多种日志格式（JSON、Console）
- 支持多种输出方式（文件、控制台、网络）
- 支持日志级别控制
- 支持结构化日志
- **支持日志轮转**：按大小轮转、按时间轮转（天、小时、月等）
- 支持多种日志库（Zap、Logrus、Klog）
- **支持彩色日志输出**：不同级别的日志显示不同颜色
- **日志级别带方括号**：更清晰的日志级别标识
- **支持日志分级输出**：不同级别日志输出到不同文件（Duplicate 模式推荐）
- **支持类型化日志**：HTTP、SQL、gRPC、Redis 等类型标记，便于分析
- **支持链路追踪**：自动追踪请求调用链

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/FangcunMount/iam-contracts/pkg/log"
)

func main() {
    // 初始化日志
    log.Init(&log.Options{
        Level:      "info",
        Format:     "console",
        OutputPaths: []string{"stdout"},
    })
    defer log.Flush()

    // 使用日志
    log.Info("Hello, World!")
    log.Error("Something went wrong")
}
```

### 配置选项

```go
log.Init(&log.Options{
    Level:      "debug",           // 日志级别
    Format:     "console",         // 日志格式 (console/json)
    EnableColor: true,             // 启用彩色输出（仅 console 格式）
    OutputPaths: []string{         // 输出路径
        "stdout",
        "/var/log/app.log",
    },
    ErrorOutputPaths: []string{    // 错误输出路径
        "stderr",
        "/var/log/app-error.log",
    },
    MaxSize:    100,               // 单个日志文件最大大小（MB）
    MaxAge:     30,                // 保留旧日志文件的最大天数
    MaxBackups: 10,                // 保留旧日志文件的最大数量
    Compress:   true,              // 是否压缩旧日志文件
})
```

### 彩色日志输出

启用彩色输出可以让不同级别的日志更加醒目：

```go
log.Init(&log.Options{
    Level:       "debug",
    Format:      "console",
    EnableColor: true,  // 启用彩色输出
    OutputPaths: []string{"stdout"},
})

// 不同级别的日志将显示不同的颜色
log.Debug("This is a debug message")   // 青色 [DEBUG]
log.Info("This is an info message")    // 绿色 [INFO]
log.Warn("This is a warning message")  // 黄色 [WARN]
log.Error("This is an error message")  // 红色 [ERROR]
```

**注意**：

- 彩色输出仅在 `console` 格式下生效
- 输出到文件时建议关闭彩色输出（`EnableColor: false`）
- 所有日志级别都会带有方括号 `[LEVEL]`，无论是否启用颜色

**颜色方案**：

- `[DEBUG]` - 青色（Cyan）
- `[INFO]` - 绿色（Green）
- `[WARN]` - 黄色（Yellow）
- `[ERROR]` - 红色（Red）
- `[FATAL]` - 红色（Red）
- `[PANIC]` - 红色（Red）

## 日志级别

- `debug`: 调试信息
- `info`: 一般信息
- `warn`: 警告信息
- `error`: 错误信息
- `fatal`: 致命错误（会调用os.Exit(1)）
- `panic`: 恐慌错误（会调用panic）

## 结构化日志

```go
log.Info("User login",
    "user_id", 123,
    "ip", "192.168.1.1",
    "user_agent", "Mozilla/5.0...",
)
```

## 日志轮转

当日志文件达到指定大小时，会自动进行轮转：

```go
log.Init(&log.Options{
    OutputPaths: []string{"/var/log/app.log"},
    MaxSize:    100,    // 100MB
    MaxAge:     30,     // 30天
    MaxBackups: 10,     // 保留10个旧文件
    Compress:   true,   // 压缩旧文件
})
```

### 按时间轮转

支持按时间自动轮转日志文件，可以按天、按小时、按月等方式分割日志。

#### 按天轮转

```go
opts := log.NewOptions()
opts.EnableTimeRotation = true
opts.TimeRotationFormat = "2006-01-02"  // 按天轮转
opts.MaxAge = 7                          // 保留7天
opts.OutputPaths = []string{"/var/log/app.log"}
log.Init(opts)
// 生成的文件名：app.2025-11-01.log, app.2025-11-02.log, ...
```

#### 按小时轮转

```go
opts := log.NewOptions()
opts.EnableTimeRotation = true
opts.TimeRotationFormat = "2006-01-02-15"  // 按小时轮转
opts.MaxAge = 24                            // 保留24小时
opts.OutputPaths = []string{"/var/log/app.log"}
log.Init(opts)
// 生成的文件名：app.2025-11-01-10.log, app.2025-11-01-11.log, ...
```

**其他支持的时间格式**：

- `2006-01-02`: 按天轮转
- `2006-01-02-15`: 按小时轮转
- `2006-01`: 按月轮转
- `2006-W01`: 按周轮转（需要自定义）

**优势**：

- 日志文件按时间自然分割，便于查找和分析
- 适合长时间运行的服务
- 可以根据业务需求设置不同的轮转粒度
- 自动清理过期日志，节省磁盘空间

**示例**：

```bash
# 运行按天轮转示例
go run pkg/log/example/dailyrotation/main.go

# 运行按小时轮转示例
go run pkg/log/example/hourlyrotation/main.go
```

## 日志分级输出

支持将不同级别的日志输出到不同的文件，便于日志分析和问题排查。

### 基本用法

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "duplicate" // 推荐使用 duplicate 模式

// 配置输出路径
opts.LevelOutputPaths = map[string][]string{
    "all":   []string{"/var/log/app.log"},   // 记录所有日志
    "error": []string{"/var/log/error.log"}, // 额外记录错误
}

log.Init(opts)
```

### 输出模式

#### 1. Duplicate 模式（推荐，默认）

支持重复输出，适合生产环境：

```go
opts.LevelOutputMode = "duplicate"
opts.LevelOutputPaths = map[string][]string{
    "all":   []string{"/var/log/app.log"},   // 记录所有级别日志
    "error": []string{"/var/log/error.log"}, // 额外记录 ERROR
    "warn":  []string{"/var/log/warn.log"},  // 额外记录 WARN（可选）
}
```

**输出结果**：
- `app.log`: 包含 DEBUG, INFO, WARN, ERROR（完整日志）
- `error.log`: 只包含 ERROR（便于快速定位故障）
- `warn.log`: 只包含 WARN（便于发现潜在问题）

**适用场景**：
- ✅ **生产环境推荐**：完整日志 + 错误日志分离
- ✅ 需要快速定位错误，同时保留完整上下文
- ✅ 配合监控系统，对 error.log 设置告警

#### 2. Above 模式

输出该级别及以上的日志：

```go
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "info":  []string{"/var/log/info.log"},  // 包含: INFO, WARN, ERROR
    "error": []string{"/var/log/error.log"}, // 包含: ERROR
}
```

**输出结果**：
- `info.log`: INFO + WARN + ERROR
- `error.log`: 仅 ERROR

**适用场景**：
- 需要在一个文件中查看所有重要日志
- 错误日志单独存储

⚠️ **注意**：此模式下日志会重复（info.log 和 error.log 都包含 ERROR）

#### 3. Exact 模式

只输出精确匹配的日志级别：

```go
opts.LevelOutputMode = "exact"
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"/var/log/debug.log"},  // 仅 DEBUG
    "info":  []string{"/var/log/info.log"},   // 仅 INFO
    "warn":  []string{"/var/log/warn.log"},   // 仅 WARN
    "error": []string{"/var/log/error.log"},  // 仅 ERROR
}
```

**输出结果**：
- 每个文件只包含对应级别的日志
- 没有重复，严格分离

**适用场景**：
- 需要精确统计各级别日志数量
- 不同级别日志需要不同的处理策略

### 生产环境推荐配置

```go
opts := log.NewOptions()
opts.Level = "info"                  // INFO 及以上
opts.Format = "json"                 // JSON 格式便于分析
opts.EnableLevelOutput = true
opts.LevelOutputMode = "duplicate"   // 使用 duplicate 模式
opts.EnableTimeRotation = true       // 启用按天轮转
opts.TimeRotationFormat = "2006-01-02"
opts.MaxAge = 30                     // 保留 30 天
opts.Compress = true                 // 压缩旧日志

opts.LevelOutputPaths = map[string][]string{
    "all":   []string{"/var/log/app.log"},   // 完整日志
    "error": []string{"/var/log/error.log"}, // 错误日志（设置告警）
}

log.Init(opts)
```

**优势**：
- ✅ `app.log` 包含完整日志，便于问题追溯
- ✅ `error.log` 只含错误，便于快速定位
- ✅ 按天轮转，自动管理磁盘空间
- ✅ JSON 格式，便于 ELK/日志分析工具处理

### 示例

查看各种模式的完整示例：

```bash
# Duplicate 模式（推荐）
go run pkg/log/example/duplicatemode/main.go

# 生产环境配置
go run pkg/log/example/production/main.go

# Above 模式
go run pkg/log/example/leveloutput/main.go

# Exact 模式
go run pkg/log/example/exactlevel/main.go
```

# 运行 exact 模式示例
go run pkg/log/example/exactlevel/main.go
```

## 多种日志库支持

### Zap

```go
import "github.com/FangcunMount/iam-contracts/pkg/log"

log.Init(&log.Options{
    Level:      "info",
    Format:     "json",
    OutputPaths: []string{"stdout"},
})
```

### Logrus

```go
import "github.com/FangcunMount/iam-contracts/pkg/log/logrus"

logger := logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{})
logger.Info("Hello, World!")
```

### Klog

```go
import "github.com/FangcunMount/iam-contracts/pkg/log/klog"

klog.InitFlags(nil)
klog.Info("Hello, World!")
klog.Flush()
```

## 开发工具

### 开发环境日志

```go
log.Init(&log.Options{
    Level:      "debug",
    Format:     "console",
    OutputPaths: []string{"stdout"},
    Development: true,  // 开发模式
})
```

### 测试环境日志

```go
log.Init(&log.Options{
    Level:      "info",
    Format:     "json",
    OutputPaths: []string{"/var/log/test.log"},
    Development: false,
})
```

## 链路追踪集成

### 基本使用

```go
import (
    "context"
    "github.com/FangcunMount/component-base/pkg/log"
    "github.com/FangcunMount/component-base/pkg/util/idutil"
)

func main() {
    log.Init(log.NewOptions())
    defer log.Flush()

    // 创建追踪上下文
    ctx := context.Background()
    traceID := idutil.NewTraceID()    // 生成 32 字符 Trace ID
    spanID := idutil.NewSpanID()      // 生成 16 字符 Span ID
    requestID := idutil.NewRequestID() // 生成请求 ID
    
    ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)
    
    // 使用带追踪信息的日志
    log.InfoContext(ctx, "处理订单", log.String("order_id", "ORD-123"))
    log.DebugContext(ctx, "验证库存")
    log.ErrorContext(ctx, "库存不足")
}
```

### HTTP 中间件集成

```go
import (
    "net/http"
    "github.com/FangcunMount/component-base/pkg/log"
    "github.com/FangcunMount/component-base/pkg/log/middleware"
)

func main() {
    log.Init(log.NewOptions())
    defer log.Flush()
    
    // 创建 HTTP 处理器
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // 日志会自动包含 trace_id, span_id, request_id
        log.InfoContext(ctx, "处理请求", 
            log.String("path", r.URL.Path),
            log.String("method", r.Method),
        )
        
        w.WriteHeader(http.StatusOK)
    })
    
    // 使用追踪中间件
    tracedHandler := middleware.TracingMiddleware(handler)
    
    http.ListenAndServe(":8080", tracedHandler)
}
```

### 微服务调用链追踪

```go
// 服务 A 调用服务 B
func callServiceB(ctx context.Context) {
    // 创建子 span
    spanID := idutil.NewSpanID()
    childCtx := log.WithSpanID(ctx, spanID)
    
    log.InfoContext(childCtx, "调用服务 B", log.String("service", "service-b"))
    
    // 调用服务 B...
    // 将 trace_id 和 span_id 通过 HTTP Header 传递
}
```

### 追踪上下文 API

```go
// 注入完整追踪信息
ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

// 只更新 Span ID（创建子 span）
ctx = log.WithSpanID(ctx, newSpanID)

// 获取追踪信息
traceID := log.TraceID(ctx)
spanID := log.SpanID(ctx)
requestID := log.RequestID(ctx)

// 带追踪信息的日志方法
log.InfoContext(ctx, "message", fields...)
log.DebugContext(ctx, "message", fields...)
log.WarnContext(ctx, "message", fields...)
log.ErrorContext(ctx, "message", fields...)
```

### ID 生成工具

```go
import "github.com/FangcunMount/component-base/pkg/util/idutil"

// 生成 Trace ID（32 字符十六进制，符合 OpenTelemetry 规范）
traceID := idutil.NewTraceID()  // 例如：51c89d3c4c18d317915cae78f4221908

// 生成 Span ID（16 字符十六进制）
spanID := idutil.NewSpanID()    // 例如：2cc53d25e033223e

// 生成 Request ID（带时间戳）
requestID := idutil.NewRequestID()  // 例如：req-1761965855567-63dg7rsf
```

### HTTP 中间件特性

追踪中间件自动处理：

1. **提取追踪信息**：从 HTTP 请求头提取现有的 trace_id
2. **生成追踪 ID**：如果请求没有 trace_id，自动生成新的
3. **注入到上下文**：将追踪信息注入 context.Context
4. **响应头返回**：在响应中添加 X-Trace-Id 和 X-Request-Id 头
5. **自动记录**：记录请求开始和结束，包含耗时

支持的 HTTP Header：
- `X-Trace-Id`：分布式追踪 ID
- `X-Span-Id`：当前 span ID
- `X-Request-Id`：请求 ID

### OpenTelemetry 集成（可选）

如果需要与 OpenTelemetry 集成，可以参考 `pkg/log/otel/otel.go` 中的示例代码。

### 示例程序

查看完整示例：
- `pkg/log/example/tracing/main.go` - 基础追踪示例
- `pkg/log/example/microservices/main.go` - 微服务调用链示例

运行示例：
```bash
# 基础追踪示例
go run pkg/log/example/tracing/main.go

# 微服务示例
go run pkg/log/example/microservices/main.go
```

## 类型化日志

支持为不同类型的操作添加自动类型标记，便于日志分类和分析。

### 支持的类型

- **HTTP** - HTTP/REST API 请求日志
- **SQL** - 数据库查询日志
- **GRPC** - gRPC 服务调用日志
- **Redis** - Redis 缓存操作日志
- **MQ** - 消息队列操作日志
- **Cache** - 通用缓存操作日志
- **RPC** - 通用 RPC 调用日志

### 基本使用

```go
// HTTP 请求日志（自动添加 type=HTTP）
log.HTTP("GET /api/users",
    log.String("method", "GET"),
    log.String("path", "/api/users"),
    log.Int("status", 200),
    log.Float64("duration_ms", 45.6),
)

// SQL 查询日志（自动添加 type=SQL）
log.SQL("查询用户信息",
    log.String("query", "SELECT * FROM users WHERE id = ?"),
    log.Int("rows", 1),
    log.Float64("duration_ms", 12.3),
)

// gRPC 调用日志（自动添加 type=GRPC）
log.GRPC("UserService.GetUser",
    log.String("service", "UserService"),
    log.String("method", "GetUser"),
    log.String("code", "OK"),
)

// Redis 操作日志（自动添加 type=Redis）
log.Redis("GET user:10086",
    log.String("command", "GET"),
    log.String("key", "user:10086"),
    log.Bool("hit", true),
)
```

### 不同级别

每种类型都提供 4 个级别：

```go
log.HTTP()       // INFO 级别
log.HTTPDebug()  // DEBUG 级别
log.HTTPWarn()   // WARN 级别
log.HTTPError()  // ERROR 级别
```

### 实际应用

```go
func HandleOrder(w http.ResponseWriter, r *http.Request) {
    // 记录 HTTP 请求
    log.HTTP("接收订单请求",
        log.String("method", r.Method),
        log.String("path", r.URL.Path),
    )
    
    // 查询数据库
    log.SQL("查询用户信息",
        log.String("query", "SELECT * FROM users WHERE id = ?"),
        log.Float64("duration_ms", 5.2),
    )
    
    // 检查缓存
    log.Redis("检查库存缓存",
        log.String("key", "inventory:PROD-12345"),
        log.Bool("hit", true),
    )
    
    // 调用服务
    log.GRPC("调用支付服务",
        log.String("service", "PaymentService"),
        log.String("method", "Charge"),
    )
    
    // 发送消息
    log.MQ("发送订单事件",
        log.String("topic", "order-created"),
    )
}
```

### 日志分析

类型化日志便于过滤和分析：

```bash
# 只查看 HTTP 日志
cat app.log | grep '"type":"HTTP"'

# 只查看 SQL 日志
cat app.log | grep '"type":"SQL"'

# 统计各类型日志数量
cat app.log | jq -r '.type' | sort | uniq -c

# 计算 HTTP 平均响应时间
cat app.log | grep '"type":"HTTP"' | jq '.duration_ms' | awk '{sum+=$1; count++} END {print sum/count}'
```

### 详细文档

查看完整的类型化日志文档：[TYPED_LOGS.md](TYPED_LOGS.md)

运行示例：
```bash
go run pkg/log/example/typed-logs/main.go
```

## 最佳实践

1. **选择合适的日志级别**：不要在生产环境使用debug级别
2. **使用结构化日志**：便于日志分析和搜索
3. **配置日志轮转**：避免日志文件过大
4. **分离错误日志**：将错误日志输出到单独的文件（使用 Duplicate 模式）
5. **使用有意义的日志消息**：便于问题排查
6. **使用链路追踪**：在微服务架构中使用追踪 ID 关联日志
7. **传递追踪上下文**：跨服务调用时通过 HTTP Header 传递追踪信息
8. **创建子 Span**：在关键操作时创建新的 Span ID，便于定位问题
9. **使用类型化日志**：为不同操作使用对应的日志类型（HTTP、SQL、gRPC 等）
10. **记录关键字段**：duration、status、error 等便于监控和分析

## 许可证

MIT License
