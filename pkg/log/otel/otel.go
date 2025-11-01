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

// Package otel 提供 OpenTelemetry 集成
//
// 要使用此包，需要先安装 OpenTelemetry 依赖：
// go get go.opentelemetry.io/otel
// go get go.opentelemetry.io/otel/trace
//
// 使用示例：
//
//	import "github.com/FangcunMount/component-base/pkg/log/otel"
//
//	// 启用 OpenTelemetry 支持
//	otel.EnableOtelExtraction()
//
//	// 使用带追踪的日志
//	log.InfoContext(ctx, "处理请求")
package otel

/*
import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"github.com/FangcunMount/component-base/pkg/log"
)

var extractFromOtel = false

// EnableOtelExtraction 启用从 OpenTelemetry 提取追踪信息
func EnableOtelExtraction() {
	extractFromOtel = true
	// 覆盖默认的提取函数
	log.SetTraceExtractor(ExtractFromOtel)
}

// ExtractFromOtel 从 OpenTelemetry context 提取追踪信息
func ExtractFromOtel(ctx context.Context) (traceID, spanID string) {
	if ctx == nil {
		return "", ""
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", ""
	}

	return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
}
*/

// 说明：
// 由于 OpenTelemetry 是可选依赖，默认情况下不启用。
// 如果需要使用 OpenTelemetry，请：
// 1. 安装依赖：go get go.opentelemetry.io/otel/trace
// 2. 取消注释上面的代码
// 3. 在应用启动时调用 otel.EnableOtelExtraction()
