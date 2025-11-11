# 日志分级输出功能详解

## 功能概述

日志分级输出允许你将不同级别的日志输出到不同的文件或目标，这对于日志管理和问题排查非常有用。

## 最新更新：Duplicate 模式（推荐）

**问题**：之前的模式存在日志重复或难以追溯的问题。

**解决**：新增 **Duplicate 模式**，实现"完整日志 + 关键日志分离"：
- `app.log` 记录所有日志（完整上下文）
- `error.log` 只额外记录 ERROR（快速定位）
- 最小化重复，最大化实用性

### 快速开始

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "duplicate" // 推荐使用

opts.LevelOutputPaths = map[string][]string{
    "all":   []string{"/var/log/app.log"},   // 记录所有日志
    "error": []string{"/var/log/error.log"}, // 额外记录错误
}

log.Init(opts)
```

### 输出效果示例

输入日志：
```
[DEBUG] 调试信息
[INFO]  用户登录
[WARN]  内存使用率高
[ERROR] 数据库连接失败
```

**app.log（完整）**：
```
[DEBUG] 调试信息
[INFO]  用户登录
[WARN]  内存使用率高
[ERROR] 数据库连接失败
```

**error.log（只有错误）**：
```
[ERROR] 数据库连接失败
```

✅ **优势**：
- app.log 提供完整上下文，便于问题追溯
- error.log 只含错误，便于监控告警
- ERROR 只额外记录一次，避免重复

## 配置项

### EnableLevelOutput

是否启用分级输出功能。

- `true`: 使用 `LevelOutputPaths` 配置
- `false`: 使用传统的 `OutputPaths` 配置（默认）

### LevelOutputPaths

为不同日志级别配置输出路径的 map。

**Duplicate 模式特殊键**：
- `"all"`: 记录所有级别的日志（推荐）
- 其他级别（如 `"error"`, `"warn"`）：只记录该特定级别

```go
LevelOutputPaths: map[string][]string{
    "debug": []string{"stdout"},
    "info":  []string{"stdout", "/var/log/info.log"},
    "warn":  []string{"/var/log/warn.log"},
    "error": []string{"/var/log/error.log"},
}
```

支持的级别：
- `debug`
- `info`
- `warn`
- `error`
- `dpanic`
- `panic`
- `fatal`

### LevelOutputMode

控制每个级别输出的日志范围。

#### Above 模式（默认）

输出该级别及以上的所有日志。

```go
LevelOutputMode: "above"
```

**行为**：
```
debug 配置 -> 输出: DEBUG, INFO, WARN, ERROR, FATAL, PANIC
info 配置  -> 输出: INFO, WARN, ERROR, FATAL, PANIC
warn 配置  -> 输出: WARN, ERROR, FATAL, PANIC
error 配置 -> 输出: ERROR, FATAL, PANIC
```

**优点**：
- 一个文件包含多个级别，减少文件数量
- 适合需要按重要性查看日志的场景
- 错误日志文件只包含错误及以上级别

**缺点**：
- 可能导致日志重复（如果多个级别都配置了输出）
- 文件大小可能不均衡

#### Exact 模式

只输出精确匹配的日志级别。

```go
LevelOutputMode: "exact"
```

**行为**：
```
debug 配置 -> 仅输出: DEBUG
info 配置  -> 仅输出: INFO
warn 配置  -> 仅输出: WARN
error 配置 -> 仅输出: ERROR
```

**优点**：
- 日志完全分离，不会重复
- 便于统计各级别的日志量
- 可以为不同级别设置不同的轮转策略

**缺点**：
- 需要查看多个文件才能获得完整日志
- 文件数量较多

## 使用场景

### 场景 1：错误日志单独存储

**需求**：在控制台看所有日志，但错误日志单独保存到文件。

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"stdout"},
    "error": []string{"/var/log/error.log"},
}
log.Init(opts)
```

### 场景 2：完全分级存储

