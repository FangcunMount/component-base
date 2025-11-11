# 类型化日志使用指南

## 概述

类型化日志为不同类型的操作提供了专门的日志记录函数，自动添加 `type` 字段，便于日志分类、过滤和分析。

## 支持的日志类型

| 类型 | 函数前缀 | type 字段值 | 用途 |
|------|---------|------------|------|
| HTTP 请求 | `HTTP*` | `HTTP` | 记录 HTTP/REST API 请求 |
| SQL 查询 | `SQL*` | `SQL` | 记录数据库查询 |
| gRPC 调用 | `GRPC*` | `GRPC` | 记录 gRPC 服务调用 |
| Redis 操作 | `Redis*` | `Redis` | 记录 Redis 缓存操作 |
| 消息队列 | `MQ*` | `MQ` | 记录 Kafka/RabbitMQ 等操作 |
| 缓存操作 | `Cache*` | `Cache` | 记录通用缓存操作 |
| RPC 调用 | `RPC*` | `RPC` | 记录通用 RPC 调用 |

每种类型都提供 4 个级别的函数：
- `Type()` - INFO 级别
- `TypeDebug()` - DEBUG 级别
- `TypeWarn()` - WARN 级别
- `TypeError()` - ERROR 级别

## 基本使用

### HTTP 请求日志

```go
// INFO 级别
log.HTTP("GET /api/users",
    log.String("method", "GET"),
    log.String("path", "/api/users"),
    log.Int("status", 200),
    log.Float64("duration_ms", 45.6),
)

// DEBUG 级别
log.HTTPDebug("请求头详情",
    log.String("user_agent", "Mozilla/5.0"),
)

// WARN 级别
log.HTTPWarn("响应慢",
    log.String("path", "/api/orders"),
    log.Float64("duration_ms", 1523.4),
)

// ERROR 级别
log.HTTPError("请求失败",
    log.String("path", "/api/payment"),
    log.Int("status", 500),
    log.String("error", "internal server error"),
)
```

**输出示例**：
```json
{
  "level": "[INFO]",
  "timestamp": "2025-11-11 10:54:12.046",
  "message": "GET /api/users",
  "type": "HTTP",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "duration_ms": 45.6
}
```

### SQL 查询日志

```go
log.SQL("查询用户信息",
    log.String("query", "SELECT * FROM users WHERE id = ?"),
    log.Int("rows", 1),
    log.Float64("duration_ms", 12.3),
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
```

### gRPC 调用日志

```go
log.GRPC("UserService.GetUser",
    log.String("service", "UserService"),
    log.String("method", "GetUser"),
    log.String("code", "OK"),
    log.Float64("duration_ms", 23.5),
)

log.GRPCError("调用失败",
    log.String("service", "PaymentService"),
    log.String("method", "Charge"),
    log.String("code", "UNAVAILABLE"),
    log.String("error", "connection refused"),
)
```

### Redis 操作日志

```go
log.Redis("GET user:10086",
    log.String("command", "GET"),
    log.String("key", "user:10086"),
    log.Bool("hit", true),
    log.Float64("duration_ms", 1.2),
)

log.RedisError("连接失败",
    log.String("host", "redis.example.com:6379"),
    log.String("error", "dial timeout"),
)
```

### 消息队列日志

```go
log.MQ("发送消息",
    log.String("topic", "order-events"),
    log.String("message_id", "MSG-001"),
    log.Int("size", 1024),
)

log.MQWarn("消息积压",
    log.String("topic", "user-events"),
    log.Int("pending", 5000),
    log.Int("threshold", 1000),
)
```

### 缓存操作日志

```go
log.Cache("缓存命中",
    log.String("key", "product:12345"),
    log.Int("ttl", 3600),
)

log.CacheError("缓存删除失败",
    log.String("key", "temp:xyz789"),
    log.String("error", "key not found"),
)
```

## 实际应用场景

### 场景 1：完整的 HTTP 请求处理流程

