/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

// Package log is a log package used by TKE team.
package log

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/FangcunMount/component-base/pkg/log/klog"
)

// InfoLogger represents the ability to log non-error messages, at a particular verbosity.
type InfoLogger interface {
	// Info logs a non-error message with the given key/value pairs as context.
	//
	// The msg argument should be used to add some constant description to
	// the log line.  The key/value pairs can then be used to add additional
	// variable information.  The key/value pairs should alternate string
	// keys and arbitrary values.
	Info(msg string, fields ...Field)
	Infof(format string, v ...interface{})
	Infow(msg string, keysAndValues ...interface{})

	// Enabled tests whether this InfoLogger is enabled.  For example,
	// commandline flags might be used to set the logging verbosity and disable
	// some info logs.
	Enabled() bool
}

// Logger represents the ability to log messages, both errors and not.
type Logger interface {
	// All Loggers implement InfoLogger.  Calling InfoLogger methods directly on
	// a Logger value is equivalent to calling them on a V(0) InfoLogger.  For
	// example, logger.Info() produces the same result as logger.V(0).Info.
	InfoLogger
	Debug(msg string, fields ...Field)
	Debugf(format string, v ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Warn(msg string, fields ...Field)
	Warnf(format string, v ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Error(msg string, fields ...Field)
	Errorf(format string, v ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Panic(msg string, fields ...Field)
	Panicf(format string, v ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatal(msg string, fields ...Field)
	Fatalf(format string, v ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	// V returns an InfoLogger value for a specific verbosity level.  A higher
	// verbosity level means a log message is less important.  It's illegal to
	// pass a log level less than zero.
	V(level Level) InfoLogger
	Write(p []byte) (n int, err error)

	// WithValues adds some key-value pairs of context to a logger.
	// See Info for documentation on how key/value pairs work.
	WithValues(keysAndValues ...interface{}) Logger

	// WithName adds a new element to the logger's name.
	// Successive calls with WithName continue to append
	// suffixes to the logger's name.  It's strongly recommended
	// that name segments contain only letters, digits, and hyphens
	// (see the package documentation for more information).
	WithName(name string) Logger

	// WithContext returns a copy of context in which the log value is set.
	WithContext(ctx context.Context) context.Context

	// Flush calls the underlying Core's Sync method, flushing any buffered
	// log entries. Applications should take care to call Sync before exiting.
	Flush()
}

var _ Logger = &zapLogger{}

// noopInfoLogger is a logr.InfoLogger that's always disabled, and does nothing.
type noopInfoLogger struct{}

func (l *noopInfoLogger) Enabled() bool                    { return false }
func (l *noopInfoLogger) Info(_ string, _ ...Field)        {}
func (l *noopInfoLogger) Infof(_ string, _ ...interface{}) {}
func (l *noopInfoLogger) Infow(_ string, _ ...interface{}) {}

var disabledInfoLogger = &noopInfoLogger{}

// NB: right now, we always use the equivalent of sugared logging.
// This is necessary, since logr doesn't define non-suggared types,
// and using zap-specific non-suggared types would make uses tied
// directly to Zap.

// infoLogger is a logr.InfoLogger that uses Zap to log at a particular
// level.  The level has already been converted to a Zap level, which
// is to say that `logrLevel = -1*zapLevel`.
type infoLogger struct {
	level zapcore.Level
	log   *zap.Logger
}

func (l *infoLogger) Enabled() bool { return true }
func (l *infoLogger) Info(msg string, fields ...Field) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(fields...)
	}
}

func (l *infoLogger) Infof(format string, args ...interface{}) {
	if checkedEntry := l.log.Check(l.level, fmt.Sprintf(format, args...)); checkedEntry != nil {
		checkedEntry.Write()
	}
}

func (l *infoLogger) Infow(msg string, keysAndValues ...interface{}) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(handleFields(l.log, keysAndValues)...)
	}
}

// zapLogger is a logr.Logger that uses Zap to log.
type zapLogger struct {
	// NB: this looks very similar to zap.SugaredLogger, but
	// deals with our desire to have multiple verbosity levels.
	zapLogger *zap.Logger
	infoLogger
}

// handleFields converts a bunch of arbitrary key-value pairs into Zap fields.  It takes
// additional pre-converted Zap fields, for use with automatically attached fields, like
// `error`.
func handleFields(l *zap.Logger, args []interface{}, additional ...zap.Field) []zap.Field {
	// a slightly modified version of zap.SugaredLogger.sweetenFields
	if len(args) == 0 {
		// fast-return if we have no suggared fields.
		return additional
	}

	// unlike Zap, we can be pretty sure users aren't passing structured
	// fields (since logr has no concept of that), so guess that we need a
	// little less space.
	fields := make([]zap.Field, 0, len(args)/2+len(additional))
	for i := 0; i < len(args); {
		// check just in case for strongly-typed Zap fields, which is illegal (since
		// it breaks implementation agnosticism), so we can give a better error message.
		if _, ok := args[i].(zap.Field); ok {
			l.DPanic("strongly-typed Zap Field passed to logr", zap.Any("zap field", args[i]))

			break
		}

		// make sure this isn't a mismatched key
		if i == len(args)-1 {
			l.DPanic("odd number of arguments passed as key-value pairs for logging", zap.Any("ignored key", args[i]))

			break
		}

		// process a key-value pair,
		// ensuring that the key is a string
		key, val := args[i], args[i+1]
		keyStr, isString := key.(string)
		if !isString {
			// if the key isn't a string, DPanic and stop logging
			l.DPanic(
				"non-string key argument passed to logging, ignoring all later arguments",
				zap.Any("invalid key", key),
			)

			break
		}

		fields = append(fields, zap.Any(keyStr, val))
		i += 2
	}

	return append(fields, additional...)
}

// newLoggerWithLevelOutput 创建支持分级输出的 logger
func newLoggerWithLevelOutput(opts *Options, zapLevel zapcore.Level, encoderConfig zapcore.EncoderConfig) (*zap.Logger, error) {
	var cores []zapcore.Core

	// 创建 encoder
	var encoder zapcore.Encoder
	if opts.Format == jsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 根据不同的输出模式处理
	if opts.LevelOutputMode == "duplicate" {
		// duplicate 模式：支持多个输出目标
		// 典型场景：app.log 记录所有日志，error.log 额外记录错误日志

		// 首先检查是否有 "all" 配置（记录所有级别）
		if allPaths, ok := opts.LevelOutputPaths["all"]; ok && len(allPaths) > 0 {
			// 为 "all" 创建一个记录所有日志的 Core
			writeSyncers := make([]zapcore.WriteSyncer, 0, len(allPaths))
			for _, path := range allPaths {
				ws := createWriteSyncer(path, opts)
				if ws != nil {
					writeSyncers = append(writeSyncers, ws)
				}
			}

			if len(writeSyncers) > 0 {
				multiWriteSyncer := zapcore.NewMultiWriteSyncer(writeSyncers...)
				// 记录所有级别的日志（从配置的最低级别开始）
				core := zapcore.NewCore(encoder, multiWriteSyncer, zapLevel)
				cores = append(cores, core)
			}
		}

		// 然后为每个特定级别创建额外的输出
		for levelStr, paths := range opts.LevelOutputPaths {
			if levelStr == "all" {
				continue // 已经处理过了
			}

			var level zapcore.Level
			if err := level.UnmarshalText([]byte(levelStr)); err != nil {
				continue
			}

			writeSyncers := make([]zapcore.WriteSyncer, 0, len(paths))
			for _, path := range paths {
				ws := createWriteSyncer(path, opts)
				if ws != nil {
					writeSyncers = append(writeSyncers, ws)
				}
			}

			if len(writeSyncers) > 0 {
				multiWriteSyncer := zapcore.NewMultiWriteSyncer(writeSyncers...)
				// 只记录该特定级别的日志
				levelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
					return l == level
				})
				core := zapcore.NewCore(encoder, multiWriteSyncer, levelEnabler)
				cores = append(cores, core)
			}
		}

	} else {
		// 原有的 exact 和 above 模式
		for levelStr, paths := range opts.LevelOutputPaths {
			var level zapcore.Level
			if err := level.UnmarshalText([]byte(levelStr)); err != nil {
				continue
			}

			writeSyncers := make([]zapcore.WriteSyncer, 0, len(paths))
			for _, path := range paths {
				ws := createWriteSyncer(path, opts)
				if ws != nil {
					writeSyncers = append(writeSyncers, ws)
				}
			}

			if len(writeSyncers) == 0 {
				continue
			}

			multiWriteSyncer := zapcore.NewMultiWriteSyncer(writeSyncers...)

			// 创建 LevelEnabler
			var levelEnabler zapcore.LevelEnabler
			if opts.LevelOutputMode == "exact" {
				// 只输出精确匹配的日志级别
				levelEnabler = zap.LevelEnablerFunc(func(l zapcore.Level) bool {
					return l == level
				})
			} else {
				// 默认 above：输出该级别及以上的日志
				levelEnabler = zap.LevelEnablerFunc(func(l zapcore.Level) bool {
					return l >= level
				})
			}

			core := zapcore.NewCore(encoder, multiWriteSyncer, levelEnabler)
			cores = append(cores, core)
		}
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no valid log output configured")
	}

	// 使用 Tee 组合多个 Core
	core := zapcore.NewTee(cores...)

	// 创建 logger
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.PanicLevel),
		zap.AddCallerSkip(1),
	)

	if opts.Development {
		logger = logger.WithOptions(zap.Development())
	}

	return logger, nil
}

