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

package log

import (
	"fmt"
	"strings"

	"github.com/FangcunMount/component-base/pkg/json"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	flagLevel             = "log.level"
	flagDisableCaller     = "log.disable-caller"
	flagDisableStacktrace = "log.disable-stacktrace"
	flagFormat            = "log.format"
	flagEnableColor       = "log.enable-color"
	flagOutputPaths       = "log.output-paths"
	flagErrorOutputPaths  = "log.error-output-paths"
	flagDevelopment       = "log.development"
	flagName              = "log.name"

	consoleFormat = "console"
	jsonFormat    = "json"
)

// Options contains configuration items related to log.
type Options struct {
	OutputPaths       []string `json:"output-paths"       mapstructure:"output-paths"`
	ErrorOutputPaths  []string `json:"error-output-paths" mapstructure:"error-output-paths"`
	Level             string   `json:"level"              mapstructure:"level"`
	Format            string   `json:"format"             mapstructure:"format"`
	DisableCaller     bool     `json:"disable-caller"     mapstructure:"disable-caller"`
	DisableStacktrace bool     `json:"disable-stacktrace" mapstructure:"disable-stacktrace"`
	EnableColor       bool     `json:"enable-color"       mapstructure:"enable-color"`
	Development       bool     `json:"development"        mapstructure:"development"`
	Name              string   `json:"name"               mapstructure:"name"`

	// 日志轮转配置
	MaxSize    int  `json:"max-size"    mapstructure:"max-size"`    // 单个日志文件最大大小（MB）
	MaxAge     int  `json:"max-age"     mapstructure:"max-age"`     // 保留旧日志文件的最大天数
	MaxBackups int  `json:"max-backups" mapstructure:"max-backups"` // 保留旧日志文件的最大个数
	Compress   bool `json:"compress"    mapstructure:"compress"`    // 是否压缩旧日志文件

	// 按时间轮转配置
	EnableTimeRotation bool   `json:"enable-time-rotation" mapstructure:"enable-time-rotation"` // 是否启用按时间轮转
	TimeRotationFormat string `json:"time-rotation-format" mapstructure:"time-rotation-format"` // 时间轮转格式，如 "2006-01-02" 表示按天

	// 日志分级输出配置
	// 为不同日志级别配置独立的输出路径
	// 例如: {"info": []string{"stdout", "/var/log/info.log"}, "error": []string{"/var/log/error.log"}}
	LevelOutputPaths map[string][]string `json:"level-output-paths" mapstructure:"level-output-paths"`
	// 是否启用分级输出（如果为 true，则使用 LevelOutputPaths；否则使用 OutputPaths）
	EnableLevelOutput bool `json:"enable-level-output" mapstructure:"enable-level-output"`
	// 分级输出模式：
	// "exact" - 只输出精确匹配的日志级别
	// "above" - 输出该级别及以上的日志（默认）
	LevelOutputMode string `json:"level-output-mode" mapstructure:"level-output-mode"`
}

// NewOptions creates an Options object with default parameters.
func NewOptions() *Options {
	return &Options{
		Level:              zapcore.InfoLevel.String(),
		DisableCaller:      false,
		DisableStacktrace:  false,
		Format:             consoleFormat,
		EnableColor:        false,
		Development:        false,
		OutputPaths:        []string{"stdout"},
		ErrorOutputPaths:   []string{"stderr"},
		MaxSize:            100,          // 100MB
		MaxAge:             30,           // 30天
		MaxBackups:         10,           // 保留10个备份文件
		Compress:           true,         // 压缩旧文件
		EnableTimeRotation: false,        // 默认不启用按时间轮转
		TimeRotationFormat: "2006-01-02", // 默认按天轮转
		EnableLevelOutput:  false,        // 默认不启用分级输出
		LevelOutputPaths:   make(map[string][]string),
		LevelOutputMode:    "above", // 默认输出该级别及以上的日志
	}
}

// Validate validate the options fields.
func (o *Options) Validate() []error {
	var errs []error

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		errs = append(errs, err)
	}

	format := strings.ToLower(o.Format)
	if format != consoleFormat && format != jsonFormat {
		errs = append(errs, fmt.Errorf("not a valid log format: %q", o.Format))
	}

	// 验证日志轮转配置
	if o.MaxSize <= 0 {
		errs = append(errs, fmt.Errorf("max-size must be greater than 0"))
	}

	if o.MaxAge < 0 {
		errs = append(errs, fmt.Errorf("max-age cannot be negative"))
	}

	if o.MaxBackups < 0 {
		errs = append(errs, fmt.Errorf("max-backups cannot be negative"))
	}

	return errs
}

