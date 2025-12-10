package logger

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	gormlogger "gorm.io/gorm/logger"
)

// GORM 日志级别
const (
	GormSilent gormlogger.LogLevel = iota + 1
	GormError
	GormWarn
	GormInfo
)

// GormConfig 定义 GORM 日志配置
type GormConfig struct {
	// SlowThreshold 慢查询阈值，超过此时间会记录警告
	SlowThreshold time.Duration
	// Colorful 是否使用彩色输出（仅开发环境）
	Colorful bool
	// LogLevel 日志级别
	LogLevel gormlogger.LogLevel
}

// DefaultGormConfig 返回默认 GORM 日志配置
func DefaultGormConfig() GormConfig {
	return GormConfig{
		SlowThreshold: 200 * time.Millisecond,
		Colorful:      false,
		LogLevel:      GormInfo,
	}
}

// NewGormLogger 创建 GORM 日志适配器
// 将 GORM 的日志输出适配到 component-base 的类型化日志系统
func NewGormLogger(level int) gormlogger.Interface {
	return NewGormLoggerWithConfig(GormConfig{
		SlowThreshold: 200 * time.Millisecond,
		Colorful:      false,
		LogLevel:      gormlogger.LogLevel(level),
	})
}

// NewGormLoggerWithConfig 使用指定配置创建 GORM 日志适配器
func NewGormLoggerWithConfig(config GormConfig) gormlogger.Interface {
	return &gormLogger{
		GormConfig: config,
	}
}

type gormLogger struct {
	GormConfig
}

// LogMode 设置日志级别
func (l *gormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info 打印 info 日志
func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= GormInfo {
		l.logWithLevel(ctx, GormInfo, msg, data...)
	}
}

// Warn 打印 warn 日志
func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= GormWarn {
		l.logWithLevel(ctx, GormWarn, msg, data...)
	}
}

// Error 打印 error 日志
func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= GormError {
		l.logWithLevel(ctx, GormError, msg, data...)
	}
}

// Trace 打印 sql 日志
func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= 0 {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= GormError:
		sql, rows := fc()
		fields := l.traceFields(ctx, sql, rows, elapsed)
		fields = append(fields, log.String("error", err.Error()))
		log.SQLError("GORM trace failed", fields...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= GormWarn:
		sql, rows := fc()
		fields := l.traceFields(ctx, sql, rows, elapsed)
		fields = append(fields, log.String("event", "slow_query"), log.Duration("slow_threshold", l.SlowThreshold))
		log.SQLWarn("GORM slow query", fields...)
	case l.LogLevel >= GormInfo:
		sql, rows := fc()
		fields := l.traceFields(ctx, sql, rows, elapsed)
		log.SQLDebug("GORM trace", fields...)
	}
}

func (l *gormLogger) logWithLevel(ctx context.Context, level gormlogger.LogLevel, msg string, data ...interface{}) {
	formatted := msg
	if len(data) > 0 {
		formatted = fmt.Sprintf(msg, data...)
	}

	fields := []log.Field{
		log.String("caller", gormFileWithLineNum()),
		log.String("message", formatted),
	}
	fields = append(fields, log.TraceFields(ctx)...)

	switch level {
	case GormError:
		log.SQLError("GORM error", fields...)
	case GormWarn:
		log.SQLWarn("GORM warning", fields...)
	default:
		log.SQL("GORM info", fields...)
	}
}

func (l *gormLogger) traceFields(ctx context.Context, sql string, rows int64, elapsed time.Duration) []log.Field {
	fields := []log.Field{
		log.String("caller", gormFileWithLineNum()),
		log.String("sql", sql),
		log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}

	if rows >= 0 {
		fields = append(fields, log.Int64("rows", rows))
	} else {
		fields = append(fields, log.String("rows", "-1"))
	}

	fields = append(fields, log.TraceFields(ctx)...)
	return fields
}

// gormFileWithLineNum 获取文件名和行号
func gormFileWithLineNum() string {
	for i := 4; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)

		if ok && !strings.HasSuffix(file, "_test.go") {
			dir, f := filepath.Split(file)
			return filepath.Join(filepath.Base(dir), f) + ":" + strconv.FormatInt(int64(line), 10)
		}
	}

	return ""
}