// createWriteSyncer 创建 WriteSyncer
func createWriteSyncer(path string, opts *Options) zapcore.WriteSyncer {
	// 判断是否为特殊路径
	if path == "stdout" || path == "stderr" {
		ws, _, err := zap.Open(path)
		if err != nil {
			return nil
		}
		return ws
	}

	// 文件路径：根据配置选择轮转方式
	if opts.EnableTimeRotation {
		return NewTimeRotationWriter(path, opts.TimeRotationFormat, opts.MaxAge, opts.Compress)
	}

	return getLumberjackWriter(path, opts)
}

// newLoggerWithTimeRotation 创建支持时间轮转的 logger
func newLoggerWithTimeRotation(opts *Options, zapLevel zapcore.Level, encoderConfig zapcore.EncoderConfig) (*zap.Logger, error) {
	// 创建 encoder
	var encoder zapcore.Encoder
	if opts.Format == jsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 处理输出路径
	var writeSyncers []zapcore.WriteSyncer
	for _, path := range opts.OutputPaths {
		if path == "stdout" {
			writeSyncers = append(writeSyncers, zapcore.AddSync(os.Stdout))
		} else if path == "stderr" {
			writeSyncers = append(writeSyncers, zapcore.AddSync(os.Stderr))
		} else {
			// 文件路径，使用时间轮转
			writer := NewTimeRotationWriter(path, opts.TimeRotationFormat, opts.MaxAge, opts.Compress)
			writeSyncers = append(writeSyncers, zapcore.AddSync(writer))
		}
	}

	if len(writeSyncers) == 0 {
		return nil, fmt.Errorf("no valid output paths configured")
	}

	// 创建 Core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writeSyncers...),
		zapLevel,
	)

	// 创建 logger
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.PanicLevel),
		zap.AddCallerSkip(1),
	)

	if opts.Development {
		logger = logger.WithOptions(zap.Development())
	}

	if !opts.DisableCaller {
		logger = logger.WithOptions(zap.AddCaller())
	}

	if !opts.DisableStacktrace {
		logger = logger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
	}

	return logger, nil
}