**需求**：每个级别的日志都单独存储，便于分析。

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "exact"
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"/var/log/debug.log"},
    "info":  []string{"/var/log/info.log"},
    "warn":  []string{"/var/log/warn.log"},
    "error": []string{"/var/log/error.log"},
}
log.Init(opts)
```

### 场景 3：分层存储

**需求**：INFO 及以上输出到文件，ERROR 单独存储。

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"stdout"},
    "info":  []string{"/var/log/app.log"},
    "error": []string{"/var/log/error.log"},
}
log.Init(opts)
```

结果：
- `app.log`: INFO, WARN, ERROR
- `error.log`: ERROR
- 控制台: 所有日志

### 场景 4：生产环境配置

**需求**：最小化磁盘使用，只记录重要日志。

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.Level = "info" // 不记录 DEBUG
opts.LevelOutputPaths = map[string][]string{
    "info":  []string{"/var/log/app.log"},
    "error": []string{"/var/log/error.log"},
}
log.Init(opts)
```

## 性能考虑

### 多目标输出的性能影响

每个 Core 都会独立处理日志，多个目标会增加一定开销。

**建议**：
1. 避免过多的输出目标
2. 使用 `above` 模式减少 Core 数量
3. 在生产环境关闭不必要的 DEBUG 日志

### 文件 I/O 优化

```go
// 使用缓冲写入
opts.OutputPaths = []string{
    "stdout",
    "/var/log/app.log",
}

// 配置轮转以避免单个文件过大
opts.MaxSize = 100    // 100MB
opts.MaxBackups = 10  // 保留10个备份
opts.Compress = true  // 压缩旧文件
```

## 常见问题

### Q1: 日志重复输出怎么办？

**A**: 这通常发生在 `above` 模式下，多个级别都配置了相同的输出。

解决方案：
1. 使用 `exact` 模式
2. 或者只配置必要的级别

### Q2: 如何在运行时切换日志级别？

**A**: 当前不支持运行时动态切换。需要重新初始化日志。

### Q3: 可以输出到网络吗？

**A**: 是的，使用网络路径：

```go
opts.LevelOutputPaths = map[string][]string{
    "error": []string{"tcp://logserver:9000"},
}
```

### Q4: 控制台输出会有颜色吗？

**A**: 是的，如果启用了 `EnableColor`，输出到 `stdout` 的日志会有颜色。

```go
opts.EnableColor = true
opts.LevelOutputPaths = map[string][]string{
    "info": []string{"stdout"},
}
```

## 最佳实践

### 1. 开发环境

```go
opts := log.NewOptions()
opts.EnableColor = true
opts.Level = "debug"
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"stdout"},
}
```

### 2. 测试环境

```go
opts := log.NewOptions()
opts.Level = "info"
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "info":  []string{"stdout", "/var/log/test.log"},
    "error": []string{"/var/log/test-error.log"},
}
```

### 3. 生产环境

```go
opts := log.NewOptions()
opts.Level = "info"
opts.Format = "json" // JSON 格式便于日志收集
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "info":  []string{"/var/log/app/app.log"},
    "error": []string{"/var/log/app/error.log"},
}
opts.MaxSize = 100
opts.MaxBackups = 30
opts.Compress = true
```

### 4. 容器环境

```go
opts := log.NewOptions()
opts.Level = "info"
opts.Format = "json"
opts.EnableLevelOutput = false // 简化配置
opts.OutputPaths = []string{"stdout"} // 让容器运行时收集
```

## 迁移指南

### 从传统配置迁移

**之前**：
```go
opts.OutputPaths = []string{"stdout", "/var/log/app.log"}
opts.ErrorOutputPaths = []string{"/var/log/error.log"}
```

**之后**：
```go
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"stdout", "/var/log/app.log"},
    "error": []string{"/var/log/error.log"},
}
```

## 示例代码

完整示例请查看：
- `pkg/log/example/leveloutput/main.go` - Above 模式示例
- `pkg/log/example/exactlevel/main.go` - Exact 模式示例
