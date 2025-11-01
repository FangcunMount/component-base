# Log Package

一个功能强大的日志包，支持多种日志格式和输出方式。

## 特性

- 支持多种日志格式（JSON、Console）
- 支持多种输出方式（文件、控制台、网络）
- 支持日志级别控制
- 支持结构化日志
- 支持日志轮转
- 支持多种日志库（Zap、Logrus、Klog）
- **支持彩色日志输出**：不同级别的日志显示不同颜色
- **日志级别带方括号**：更清晰的日志级别标识

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

## 日志分级输出

支持将不同级别的日志输出到不同的文件，便于日志分析和问题排查。

### 基本用法

```go
opts := log.NewOptions()
opts.EnableLevelOutput = true
opts.LevelOutputMode = "above" // 或 "exact"

// 为不同级别配置输出路径
opts.LevelOutputPaths = map[string][]string{
    "debug": []string{"stdout"},
    "info":  []string{"stdout", "/var/log/app-info.log"},
    "warn":  []string{"/var/log/app-warn.log"},
    "error": []string{"/var/log/app-error.log"},
}

log.Init(opts)
```

### 输出模式

#### 1. Above 模式（默认）

输出该级别及以上的日志：

```go
opts.LevelOutputMode = "above"
opts.LevelOutputPaths = map[string][]string{
    "info":  []string{"/var/log/info.log"},  // 包含: INFO, WARN, ERROR
    "error": []string{"/var/log/error.log"}, // 包含: ERROR
}
```

**适用场景**：

- 需要在一个文件中查看所有重要日志
- 错误日志单独存储，便于快速定位问题

#### 2. Exact 模式

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

**适用场景**：

- 需要精确统计各级别日志数量
- 不同级别日志需要不同的处理策略

### 示例

查看 `pkg/log/example/leveloutput` 和 `pkg/log/example/exactlevel` 目录中的完整示例。

```bash
# 运行 above 模式示例
go run pkg/log/example/leveloutput/main.go

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

## 最佳实践

1. **选择合适的日志级别**：不要在生产环境使用debug级别
2. **使用结构化日志**：便于日志分析和搜索
3. **配置日志轮转**：避免日志文件过大
4. **分离错误日志**：将错误日志输出到单独的文件
5. **使用有意义的日志消息**：便于问题排查

## 许可证

MIT License