var (
	std = New(NewOptions())
	mu  sync.Mutex
)

// Init initializes logger with specified options.
func Init(opts *Options) {
	mu.Lock()
	defer mu.Unlock()
	std = New(opts)
}

// New create logger by opts which can custmoized by command arguments.
func New(opts *Options) *zapLogger {
	if opts == nil {
		opts = NewOptions()
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	// 使用自定义的级别编码器，添加方括号和颜色
	encodeLevel := customLevelEncoderNoColor
	// when output to local path, with color is forbidden
	if opts.Format == consoleFormat && opts.EnableColor {
		encodeLevel = customLevelEncoder
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     timeEncoder,
		EncodeDuration: milliSecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var l *zap.Logger
	var err error

	// 如果启用了分级输出，使用自定义 Core
	if opts.EnableLevelOutput && len(opts.LevelOutputPaths) > 0 {
		l, err = newLoggerWithLevelOutput(opts, zapLevel, encoderConfig)
	} else if opts.EnableTimeRotation {
		// 如果启用了时间轮转，使用自定义 Core
		l, err = newLoggerWithTimeRotation(opts, zapLevel, encoderConfig)
	} else {
		// 使用默认配置
		loggerConfig := &zap.Config{
			Level:             zap.NewAtomicLevelAt(zapLevel),
			Development:       opts.Development,
			DisableCaller:     opts.DisableCaller,
			DisableStacktrace: opts.DisableStacktrace,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         opts.Format,
			EncoderConfig:    encoderConfig,
			OutputPaths:      opts.OutputPaths,
			ErrorOutputPaths: opts.ErrorOutputPaths,
		}
		l, err = loggerConfig.Build(zap.AddStacktrace(zapcore.PanicLevel), zap.AddCallerSkip(1))
	}

	if err != nil {
		panic(err)
	}

	logger := &zapLogger{
		zapLogger: l.Named(opts.Name),
		infoLogger: infoLogger{
			log:   l,
			level: zap.InfoLevel,
		},
	}
	klog.InitLogger(l)
	zap.RedirectStdLog(l)

	return logger
}

// SugaredLogger returns global sugared logger.
func SugaredLogger() *zap.SugaredLogger {
	return std.zapLogger.Sugar()
}

// StdErrLogger returns logger of standard library which writes to supplied zap
// logger at error level.
func StdErrLogger() *log.Logger {
	if std == nil {
		return nil
	}
	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.ErrorLevel); err == nil {
		return l
	}

	return nil
}

// StdInfoLogger returns logger of standard library which writes to supplied zap
// logger at info level.
func StdInfoLogger() *log.Logger {
	if std == nil {
		return nil
	}
	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.InfoLevel); err == nil {
		return l
	}

	return nil
}

