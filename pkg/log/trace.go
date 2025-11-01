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
	"context"

	"go.uber.org/zap"
)

// TraceContextKey 用于在 context 中存储自定义 trace ID
type TraceContextKey string

const (
	// TraceIDKey trace ID 的 context key
	TraceIDKey TraceContextKey = "trace_id"
	// SpanIDKey span ID 的 context key
	SpanIDKey TraceContextKey = "span_id"
	// RequestIDKey request ID 的 context key
	RequestIDKey TraceContextKey = "request_id"
)

// TraceField 返回 trace ID 字段
func TraceID(ctx context.Context) zap.Field {
	traceID := ExtractTraceID(ctx)
	if traceID == "" {
		return zap.Skip()
	}
	return zap.String("trace_id", traceID)
}

// SpanID 返回 span ID 字段
func SpanID(ctx context.Context) zap.Field {
	spanID := ExtractSpanID(ctx)
	if spanID == "" {
		return zap.Skip()
	}
	return zap.String("span_id", spanID)
}

// RequestID 返回 request ID 字段
func RequestID(ctx context.Context) zap.Field {
	requestID := ExtractRequestID(ctx)
	if requestID == "" {
		return zap.Skip()
	}
	return zap.String("request_id", requestID)
}

// TraceFields 返回所有追踪相关字段
func TraceFields(ctx context.Context) []zap.Field {
	var fields []zap.Field

	if traceID := ExtractTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID := ExtractSpanID(ctx); spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	if requestID := ExtractRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	return fields
}

// ExtractTraceID 从 context 中提取 trace ID
// 支持 OpenTelemetry 和自定义 trace ID
func ExtractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 尝试从自定义 context 提取
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}

	return ""
}

// ExtractSpanID 从 context 中提取 span ID
func ExtractSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 尝试从自定义 context 提取
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}

	return ""
}

// ExtractRequestID 从 context 中提取 request ID
func ExtractRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}

	return ""
}

// WithTraceID 将 trace ID 添加到 context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID 将 span ID 添加到 context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithRequestID 将 request ID 添加到 context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTraceContext 将完整的追踪上下文添加到 context
func WithTraceContext(ctx context.Context, traceID, spanID, requestID string) context.Context {
	ctx = WithTraceID(ctx, traceID)
	ctx = WithSpanID(ctx, spanID)
	if requestID != "" {
		ctx = WithRequestID(ctx, requestID)
	}
	return ctx
}

// InfoContext 记录带追踪信息的 info 日志
func InfoContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(TraceFields(ctx), fields...)
	std.Info(msg, fields...)
}

// DebugContext 记录带追踪信息的 debug 日志
func DebugContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(TraceFields(ctx), fields...)
	std.Debug(msg, fields...)
}

// WarnContext 记录带追踪信息的 warn 日志
func WarnContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(TraceFields(ctx), fields...)
	std.Warn(msg, fields...)
}

// ErrorContext 记录带追踪信息的 error 日志
func ErrorContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(TraceFields(ctx), fields...)
	std.Error(msg, fields...)
}

// InfofContext 记录带追踪信息的格式化 info 日志
func InfofContext(ctx context.Context, format string, v ...interface{}) {
	fields := TraceFields(ctx)
	if len(fields) > 0 {
		std.zapLogger.Sugar().With(fields).Infof(format, v...)
	} else {
		std.Infof(format, v...)
	}
}

// DebugfContext 记录带追踪信息的格式化 debug 日志
func DebugfContext(ctx context.Context, format string, v ...interface{}) {
	fields := TraceFields(ctx)
	if len(fields) > 0 {
		std.zapLogger.Sugar().With(fields).Debugf(format, v...)
	} else {
		std.Debugf(format, v...)
	}
}

// WarnfContext 记录带追踪信息的格式化 warn 日志
func WarnfContext(ctx context.Context, format string, v ...interface{}) {
	fields := TraceFields(ctx)
	if len(fields) > 0 {
		std.zapLogger.Sugar().With(fields).Warnf(format, v...)
	} else {
		std.Warnf(format, v...)
	}
}

// ErrorfContext 记录带追踪信息的格式化 error 日志
func ErrorfContext(ctx context.Context, format string, v ...interface{}) {
	fields := TraceFields(ctx)
	if len(fields) > 0 {
		std.zapLogger.Sugar().With(fields).Errorf(format, v...)
	} else {
		std.Errorf(format, v...)
	}
}