```go
func HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
    // 1. 记录请求开始
    log.HTTP("接收订单请求",
        log.String("method", r.Method),
        log.String("path", r.URL.Path),
    )
    
    // 2. 查询数据库
    log.SQL("查询用户信息",
        log.String("query", "SELECT * FROM users WHERE id = ?"),
        log.Float64("duration_ms", 5.2),
    )
    
    // 3. 检查缓存
    log.Redis("检查库存缓存",
        log.String("key", "inventory:PROD-12345"),
        log.Bool("hit", true),
    )
    
    // 4. 调用支付服务
    log.GRPC("调用支付服务",
        log.String("service", "PaymentService"),
        log.String("method", "Charge"),
        log.String("code", "OK"),
    )
    
    // 5. 发送消息
    log.MQ("发送订单事件",
        log.String("topic", "order-created"),
        log.String("order_id", orderID),
    )
    
    // 6. 记录响应
    log.HTTP("订单创建成功",
        log.String("order_id", orderID),
        log.Int("status", 200),
    )
}
```

### 场景 2：数据库性能监控

```go
func QueryWithMonitoring(query string, args ...interface{}) ([]Row, error) {
    start := time.Now()
    rows, err := db.Query(query, args...)
    duration := time.Since(start).Milliseconds()
    
    if err != nil {
        log.SQLError("查询失败",
            log.String("query", query),
            log.String("error", err.Error()),
        )
        return nil, err
    }
    
    if duration > 1000 {
        log.SQLWarn("慢查询",
            log.String("query", query),
            log.Float64("duration_ms", float64(duration)),
        )
    } else {
        log.SQL("查询成功",
            log.String("query", query),
            log.Int("rows", len(rows)),
            log.Float64("duration_ms", float64(duration)),
        )
    }
    
    return rows, nil
}
```

### 场景 3：微服务调用链追踪

```go
// Service A
func ProcessOrder(ctx context.Context, orderID string) error {
    log.HTTP("处理订单请求", log.String("order_id", orderID))
    
    // 调用库存服务
    log.GRPC("调用库存服务",
        log.String("service", "InventoryService"),
        log.String("method", "CheckStock"),
    )
    
    // 调用支付服务
    log.GRPC("调用支付服务",
        log.String("service", "PaymentService"),
        log.String("method", "CreateCharge"),
    )
    
    // 发送通知
    log.MQ("发送订单通知",
        log.String("topic", "order-notifications"),
    )
    
    return nil
}
```

## 日志分析

### 命令行过滤

```bash
# 只查看 HTTP 日志
cat app.log | grep '"type":"HTTP"'

# 只查看 SQL 日志
cat app.log | grep '"type":"SQL"'

# 查看所有错误级别的 HTTP 日志
cat app.log | grep '"type":"HTTP"' | grep '"level":"\[ERROR\]"'

# 查看慢查询（duration_ms > 1000）
cat app.log | grep '"type":"SQL"' | grep -E '"duration_ms":[0-9]{4,}'
```

### 统计分析

```bash
# 统计各类型日志数量
cat app.log | jq -r '.type' | sort | uniq -c

# 统计 HTTP 状态码分布
cat app.log | grep '"type":"HTTP"' | jq -r '.status' | sort | uniq -c

# 计算 HTTP 平均响应时间
cat app.log | grep '"type":"HTTP"' | jq '.duration_ms' | awk '{sum+=$1; count++} END {print sum/count}'

# 找出最慢的 SQL 查询
cat app.log | grep '"type":"SQL"' | jq -r '[.duration_ms, .query] | @tsv' | sort -rn | head -10
```

### ELK/Kibana 分析

在 Kibana 中创建仪表盘：

1. **HTTP 请求监控**
   - 过滤条件：`type: "HTTP"`
   - 图表：
     - 每秒请求数 (QPS)
     - 平均响应时间
     - 状态码分布
     - Top 10 慢接口

2. **SQL 性能监控**
   - 过滤条件：`type: "SQL"`
   - 图表：
     - 查询总数
     - 慢查询数量（duration_ms > 1000）
     - 平均查询时间
     - Top 10 慢查询

3. **gRPC 调用监控**
   - 过滤条件：`type: "GRPC"`
   - 图表：
     - 调用成功率
     - 错误码分布
     - 平均调用时间
     - 服务调用热力图

4. **缓存效率监控**
   - 过滤条件：`type: "Redis"` 或 `type: "Cache"`
   - 图表：
     - 缓存命中率
     - 平均响应时间
     - 连接错误数

### Grafana 告警