// V return a leveled InfoLogger.
func V(level Level) InfoLogger { return std.V(level) }

func (l *zapLogger) V(level Level) InfoLogger {
	if l.zapLogger.Core().Enabled(level) {
		return &infoLogger{
			level: level,
			log:   l.zapLogger,
		}
	}

	return disabledInfoLogger
}

func (l *zapLogger) Write(p []byte) (n int, err error) {
	l.zapLogger.Info(string(p))

	return len(p), nil
}

// WithValues creates a child logger and adds adds Zap fields to it.
func WithValues(keysAndValues ...interface{}) Logger { return std.WithValues(keysAndValues...) }

func (l *zapLogger) WithValues(keysAndValues ...interface{}) Logger {
	newLogger := l.zapLogger.With(handleFields(l.zapLogger, keysAndValues)...)

	return NewLogger(newLogger)
}

// WithName adds a new path segment to the logger's name. Segments are joined by
// periods. By default, Loggers are unnamed.
func WithName(s string) Logger { return std.WithName(s) }

func (l *zapLogger) WithName(name string) Logger {
	newLogger := l.zapLogger.Named(name)

	return NewLogger(newLogger)
}

// Flush calls the underlying Core's Sync method, flushing any buffered
// log entries. Applications should take care to call Sync before exiting.
func Flush() { std.Flush() }

func (l *zapLogger) Flush() {
	_ = l.zapLogger.Sync()
}

// NewLogger creates a new logr.Logger using the given Zap Logger to log.
func NewLogger(l *zap.Logger) Logger {
	return &zapLogger{
		zapLogger: l,
		infoLogger: infoLogger{
			log:   l,
			level: zap.InfoLevel,
		},
	}
}

// ZapLogger used for other log wrapper such as klog.
func ZapLogger() *zap.Logger {
	return std.zapLogger
}

// CheckIntLevel used for other log wrapper such as klog which return if logging a
// message at the specified level is enabled.
func CheckIntLevel(level int32) bool {
	var lvl zapcore.Level
	if level < 5 {
		lvl = zapcore.InfoLevel
	} else {
		lvl = zapcore.DebugLevel
	}
	checkEntry := std.zapLogger.Check(lvl, "")

	return checkEntry != nil
}