// AddFlags adds flags for log to the specified FlagSet object.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Level, flagLevel, o.Level, "Minimum log output `LEVEL`.")
	fs.BoolVar(&o.DisableCaller, flagDisableCaller, o.DisableCaller, "Disable output of caller information in the log.")
	fs.BoolVar(&o.DisableStacktrace, flagDisableStacktrace,
		o.DisableStacktrace, "Disable the log to record a stack trace for all messages at or above panic level.")
	fs.StringVar(&o.Format, flagFormat, o.Format, "Log output `FORMAT`, support plain or json format.")
	fs.BoolVar(&o.EnableColor, flagEnableColor, o.EnableColor, "Enable output ansi colors in plain format logs.")
	fs.StringSliceVar(&o.OutputPaths, flagOutputPaths, o.OutputPaths, "Output paths of log.")
	fs.StringSliceVar(&o.ErrorOutputPaths, flagErrorOutputPaths, o.ErrorOutputPaths, "Error output paths of log.")
	fs.BoolVar(
		&o.Development,
		flagDevelopment,
		o.Development,
		"Development puts the logger in development mode, which changes "+
			"the behavior of DPanicLevel and takes stacktraces more liberally.",
	)
	fs.StringVar(&o.Name, flagName, o.Name, "The name of the logger.")

	// 添加日志轮转相关的命令行参数
	fs.IntVar(&o.MaxSize, "log.max-size", o.MaxSize, "Maximum size in megabytes of the log file before it gets rotated.")
	fs.IntVar(&o.MaxAge, "log.max-age", o.MaxAge, "Maximum number of days to retain old log files.")
	fs.IntVar(&o.MaxBackups, "log.max-backups", o.MaxBackups, "Maximum number of old log files to retain.")
	fs.BoolVar(&o.Compress, "log.compress", o.Compress, "Compress rotated log files.")

	// 添加按时间轮转相关的命令行参数
	fs.BoolVar(&o.EnableTimeRotation, "log.enable-time-rotation", o.EnableTimeRotation,
		"Enable time-based log rotation (e.g., daily rotation).")
	fs.StringVar(&o.TimeRotationFormat, "log.time-rotation-format", o.TimeRotationFormat,
		"Time rotation format (e.g., '2006-01-02' for daily, '2006-01-02-15' for hourly).")

	// 添加日志分级输出相关的命令行参数
	fs.BoolVar(&o.EnableLevelOutput, "log.enable-level-output", o.EnableLevelOutput,
		"Enable level-based log output to different files.")
	fs.StringVar(&o.LevelOutputMode, "log.level-output-mode", o.LevelOutputMode,
		"Level output mode: 'exact' (only exact level) or 'above' (level and above).")
}

func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

// Build constructs a global zap logger from the Config and Options.
func (o *Options) Build() error {
	// 检查是否有文件输出路径，如果有则使用轮转功能
	hasFileOutput := false
	for _, path := range o.OutputPaths {
		if path != "stdout" && path != "stderr" {
			hasFileOutput = true
			break
		}
	}
	for _, path := range o.ErrorOutputPaths {
		if path != "stdout" && path != "stderr" {
			hasFileOutput = true
			break
		}
	}

	// 如果有文件输出，使用轮转功能
	if hasFileOutput {
		logger := NewWithRotation(o)
		zap.RedirectStdLog(logger.Named(o.Name))
		zap.ReplaceGlobals(logger)
		return nil
	}

	// 原有的逻辑（无文件输出时）
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	encodeLevel := zapcore.CapitalLevelEncoder
	if o.Format == consoleFormat && o.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zc := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       o.Development,
		DisableCaller:     o.DisableCaller,
		DisableStacktrace: o.DisableStacktrace,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: o.Format,
		EncoderConfig: zapcore.EncoderConfig{
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
			EncodeName:     zapcore.FullNameEncoder,
		},
		OutputPaths:      o.OutputPaths,
		ErrorOutputPaths: o.ErrorOutputPaths,
	}
	logger, err := zc.Build(zap.AddStacktrace(zapcore.PanicLevel))
	if err != nil {
		return err
	}
	zap.RedirectStdLog(logger.Named(o.Name))
	zap.ReplaceGlobals(logger)

	return nil
}
