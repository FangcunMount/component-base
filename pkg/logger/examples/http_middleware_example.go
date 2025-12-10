package examples

import (
	"github.com/FangcunMount/component-base/pkg/logger"
	"github.com/gin-gonic/gin"
)

// SetupHTTPMiddleware 演示如何在 Gin 中配置 HTTP 日志中间件
func SetupHTTPMiddleware() *gin.Engine {
	engine := gin.New()

	// 方式一：使用默认配置
	engine.Use(logger.HTTPLogger())

	// 方式二：使用自定义配置
	config := logger.HTTPLoggerConfig{
		Tag: "http.access",
		// 跳过健康检查等路径的日志记录
		SkipPaths: []string{"/health", "/healthz", "/metrics", "/favicon.ico"},
		// 是否记录请求头
		LogRequestHeaders: true,
		// 是否记录请求体
		LogRequestBody: true,
		// 是否记录响应头
		LogResponseHeaders: true,
		// 是否记录响应体
		LogResponseBody: true,
		// 是否对敏感数据脱敏
		MaskSensitiveData: true,
		// 最大记录的 body 大小 (16KB)
		MaxBodyBytes: 16 * 1024,
		// context 中 Request ID 的 key
		RequestIDKey: "X-Request-ID",
	}
	engine.Use(logger.HTTPLoggerWithConfig(config))

	return engine
}

// SetupHTTPMiddlewareMinimal 演示最简配置
func SetupHTTPMiddlewareMinimal() *gin.Engine {
	engine := gin.New()

	// 只需一行即可启用完整的请求日志
	engine.Use(logger.HTTPLogger())

	// 配置路由
	engine.GET("/api/users", func(c *gin.Context) {
		// 在 Handler 中可以直接获取请求范围的 Logger
		l := logger.L(c.Request.Context())
		l.Infow("处理用户列表请求",
			"action", logger.ActionList,
			"resource", logger.ResourceUser,
		)
		c.JSON(200, gin.H{"users": []string{}})
	})

	return engine
}

// HTTPMiddlewareOutput 说明中间件产生的日志格式
//
// 请求开始时的日志：
//
//	{
//	    "level": "info",
//	    "ts": "2025-12-10T10:00:00.000Z",
//	    "msg": "HTTP Request Started",
//	    "event": "request_start",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "method": "POST",
//	    "path": "/v1/users",
//	    "query": "page=1&size=10",
//	    "client_ip": "192.168.1.100",
//	    "user_agent": "Mozilla/5.0...",
//	    "request_headers": {"Content-Type": "application/json", ...}
//	}
//
// 请求结束时的日志：
//
//	{
//	    "level": "info",
//	    "ts": "2025-12-10T10:00:00.150Z",
//	    "msg": "HTTP Request Completed Successfully",
//	    "event": "request_end",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "method": "POST",
//	    "path": "/v1/users",
//	    "status_code": 201,
//	    "duration_ms": 150,
//	    "response_size": 256,
//	    "request_body": "{\"username\": \"john\", \"password\": \"***\"}",
//	    "response_body": "{\"id\": \"12345\", \"username\": \"john\"}"
//	}
func HTTPMiddlewareOutput() {}