// Debug method output debug level log.
func Debug(msg string, fields ...Field) {
	std.zapLogger.Debug(msg, fields...)
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.zapLogger.Debug(msg, fields...)
}

// Debugf method output debug level log.
func Debugf(format string, v ...interface{}) {
	std.zapLogger.Sugar().Debugf(format, v...)
}

func (l *zapLogger) Debugf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Debugf(format, v...)
}

// Debugw method output debug level log.
func Debugw(msg string, keysAndValues ...interface{}) {
	std.zapLogger.Sugar().Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.zapLogger.Sugar().Debugw(msg, keysAndValues...)
}

// Info method output info level log.
func Info(msg string, fields ...Field) {
	std.zapLogger.Info(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.zapLogger.Info(msg, fields...)
}

// Infof method output info level log.
func Infof(format string, v ...interface{}) {
	std.zapLogger.Sugar().Infof(format, v...)
}

func (l *zapLogger) Infof(format string, v ...interface{}) {
	l.zapLogger.Sugar().Infof(format, v...)
}

// Infow method output info level log.
func Infow(msg string, keysAndValues ...interface{}) {
	std.zapLogger.Sugar().Infow(msg, keysAndValues...)
}

func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.zapLogger.Sugar().Infow(msg, keysAndValues...)
}

// Warn method output warning level log.
func Warn(msg string, fields ...Field) {
	std.zapLogger.Warn(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.zapLogger.Warn(msg, fields...)
}

// Warnf method output warning level log.
func Warnf(format string, v ...interface{}) {
	std.zapLogger.Sugar().Warnf(format, v...)
}

func (l *zapLogger) Warnf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Warnf(format, v...)
}

// Warnw method output warning level log.
func Warnw(msg string, keysAndValues ...interface{}) {
	std.zapLogger.Sugar().Warnw(msg, keysAndValues...)
}

func (l *zapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.zapLogger.Sugar().Warnw(msg, keysAndValues...)
}

// Error method output error level log.
func Error(msg string, fields ...Field) {
	std.zapLogger.Error(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.zapLogger.Error(msg, fields...)
}

// Errorf method output error level log.
func Errorf(format string, v ...interface{}) {
	std.zapLogger.Sugar().Errorf(format, v...)
}

func (l *zapLogger) Errorf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Errorf(format, v...)
}

// Errorw method output error level log.
func Errorw(msg string, keysAndValues ...interface{}) {
	std.zapLogger.Sugar().Errorw(msg, keysAndValues...)
}

func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.zapLogger.Sugar().Errorw(msg, keysAndValues...)
}

// Panic method output panic level log and shutdown application.
func Panic(msg string, fields ...Field) {
	std.zapLogger.Panic(msg, fields...)
}

func (l *zapLogger) Panic(msg string, fields ...Field) {
	l.zapLogger.Panic(msg, fields...)
}

// Panicf method output panic level log and shutdown application.
func Panicf(format string, v ...interface{}) {
	std.zapLogger.Sugar().Panicf(format, v...)
}

func (l *zapLogger) Panicf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Panicf(format, v...)
}

// Panicw method output panic level log.
func Panicw(msg string, keysAndValues ...interface{}) {
	std.zapLogger.Sugar().Panicw(msg, keysAndValues...)
}

func (l *zapLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.zapLogger.Sugar().Panicw(msg, keysAndValues...)
}

// Fatal method output fatal level log.
func Fatal(msg string, fields ...Field) {
	std.zapLogger.Fatal(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.zapLogger.Fatal(msg, fields...)
}

// Fatalf method output fatal level log.
func Fatalf(format string, v ...interface{}) {
	std.zapLogger.Sugar().Fatalf(format, v...)
}

func (l *zapLogger) Fatalf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Fatalf(format, v...)
}

// Fatalw method output Fatalw level log.
func Fatalw(msg string, keysAndValues ...interface{}) {
	std.zapLogger.Sugar().Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.zapLogger.Sugar().Fatalw(msg, keysAndValues...)
}

// L method output with specified context value.
func L(ctx context.Context) *zapLogger {
	return std.L(ctx)
}

