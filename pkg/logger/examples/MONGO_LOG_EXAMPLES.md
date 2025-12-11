# MongoDB 日志输出示例

## 概述

MongoDB 日志驱动现在默认记录查询语句（类似 GORM 记录 SQL），同时保持对敏感信息的脱敏保护。

## 配置说明

### 默认配置（推荐）

```go
// 使用默认配置 - 记录查询语句，自动脱敏
hook := logger.NewMongoHookWithConfig(logger.DefaultMongoHookConfig())

// 或者使用简化方式
hook := logger.NewMongoHook(true, 200*time.Millisecond)
```

**默认行为**：
- ✅ 记录查询语句（`query` 字段）
- ✅ 自动脱敏敏感信息
- ✅ 记录数据库和集合名称
- ❌ 不记录响应详情（避免大量数据）
- ❌ 不记录心跳和握手命令

## 日志输出示例

### 1. 普通查询（Debug 级别）

```json
{
  "level": "debug",
  "ts": "2025-12-11T14:30:25.123+08:00",
  "type": "MongoDB",
  "msg": "MongoDB command succeeded",
  "request_id": 12345,
  "command": "find",
  "connection_id": "localhost:27017[1]",
  "elapsed_ms": 15.234,
  "database": "mydb",
  "collection": "users",
  "query": "{find: \"users\", filter: {age: {$gt: 18}}, limit: 10}",
  "trace_id": "abc123",
  "span_id": "def456"
}
```

### 2. 慢查询（Warn 级别）

```json
{
  "level": "warn",
  "ts": "2025-12-11T14:30:25.456+08:00",
  "type": "MongoDB",
  "msg": "MongoDB slow command",
  "request_id": 12346,
  "command": "aggregate",
  "connection_id": "localhost:27017[1]",
  "elapsed_ms": 350.678,
  "database": "mydb",
  "collection": "orders",
  "query": "{aggregate: \"orders\", pipeline: [{$match: {status: \"pending\"}}, {$group: {_id: \"$userId\", total: {$sum: \"$amount\"}}}]}",
  "event": "slow_command",
  "slow_threshold": "200ms",
  "trace_id": "abc124",
  "span_id": "def457"
}
```

### 3. 插入操作

```json
{
  "level": "debug",
  "ts": "2025-12-11T14:30:25.789+08:00",
  "type": "MongoDB",
  "msg": "MongoDB command succeeded",
  "request_id": 12347,
  "command": "insert",
  "connection_id": "localhost:27017[1]",
  "elapsed_ms": 8.123,
  "database": "mydb",
  "collection": "users",
  "query": "{collection: users, count: 1}",
  "trace_id": "abc125",
  "span_id": "def458"
}
```

**注意**：插入/更新/删除操作只记录元数据（集合名和操作数量），不记录具体数据，避免敏感信息泄露。

### 4. 更新操作

```json
{
  "level": "debug",
  "ts": "2025-12-11T14:30:26.012+08:00",
  "type": "MongoDB",
  "msg": "MongoDB command succeeded",
  "request_id": 12348,
  "command": "update",
  "connection_id": "localhost:27017[1]",
  "elapsed_ms": 12.345,
  "database": "mydb",
  "collection": "users",
  "query": "{collection: users, count: 5}",
  "trace_id": "abc126",
  "span_id": "def459"
}
```

### 5. 认证命令（脱敏）

```json
{
  "level": "debug",
  "ts": "2025-12-11T14:30:26.234+08:00",
  "type": "MongoDB",
  "msg": "MongoDB command succeeded",
  "request_id": 12349,
  "command": "authenticate",
  "connection_id": "localhost:27017[1]",
  "elapsed_ms": 5.678,
  "query": "[REDACTED: authentication command]",
  "trace_id": "abc127",
  "span_id": "def460"
}
```

### 6. 查询失败（Error 级别）

