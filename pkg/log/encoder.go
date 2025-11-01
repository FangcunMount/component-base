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
	"time"

	"go.uber.org/zap/zapcore"
)

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func milliSecondsDurationEncoder(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendFloat64(float64(d) / float64(time.Millisecond))
}

// customLevelEncoder 自定义日志级别编码器，添加方括号和颜色
func customLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString(fmt.Sprintf("%s[DEBUG]%s", colorCyan, colorReset))
	case zapcore.InfoLevel:
		enc.AppendString(fmt.Sprintf("%s[INFO]%s", colorGreen, colorReset))
	case zapcore.WarnLevel:
		enc.AppendString(fmt.Sprintf("%s[WARN]%s", colorYellow, colorReset))
	case zapcore.ErrorLevel:
		enc.AppendString(fmt.Sprintf("%s[ERROR]%s", colorRed, colorReset))
	case zapcore.DPanicLevel:
		enc.AppendString(fmt.Sprintf("%s[DPANIC]%s", colorRed, colorReset))
	case zapcore.PanicLevel:
		enc.AppendString(fmt.Sprintf("%s[PANIC]%s", colorRed, colorReset))
	case zapcore.FatalLevel:
		enc.AppendString(fmt.Sprintf("%s[FATAL]%s", colorRed, colorReset))
	default:
		enc.AppendString(fmt.Sprintf("[%s]", l.CapitalString()))
	}
}

// customLevelEncoderNoColor 不带颜色的日志级别编码器，仅添加方括号
func customLevelEncoderNoColor(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString("[DEBUG]")
	case zapcore.InfoLevel:
		enc.AppendString("[INFO]")
	case zapcore.WarnLevel:
		enc.AppendString("[WARN]")
	case zapcore.ErrorLevel:
		enc.AppendString("[ERROR]")
	case zapcore.DPanicLevel:
		enc.AppendString("[DPANIC]")
	case zapcore.PanicLevel:
		enc.AppendString("[PANIC]")
	case zapcore.FatalLevel:
		enc.AppendString("[FATAL]")
	default:
		enc.AppendString(fmt.Sprintf("[%s]", l.CapitalString()))
	}
}