func (l *zapLogger) L(ctx context.Context) *zapLogger {
	lg := l.clone()

	if requestID := ctx.Value(KeyRequestID); requestID != nil {
		lg.zapLogger = lg.zapLogger.With(zap.Any(KeyRequestID, requestID))
	}
	if username := ctx.Value(KeyUsername); username != nil {
		lg.zapLogger = lg.zapLogger.With(zap.Any(KeyUsername, username))
	}
	if watcherName := ctx.Value(KeyWatcherName); watcherName != nil {
		lg.zapLogger = lg.zapLogger.With(zap.Any(KeyWatcherName, watcherName))
	}

	return lg
}

//nolint:predeclared
func (l *zapLogger) clone() *zapLogger {
	copy := *l

	return &copy
}

// 类型化日志辅助函数
// 这些函数为不同类型的操作提供了便捷的日志记录方式，并自动添加 type 字段

// HTTP 记录 HTTP 请求日志
// 自动添加 type=HTTP 字段
func HTTP(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "HTTP")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// HTTPDebug 记录 HTTP 请求的调试日志
func HTTPDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "HTTP")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// HTTPWarn 记录 HTTP 请求的警告日志
func HTTPWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "HTTP")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// HTTPError 记录 HTTP 请求的错误日志
func HTTPError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "HTTP")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// SQL 记录 SQL 查询日志
// 自动添加 type=SQL 字段
func SQL(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "SQL")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// SQLDebug 记录 SQL 查询的调试日志
func SQLDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "SQL")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// SQLWarn 记录 SQL 查询的警告日志
func SQLWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "SQL")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// SQLError 记录 SQL 查询的错误日志
func SQLError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "SQL")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// GRPC 记录 gRPC 调用日志
// 自动添加 type=GRPC 字段
func GRPC(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "GRPC")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// GRPCDebug 记录 gRPC 调用的调试日志
func GRPCDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "GRPC")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// GRPCWarn 记录 gRPC 调用的警告日志
func GRPCWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "GRPC")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// GRPCError 记录 gRPC 调用的错误日志
func GRPCError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "GRPC")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// Redis 记录 Redis 操作日志
// 自动添加 type=Redis 字段
func Redis(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Redis")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// RedisDebug 记录 Redis 操作的调试日志
func RedisDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Redis")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// RedisWarn 记录 Redis 操作的警告日志
func RedisWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Redis")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// RedisError 记录 Redis 操作的错误日志
func RedisError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Redis")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// MQ 记录消息队列操作日志
// 自动添加 type=MQ 字段
func MQ(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MQ")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// MQDebug 记录消息队列操作的调试日志
func MQDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MQ")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// MQWarn 记录消息队列操作的警告日志
func MQWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MQ")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// MQError 记录消息队列操作的错误日志
func MQError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MQ")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// Cache 记录缓存操作日志
// 自动添加 type=Cache 字段
func Cache(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Cache")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// CacheDebug 记录缓存操作的调试日志
func CacheDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Cache")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// CacheWarn 记录缓存操作的警告日志
func CacheWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Cache")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// CacheError 记录缓存操作的错误日志
func CacheError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "Cache")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// RPC 记录 RPC 调用日志（通用）
// 自动添加 type=RPC 字段
func RPC(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "RPC")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// RPCDebug 记录 RPC 调用的调试日志
func RPCDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "RPC")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// RPCWarn 记录 RPC 调用的警告日志
func RPCWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "RPC")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// RPCError 记录 RPC 调用的错误日志
func RPCError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "RPC")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}

// Mongo 记录 MongoDB 操作日志
// 自动添加 type=MongoDB 字段
func Mongo(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MongoDB")}, fields...)
	std.zapLogger.Info(msg, allFields...)
}

// MongoDebug 记录 MongoDB 操作的调试日志
func MongoDebug(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MongoDB")}, fields...)
	std.zapLogger.Debug(msg, allFields...)
}

// MongoWarn 记录 MongoDB 操作的警告日志
func MongoWarn(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MongoDB")}, fields...)
	std.zapLogger.Warn(msg, allFields...)
}

// MongoError 记录 MongoDB 操作的错误日志
func MongoError(msg string, fields ...Field) {
	allFields := append([]Field{String("type", "MongoDB")}, fields...)
	std.zapLogger.Error(msg, allFields...)
}
