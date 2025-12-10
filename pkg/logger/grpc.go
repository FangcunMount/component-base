package logger

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// GRPCServerLoggerConfig gRPC 服务端日志配置
type GRPCServerLoggerConfig struct {
	// LogRequestPayload 是否记录请求载荷
	LogRequestPayload bool
	// LogResponsePayload 是否记录响应载荷
	LogResponsePayload bool
	// MaxPayloadSize 最大记录的载荷大小
	MaxPayloadSize int
	// SkipMethods 跳过日志记录的方法列表
	SkipMethods []string
}

// DefaultGRPCServerLoggerConfig 默认 gRPC 服务端日志配置
func DefaultGRPCServerLoggerConfig() GRPCServerLoggerConfig {
	return GRPCServerLoggerConfig{
		LogRequestPayload:  true,
		LogResponsePayload: true,
		MaxPayloadSize:     2048, // 2KB
		SkipMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		},
	}
}

// UnaryServerLoggingInterceptor gRPC 一元服务端日志拦截器
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return UnaryServerLoggingInterceptorWithConfig(DefaultGRPCServerLoggerConfig())
}

// UnaryServerLoggingInterceptorWithConfig 带配置的 gRPC 一元服务端日志拦截器
func UnaryServerLoggingInterceptorWithConfig(config GRPCServerLoggerConfig) grpc.UnaryServerInterceptor {
	skipMethods := buildSkipMethodsMap(config.SkipMethods)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过指定方法
		if _, ok := skipMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		start := time.Now()

		// 提取或生成追踪 ID
		traceID, spanID, requestID := extractOrGenerateTraceIDs(ctx)
		ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

		// 提取客户端信息
		clientIP := extractGRPCClientIP(ctx)
		service, method := splitGRPCMethodName(info.FullMethod)

		// 创建请求范围的 Logger
		reqLogger := NewRequestLogger(ctx,
			log.String(FieldGRPCService, service),
			log.String(FieldGRPCMethod, method),
			log.String(FieldClientIP, clientIP),
		)
		ctx = WithLogger(ctx, reqLogger)

		// 记录请求开始
		startFields := []interface{}{
			"event", EventRequestStart,
			"grpc.full_method", info.FullMethod,
			"client_ip", clientIP,
		}
		if config.LogRequestPayload && req != nil {
			if payload := formatGRPCPayload(req, config.MaxPayloadSize); payload != "" {
				startFields = append(startFields, "request", payload)
			}
		}
		reqLogger.Infow("gRPC request started", startFields...)

		// 执行处理
		resp, err := handler(ctx, req)

		// 计算耗时
		latency := time.Since(start)

		// 记录请求结束
		endFields := []interface{}{
			"event", EventRequestEnd,
			"grpc.full_method", info.FullMethod,
			"duration_ms", latency.Milliseconds(),
		}

		if err != nil {
			st := status.Convert(err)
			endFields = append(endFields,
				"grpc.code", st.Code().String(),
				"error", st.Message(),
				"result", ResultFailed,
			)
			reqLogger.Errorw("gRPC request failed", endFields...)
		} else {
			endFields = append(endFields,
				"grpc.code", "OK",
				"result", ResultSuccess,
			)
			if config.LogResponsePayload && resp != nil {
				if payload := formatGRPCPayload(resp, config.MaxPayloadSize); payload != "" {
					endFields = append(endFields, "response", payload)
				}
			}
			reqLogger.Infow("gRPC request completed", endFields...)
		}

		return resp, err
	}
}

// StreamServerLoggingInterceptor gRPC 流式服务端日志拦截器
func StreamServerLoggingInterceptor() grpc.StreamServerInterceptor {
	return StreamServerLoggingInterceptorWithConfig(DefaultGRPCServerLoggerConfig())
}

