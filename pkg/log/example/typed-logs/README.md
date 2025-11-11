# 类型化日志功能总结

## 概述

为 log 包添加了类型化日志功能，提供 **HTTP**、**SQL**、**GRPC**、**Redis**、**MQ**、**Cache**、**RPC** 等 7 种类型的专用日志函数。

## 实现的功能

### 1. 核心函数（28个）

每种类型提供 4 个级别的函数：

| 类型 | INFO | DEBUG | WARN | ERROR |
|------|------|-------|------|-------|
| HTTP | `HTTP()` | `HTTPDebug()` | `HTTPWarn()` | `HTTPError()` |
| SQL | `SQL()` | `SQLDebug()` | `SQLWarn()` | `SQLError()` |
| GRPC | `GRPC()` | `GRPCDebug()` | `GRPCWarn()` | `GRPCError()` |
| Redis | `Redis()` | `RedisDebug()` | `RedisWarn()` | `RedisError()` |
| MQ | `MQ()` | `MQDebug()` | `MQWarn()` | `MQError()` |
| Cache | `Cache()` | `CacheDebug()` | `CacheWarn()` | `CacheError()` |
| RPC | `RPC()` | `RPCDebug()` | `RPCWarn()` | `RPCError()` |

### 2. 自动添加 type 字段

每个函数都会自动添加对应的 `type` 字段：

```go
log.HTTP("GET /api/users", ...)
// 输出: {"type": "HTTP", "message": "GET /api/users", ...}

log.SQL("查询用户", ...)
// 输出: {"type": "SQL", "message": "查询用户", ...}
```

### 3. 文档和示例

- ✅ 创建了详细的使用文档：`TYPED_LOGS.md`
- ✅ 创建了完整示例：`pkg/log/example/typed-logs/main.go`
- ✅ 更新了 README，添加类型化日志章节

## 使用方式

### 基本使用

```go
import "github.com/FangcunMount/component-base/pkg/log"

// HTTP 请求
log.HTTP("GET /api/users",
    log.String("method", "GET"),
    log.String("path", "/api/users"),
    log.Int("status", 200),
    log.Float64("duration_ms", 45.6),
)

// SQL 查询
log.SQL("查询用户信息",
    log.String("query", "SELECT * FROM users WHERE id = ?"),
    log.Int("rows", 1),
    log.Float64("duration_ms", 12.3),
)

// gRPC 调用
log.GRPC("UserService.GetUser",
    log.String("service", "UserService"),
    log.String("method", "GetUser"),
    log.String("code", "OK"),
)
```

### 不同级别

```go
// INFO 级别（正常操作）
log.HTTP("请求成功", ...)

// DEBUG 级别（调试信息）
log.HTTPDebug("请求详情", ...)

// WARN 级别（警告）
log.HTTPWarn("响应慢", ...)

// ERROR 级别（错误）
log.HTTPError("请求失败", ...)
```

### 实际业务场景

```go
func HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
    // 1. HTTP 请求
    log.HTTP("接收订单请求",
        log.String("method", r.Method),
        log.String("path", r.URL.Path),
    )
    
    // 2. SQL 查询
    log.SQL("查询用户信息",
        log.String("query", "SELECT * FROM users WHERE id = ?"),
        log.Float64("duration_ms", 5.2),
    )
    
    // 3. Redis 操作
    log.Redis("检查库存缓存",
        log.String("key", "inventory:PROD-12345"),
        log.Bool("hit", true),
    )
    
    // 4. gRPC 调用
    log.GRPC("调用支付服务",
        log.String("service", "PaymentService"),
        log.String("method", "Charge"),
        log.String("code", "OK"),
    )
    
    // 5. 消息队列
    log.MQ("发送订单事件",
        log.String("topic", "order-created"),
        log.String("order_id", orderID),
    )
    
    // 6. 返回响应
    log.HTTP("订单创建成功",
        log.String("order_id", orderID),
        log.Int("status", 200),
    )
}
```

## 优势对比

### 传统方式

```go
// ❌ 需要手动添加 type 字段
log.Info("GET /api/users",
    log.String("type", "HTTP"),  // 容易忘记
    log.String("method", "GET"),
)

// ❌ type 值容易不统一
log.Info("查询", log.String("type", "sql"))      // 小写
log.Info("查询", log.String("type", "SQL"))      // 大写
log.Info("查询", log.String("type", "database")) // 完全不同
```

### 类型化日志

```go
// ✅ 自动添加 type 字段
log.HTTP("GET /api/users",
    log.String("method", "GET"),
)

// ✅ type 值自动统一
// ✅ 函数名表达意图清晰
// ✅ IDE 自动补全友好
```

## 日志分析示例

### 命令行过滤

```bash
# 只查看 HTTP 日志
cat app.log | grep '"type":"HTTP"'

# 只查看 SQL 慢查询（> 1000ms）
cat app.log | grep '"type":"SQL"' | grep -E '"duration_ms":[0-9]{4,}'

# 统计各类型日志数量
cat app.log | jq -r '.type' | sort | uniq -c

# 计算 HTTP 平均响应时间
cat app.log | grep '"type":"HTTP"' | jq '.duration_ms' | \
    awk '{sum+=$1; count++} END {print sum/count}'
```