```json
{
  "level": "error",
  "ts": "2025-12-11T14:30:26.567+08:00",
  "type": "MongoDB",
  "msg": "MongoDB command failed",
  "request_id": 12350,
  "command": "find",
  "connection_id": "localhost:27017[1]",
  "elapsed_ms": 10.123,
  "database": "mydb",
  "collection": "users",
  "query": "{find: \"users\", filter: {invalid_field: 1}}",
  "error": "unknown top level operator: $invalid",
  "trace_id": "abc128",
  "span_id": "def461"
}
```

## 与 GORM SQL 日志的对比

### GORM SQL 日志

```json
{
  "level": "debug",
  "type": "SQL",
  "msg": "GORM trace",
  "caller": "gorm.io/gorm@v1.25.0/callbacks.go:42",
  "sql": "SELECT * FROM `users` WHERE age > ? LIMIT ?",
  "elapsed_ms": 12.345,
  "rows": 10
}
```

### MongoDB 日志

```json
{
  "level": "debug",
  "type": "MongoDB",
  "msg": "MongoDB command succeeded",
  "request_id": 12345,
  "command": "find",
  "database": "mydb",
  "collection": "users",
  "query": "{find: \"users\", filter: {age: {$gt: 18}}, limit: 10}",
  "elapsed_ms": 15.234
}
```

## 日志字段说明

| 字段 | 说明 | 示例 |
|------|------|------|
| `level` | 日志级别 | debug/warn/error |
| `type` | 日志类型 | MongoDB |
| `msg` | 日志消息 | MongoDB command succeeded |
| `request_id` | 请求ID | 12345 |
| `command` | MongoDB 命令名 | find/insert/update |
| `database` | 数据库名 | mydb |
| `collection` | 集合名 | users |
| `query` | 查询语句（脱敏后） | {find: "users", ...} |
| `elapsed_ms` | 执行时间（毫秒） | 15.234 |
| `connection_id` | 连接ID | localhost:27017[1] |
| `trace_id` | 分布式追踪ID | abc123 |
| `span_id` | Span ID | def456 |
| `event` | 事件类型（慢查询时） | slow_command |
| `slow_threshold` | 慢查询阈值 | 200ms |
| `error` | 错误信息（失败时） | ... |

## 配置选项

### 完全禁用查询语句记录

```go
config := logger.MongoHookConfig{
    Enabled:          true,
    SlowThreshold:    200 * time.Millisecond,
    LogCommandDetail: false,  // 禁用查询语句记录
}
hook := logger.NewMongoHookWithConfig(config)
```

### 启用响应详情记录（调试用）

```go
config := logger.MongoHookConfig{
    Enabled:          true,
    SlowThreshold:    200 * time.Millisecond,
    LogCommandDetail: true,
    LogReplyDetail:   true,   // 记录响应详情
}
hook := logger.NewMongoHookWithConfig(config)
```

### 只记录慢查询

```go
config := logger.MongoHookConfig{
    Enabled:          true,
    SlowThreshold:    100 * time.Millisecond,  // 更严格的阈值
    LogCommandDetail: true,
}
hook := logger.NewMongoHookWithConfig(config)
```

## 最佳实践

1. **生产环境**：使用默认配置，记录查询语句但不记录响应
2. **开发环境**：可以启用响应详情记录，方便调试
3. **性能分析**：降低慢查询阈值，关注性能瓶颈
4. **安全审计**：查看日志中的查询模式，发现异常行为

## 性能影响

- **查询语句记录**：轻微影响（~1-2% CPU）
- **响应详情记录**：中等影响（~5-10% CPU，取决于响应大小）
- **缓存机制**：使用 `sync.Map` 缓存命令信息，低开销

## 常见问题

### Q: 为什么查询语句显示为元数据而不是完整的文档？

A: 对于写操作（insert/update/delete），为了保护敏感数据，只记录集合名和操作数量，不记录具体内容。

### Q: 如何查看完整的响应？

A: 设置 `LogReplyDetail: true`，但建议仅在调试时使用。

### Q: 认证命令会记录密码吗？

A: 不会。所有认证相关命令都会被脱敏，显示为 `[REDACTED: authentication command]`。

### Q: 查询语句太长怎么办？

A: 查询语句会自动截断（超过 500 字符），显示为 `... [truncated]`。
