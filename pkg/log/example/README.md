# 彩色日志示例

本示例展示如何使用带颜色和方括号的日志输出功能。

## 示例说明

### 1. 带颜色的日志输出

位置：`pkg/log/example/colorful/main.go`

这个示例展示了如何启用彩色日志输出，不同级别的日志将以不同的颜色显示：

- `[DEBUG]` - 青色（适合调试信息）
- `[INFO]` - 绿色（适合一般信息）
- `[WARN]` - 黄色（适合警告信息）
- `[ERROR]` - 红色（适合错误信息）

运行命令：

```bash
go run pkg/log/example/colorful/main.go
```

输出效果（带颜色）：

```text
2025-11-01 10:29:18.520 [DEBUG] colorful/main.go:20     This is a DEBUG message {"status": "testing"}
2025-11-01 10:29:18.520 [INFO]  colorful/main.go:21     This is an INFO message {"status": "running"}
2025-11-01 10:29:18.520 [WARN]  colorful/main.go:22     This is a WARN message  {"status": "warning"}
2025-11-01 10:29:18.520 [ERROR] colorful/main.go:23     This is an ERROR message {"status": "error"}
```

### 2. 不带颜色的日志输出

位置：`pkg/log/example/nocolor/main.go`

这个示例展示了禁用颜色时的日志输出，所有日志级别仍然带有方括号，但没有颜色标识。

运行命令：

```bash
go run pkg/log/example/nocolor/main.go
```

输出效果（无颜色）：

```text
2025-11-01 10:29:27.581 [DEBUG] nocolor/main.go:20      This is a DEBUG message {"status": "testing"}
2025-11-01 10:29:27.582 [INFO]  nocolor/main.go:21      This is an INFO message {"status": "running"}
2025-11-01 10:29:27.582 [WARN]  nocolor/main.go:22      This is a WARN message  {"status": "warning"}
2025-11-01 10:29:27.582 [ERROR] nocolor/main.go:23      This is an ERROR message {"status": "error"}
```

## 配置说明

### 启用彩色输出

```go
opts := log.NewOptions()
opts.EnableColor = true  // 启用颜色
opts.Format = "console"  // 必须是 console 格式
log.Init(opts)
```

### 禁用彩色输出

```go
opts := log.NewOptions()
opts.EnableColor = false // 禁用颜色
opts.Format = "console"  // console 格式
log.Init(opts)
```

## 使用建议

1. **开发环境**：建议启用彩色输出，便于快速识别不同级别的日志
2. **生产环境（控制台）**：可以启用彩色输出，提升日志可读性
3. **文件输出**：建议禁用彩色输出，避免 ANSI 转义字符污染日志文件
4. **JSON 格式**：彩色配置不影响 JSON 格式输出

## 高级用法

### 混合输出配置

同时输出到控制台（带颜色）和文件（不带颜色）：

```go
opts := log.NewOptions()
opts.EnableColor = true           // 控制台启用颜色
opts.Format = "console"
opts.OutputPaths = []string{
    "stdout",                     // 控制台输出（带颜色）
    "/var/log/app.log",          // 文件输出（建议另外配置）
}
log.Init(opts)
```

**注意**：如果同时输出到文件和控制台，颜色代码也会写入文件。如需避免，建议分开配置或使用不带颜色的选项。

### 日志级别过滤

```go
opts := log.NewOptions()
opts.Level = "warn"      // 只显示 WARN、ERROR、FATAL 级别的日志
opts.EnableColor = true
log.Init(opts)
```

## 技术实现

日志包使用自定义的 `zapcore.LevelEncoder` 来实现方括号和颜色功能：

- `customLevelEncoder`: 带颜色的级别编码器
- `customLevelEncoderNoColor`: 不带颜色的级别编码器

颜色使用 ANSI 转义序列实现，兼容大多数现代终端。