```yaml
# HTTP 错误率告警
- name: http_error_rate
  query: |
    rate(log_entries{type="HTTP", level="ERROR"}[5m])
  threshold: 0.01  # 1% 错误率
  
# SQL 慢查询告警
- name: sql_slow_query
  query: |
    count(log_entries{type="SQL", duration_ms>1000}[5m])
  threshold: 10  # 5分钟内超过10个慢查询

# gRPC 调用失败告警
- name: grpc_failure
  query: |
    rate(log_entries{type="GRPC", level="ERROR"}[5m])
  threshold: 0.05  # 5% 失败率
```

## 最佳实践

### 1. 选择合适的日志级别

```go
// ✅ 正常请求使用 INFO
log.HTTP("GET /api/users", ...)

// ✅ 调试信息使用 DEBUG
log.HTTPDebug("请求头详情", ...)

// ✅ 性能问题使用 WARN
log.HTTPWarn("响应慢", ...)

// ✅ 错误使用 ERROR
log.HTTPError("请求失败", ...)
```

### 2. 添加关键字段

```go
// ✅ HTTP 日志建议字段
log.HTTP("请求处理",
    log.String("method", "POST"),        // 请求方法
    log.String("path", "/api/orders"),   // 请求路径
    log.Int("status", 200),              // 状态码
    log.Float64("duration_ms", 45.6),    // 响应时间
    log.String("user_id", "10086"),      // 用户ID（可选）
    log.String("trace_id", "..."),       // 追踪ID（可选）
)

// ✅ SQL 日志建议字段
log.SQL("查询用户",
    log.String("query", "SELECT..."),    // SQL语句
    log.Int("rows", 100),                // 影响行数
    log.Float64("duration_ms", 12.3),    // 执行时间
)

// ✅ gRPC 日志建议字段
log.GRPC("调用服务",
    log.String("service", "UserService"), // 服务名
    log.String("method", "GetUser"),      // 方法名
    log.String("code", "OK"),             // 状态码
    log.Float64("duration_ms", 23.5),     // 调用时间
)
```

### 3. 性能考虑

```go
// ✅ 只在需要时记录详细信息
if log.V(1).Enabled() {
    log.HTTPDebug("详细请求信息",
        log.String("body", string(bodyBytes)),
        log.String("headers", fmt.Sprint(headers)),
    )
}

// ✅ 避免在循环中频繁记录
for _, item := range items {
    // ❌ 避免这样做
    // log.SQL("处理item", ...)
}
// ✅ 改为批量记录
log.SQL("批量处理", log.Int("count", len(items)))
```

### 4. 与追踪系统集成

```go
// 结合追踪ID
log.HTTP("请求处理",
    log.String("trace_id", traceID),
    log.String("span_id", spanID),
    log.String("method", "POST"),
    // ...
)
```

## 对比传统方式

### 传统方式

```go
// ❌ 需要手动添加 type 字段
log.Info("GET /api/users",
    log.String("type", "HTTP"),  // 容易忘记
    log.String("method", "GET"),
    log.String("path", "/api/users"),
)

// ❌ type 值不统一
log.Info("查询用户", log.String("type", "sql"))    // 小写
log.Info("查询订单", log.String("type", "SQL"))    // 大写
log.Info("查询商品", log.String("type", "database")) // 不一致
```

### 类型化日志

```go
// ✅ 自动添加 type 字段
log.HTTP("GET /api/users",
    log.String("method", "GET"),
    log.String("path", "/api/users"),
)

// ✅ type 值自动统一为 "HTTP"
// ✅ 代码意图更清晰
// ✅ IDE 自动补全更友好
```

## 运行示例

```bash
# 基础示例
go run pkg/log/example/typed-logs/main.go

# 查看输出并过滤
go run pkg/log/example/typed-logs/main.go 2>&1 | grep '"type":"HTTP"'
```

## 总结

类型化日志的优势：

1. ✅ **自动化**：自动添加 type 字段，减少重复代码
2. ✅ **标准化**：统一的类型值，便于分析
3. ✅ **可读性**：函数名直接表达意图
4. ✅ **可分析**：便于日志聚合和性能监控
5. ✅ **易维护**：类型集中管理，修改方便

推荐在生产环境中使用类型化日志，提升系统的可观测性！