### ELK/Kibana 分析

在 Kibana 中创建仪表盘：

1. **HTTP 监控面板**
   - 过滤：`type: "HTTP"`
   - 图表：QPS、响应时间、状态码分布

2. **SQL 性能面板**
   - 过滤：`type: "SQL"`
   - 图表：查询数量、慢查询数、平均查询时间

3. **gRPC 调用面板**
   - 过滤：`type: "GRPC"`
   - 图表：成功率、错误码分布、调用时间

### Grafana 告警

```yaml
# HTTP 错误率告警
- name: http_error_rate
  query: rate(log_entries{type="HTTP", level="ERROR"}[5m])
  threshold: 0.01  # 1% 错误率

# SQL 慢查询告警
- name: sql_slow_query
  query: count(log_entries{type="SQL", duration_ms>1000}[5m])
  threshold: 10

# gRPC 失败告警
- name: grpc_failure
  query: rate(log_entries{type="GRPC", level="ERROR"}[5m])
  threshold: 0.05  # 5% 失败率
```

## 示例程序输出

运行 `go run pkg/log/example/typed-logs/main.go`：

```json
{"level":"[INFO]","type":"HTTP","message":"GET /api/users","method":"GET","path":"/api/users","status":200,"duration_ms":45.6}
{"level":"[INFO]","type":"SQL","message":"查询用户信息","query":"SELECT * FROM users WHERE id = ?","rows":1,"duration_ms":12.3}
{"level":"[INFO]","type":"GRPC","message":"UserService.GetUser","service":"UserService","method":"GetUser","code":"OK"}
{"level":"[INFO]","type":"Redis","message":"GET user:10086","command":"GET","key":"user:10086","hit":true}
{"level":"[INFO]","type":"MQ","message":"发送消息","topic":"order-events","message_id":"MSG-001"}
```

## 适用场景

### 1. 微服务架构

```go
// 服务 A
log.HTTP("接收请求", ...)
log.GRPC("调用服务 B", ...)

// 服务 B
log.SQL("查询数据", ...)
log.Redis("缓存操作", ...)
```

### 2. API 网关

```go
log.HTTP("接收客户端请求", ...)
log.RPC("转发到后端服务", ...)
log.HTTP("返回响应", ...)
```

### 3. 数据处理服务

```go
log.MQ("接收消息", ...)
log.SQL("查询数据", ...)
log.SQL("更新数据", ...)
log.Cache("更新缓存", ...)
log.MQ("发送结果", ...)
```

### 4. 监控和性能分析

```go
// 自动记录耗时
start := time.Now()
result := doSomething()
duration := time.Since(start).Milliseconds()

if duration > 1000 {
    log.SQLWarn("慢查询",
        log.String("query", "..."),
        log.Float64("duration_ms", float64(duration)),
    )
}
```

## 最佳实践

### 1. 记录关键信息

```go
// ✅ HTTP 日志记录
log.HTTP("API 请求",
    log.String("method", "POST"),     // 请求方法
    log.String("path", "/api/orders"), // 请求路径
    log.Int("status", 200),           // 状态码
    log.Float64("duration_ms", 45.6), // 响应时间
    log.String("trace_id", "..."),    // 追踪ID
)

// ✅ SQL 日志记录
log.SQL("数据库操作",
    log.String("query", "SELECT..."), // SQL语句
    log.Int("rows", 100),            // 影响行数
    log.Float64("duration_ms", 12.3), // 执行时间
)
```

### 2. 性能考虑

```go
// ✅ 只在需要时记录详细信息
if log.V(1).Enabled() {
    log.HTTPDebug("详细信息", ...)
}

// ✅ 批量操作只记录汇总
log.SQL("批量插入", log.Int("count", len(items)))
```

### 3. 与追踪集成

```go
// 结合追踪ID使用
log.HTTP("请求处理",
    log.String("trace_id", traceID),
    log.String("span_id", spanID),
    // ...
)
```

## 文件清单

1. **核心实现**：
   - `pkg/log/log.go` - 添加了 28 个类型化日志函数

2. **文档**：
   - `pkg/log/TYPED_LOGS.md` - 详细使用文档
   - `pkg/log/README.md` - 添加了类型化日志章节

3. **示例**：
   - `pkg/log/example/typed-logs/main.go` - 完整示例程序

## 总结

类型化日志为不同操作提供了标准化的日志记录方式：

✅ **自动化**：自动添加 type 字段，减少重复代码  
✅ **标准化**：统一的类型值，便于分析  
✅ **可读性**：函数名直接表达意图  
✅ **可分析**：便于日志聚合和性能监控  
✅ **易维护**：类型集中管理，修改方便

推荐在生产环境中使用类型化日志，提升系统的可观测性！