// StreamServerLoggingInterceptorWithConfig 带配置的 gRPC 流式服务端日志拦截器
func StreamServerLoggingInterceptorWithConfig(config GRPCServerLoggerConfig) grpc.StreamServerInterceptor {
	skipMethods := buildSkipMethodsMap(config.SkipMethods)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 跳过指定方法
		if _, ok := skipMethods[info.FullMethod]; ok {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		start := time.Now()

		// 提取或生成追踪 ID
		traceID, spanID, requestID := extractOrGenerateTraceIDs(ctx)
		ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

		// 提取客户端信息
		clientIP := extractGRPCClientIP(ctx)
		service, method := splitGRPCMethodName(info.FullMethod)

		// 创建请求范围的 Logger
		reqLogger := NewRequestLogger(ctx,
			log.String(FieldGRPCService, service),
			log.String(FieldGRPCMethod, method),
			log.String(FieldClientIP, clientIP),
		)
		ctx = WithLogger(ctx, reqLogger)

		// 记录流开始
		reqLogger.Infow("gRPC stream started",
			"event", EventRequestStart,
			"grpc.full_method", info.FullMethod,
			"client_ip", clientIP,
			"is_client_stream", info.IsClientStream,
			"is_server_stream", info.IsServerStream,
		)

		// 包装 ServerStream 以注入 context
		wrappedStream := &loggingServerStream{
			ServerStream: ss,
			ctx:          ctx,
			logger:       reqLogger,
			method:       info.FullMethod,
		}

		// 执行处理
		err := handler(srv, wrappedStream)

		// 计算耗时
		latency := time.Since(start)

		// 记录流结束
		if err != nil {
			st := status.Convert(err)
			reqLogger.Errorw("gRPC stream failed",
				"event", EventRequestEnd,
				"grpc.full_method", info.FullMethod,
				"grpc.code", st.Code().String(),
				"error", st.Message(),
				"duration_ms", latency.Milliseconds(),
				"result", ResultFailed,
			)
		} else {
			reqLogger.Infow("gRPC stream completed",
				"event", EventRequestEnd,
				"grpc.full_method", info.FullMethod,
				"grpc.code", "OK",
				"duration_ms", latency.Milliseconds(),
				"result", ResultSuccess,
			)
		}

		return err
	}
}

// loggingServerStream 包装 grpc.ServerStream 以支持日志和 context 注入
type loggingServerStream struct {
	grpc.ServerStream
	ctx    context.Context
	logger *RequestLogger
	method string
}

func (s *loggingServerStream) Context() context.Context {
	return s.ctx
}

func (s *loggingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		s.logger.Warnw("gRPC stream send error",
			"grpc.full_method", s.method,
			"error", err.Error(),
		)
	}
	return err
}

func (s *loggingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		s.logger.Debugw("gRPC stream receive",
			"grpc.full_method", s.method,
			"error", err.Error(),
		)
	}
	return err
}

// ============================================================================
// 辅助函数
// ============================================================================

// extractOrGenerateTraceIDs 从 context 提取或生成追踪 ID
func extractOrGenerateTraceIDs(ctx context.Context) (traceID, spanID, requestID string) {
	// 尝试从 metadata 中获取
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-trace-id"); len(values) > 0 {
			traceID = values[0]
		}
		if values := md.Get("x-request-id"); len(values) > 0 {
			requestID = values[0]
		}
	}

	// 如果没有则生成
	if traceID == "" {
		traceID = idutil.NewTraceID()
	}
	if requestID == "" {
		requestID = idutil.NewRequestID()
	}
	spanID = idutil.NewSpanID()

	return
}

// extractGRPCClientIP 从 context 提取客户端 IP
func extractGRPCClientIP(ctx context.Context) string {
	// 尝试从 peer 获取
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		addr := p.Addr.String()
		// 移除端口号
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			return addr[:idx]
		}
		return addr
	}

	// 尝试从 metadata 获取（可能经过代理）
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-forwarded-for"); len(values) > 0 {
			// 取第一个 IP
			ips := strings.Split(values[0], ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
		if values := md.Get("x-real-ip"); len(values) > 0 {
			return values[0]
		}
	}

	return "unknown"
}

// splitGRPCMethodName 分割 gRPC 方法名为服务名和方法名
func splitGRPCMethodName(fullMethod string) (service, method string) {
	// fullMethod 格式: /package.Service/Method
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	if idx := strings.LastIndex(fullMethod, "/"); idx >= 0 {
		return fullMethod[:idx], fullMethod[idx+1:]
	}
	return fullMethod, ""
}

// formatGRPCPayload 格式化 gRPC 载荷
func formatGRPCPayload(payload interface{}, maxSize int) string {
	if payload == nil {
		return ""
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	result := string(data)
	if len(result) > maxSize {
		return result[:maxSize] + "..."
	}

	return result
}

// buildSkipMethodsMap 构建跳过方法的 map
func buildSkipMethodsMap(methods []string) map[string]struct{} {
	m := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		m[method] = struct{}{}
	}
	return m
}
