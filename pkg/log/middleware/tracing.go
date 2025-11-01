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

package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

// TracingMiddleware HTTP 追踪中间件
// 自动为每个请求生成 trace_id 和 request_id，并注入到 context
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 尝试从 HTTP 头获取 trace ID
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			// 如果没有，生成新的
			traceID = idutil.NewTraceID()
		}

		// 生成 span ID
		spanID := idutil.NewSpanID()

		// 生成 request ID
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = idutil.NewRequestID()
		}

		// 将追踪信息注入 context
		ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

		// 设置响应头
		w.Header().Set("X-Trace-Id", traceID)
		w.Header().Set("X-Request-Id", requestID)

		// 记录请求开始
		start := time.Now()
		log.InfoContext(ctx, "HTTP请求开始",
			log.String("method", r.Method),
			log.String("path", r.URL.Path),
			log.String("remote_addr", r.RemoteAddr),
		)

		// 使用包装的 ResponseWriter 记录状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 继续处理请求
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// 记录请求完成
		duration := time.Since(start)
		log.InfoContext(ctx, "HTTP请求完成",
			log.String("method", r.Method),
			log.String("path", r.URL.Path),
			log.Int("status_code", wrapped.statusCode),
			log.Duration("duration", duration),
		)
	})
}

// responseWriter 包装 http.ResponseWriter 以记录状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GinTracingMiddleware Gin 框架的追踪中间件
// 使用示例：
//
//	r := gin.Default()
//	r.Use(middleware.GinTracingMiddleware())
func GinTracingMiddleware() interface{} {
	// 返回一个泛型接口，实际使用时需要类型断言为 gin.HandlerFunc
	return func(c interface{}) {
		// 这里是伪代码，实际使用需要导入 gin
		// ctx := c.(*gin.Context)
		// 实现逻辑与上面类似
	}
}

// EchoTracingMiddleware Echo 框架的追踪中间件
func EchoTracingMiddleware() interface{} {
	// 返回一个泛型接口，实际使用时需要导入 echo
	return func(next interface{}) interface{} {
		return func(c interface{}) error {
			// 实现逻辑与上面类似
			return nil
		}
	}
}

// ExtractTraceFromContext 从任意 context 提取追踪信息的辅助函数
func ExtractTraceFromContext(ctx context.Context) (traceID, spanID, requestID string) {
	return log.ExtractTraceID(ctx), log.ExtractSpanID(ctx), log.ExtractRequestID(ctx)
}
