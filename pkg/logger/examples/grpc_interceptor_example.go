package examples

import (
	"github.com/FangcunMount/component-base/pkg/logger"
	"google.golang.org/grpc"
)

// SetupGRPCServer 演示如何在 gRPC 服务器中配置日志拦截器
func SetupGRPCServer() *grpc.Server {
	// 方式一：使用默认配置
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// 一元调用日志拦截器
			logger.UnaryServerLoggingInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			// 流式调用日志拦截器
			logger.StreamServerLoggingInterceptor(),
		),
	)

	return server
}

// SetupGRPCServerWithConfig 演示如何使用自定义配置
func SetupGRPCServerWithConfig() *grpc.Server {
	// 自定义配置
	config := logger.GRPCServerLoggerConfig{
		// 是否记录请求载荷
		LogRequestPayload: true,
		// 是否记录响应载荷
		LogResponsePayload: true,
		// 最大记录的载荷大小 (2KB)
		MaxPayloadSize: 2048,
		// 跳过日志记录的方法列表（如健康检查）
		SkipMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		},
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.UnaryServerLoggingInterceptorWithConfig(config),
		),
		grpc.ChainStreamInterceptor(
			logger.StreamServerLoggingInterceptorWithConfig(config),
		),
	)

	return server
}

// SetupGRPCServerWithMultipleInterceptors 演示多拦截器链配置
func SetupGRPCServerWithMultipleInterceptors() *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// 1. Recovery 拦截器（捕获 panic）
			// middleware.RecoveryInterceptor(),
			// 2. 日志拦截器
			logger.UnaryServerLoggingInterceptor(),
			// 3. 认证拦截器
			// middleware.AuthInterceptor(),
			// 4. 授权拦截器
			// middleware.AuthzInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			// middleware.StreamRecoveryInterceptor(),
			logger.StreamServerLoggingInterceptor(),
		),
	)

	return server
}

// GRPCInterceptorOutput 说明拦截器产生的日志格式
//
// 一元调用开始时的日志：
//
//	{
//	    "level": "info",
//	    "ts": "2025-12-10T10:00:00.000Z",
//	    "msg": "gRPC request started",
//	    "event": "request_start",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "grpc.service": "iam.authn.v1.AuthnService",
//	    "grpc.method": "Login",
//	    "grpc.full_method": "/iam.authn.v1.AuthnService/Login",
//	    "client_ip": "192.168.1.100",
//	    "request": "{\"app_code\": \"test-app\", \"login_type\": \"wechat\"}"
//	}
//
// 一元调用成功结束时的日志：
//
//	{
//	    "level": "info",
//	    "ts": "2025-12-10T10:00:00.050Z",
//	    "msg": "gRPC request completed",
//	    "event": "request_end",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "grpc.full_method": "/iam.authn.v1.AuthnService/Login",
//	    "grpc.code": "OK",
//	    "duration_ms": 50,
//	    "result": "success",
//	    "response": "{\"access_token\": \"eyJ...\", \"expires_in\": 3600}"
//	}
//
// 一元调用失败时的日志：
//
//	{
//	    "level": "error",
//	    "ts": "2025-12-10T10:00:00.030Z",
//	    "msg": "gRPC request failed",
//	    "event": "request_end",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "grpc.full_method": "/iam.authn.v1.AuthnService/Login",
//	    "grpc.code": "Unauthenticated",
//	    "error": "invalid credentials",
//	    "duration_ms": 30,
//	    "result": "failed"
//	}
//
// 流式调用日志格式类似，额外包含：
//   - is_client_stream: 是否为客户端流
//   - is_server_stream: 是否为服务端流
func GRPCInterceptorOutput() {}
