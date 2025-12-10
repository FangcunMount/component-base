package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/gin-gonic/gin"
)

const (
	defaultHTTPLogTag = "http.access"
	// 最大记录的请求/响应体大小 (16KB)
	defaultMaxBodySize = 16 * 1024
)

// 敏感字段列表，需要脱敏处理
var sensitiveFields = []string{
	"password", "secret", "token", "authorization", "api_key", "apikey",
	"access_token", "refresh_token", "private_key", "client_secret",
}

// HTTPLoggerConfig 定义 HTTP 日志中间件的可配置项
type HTTPLoggerConfig struct {
	// Tag 日志标签
	Tag string
	// SkipPaths 跳过日志记录的路径
	SkipPaths []string
	// LogRequestHeaders 是否记录请求头
	LogRequestHeaders bool
	// LogRequestBody 是否记录请求体
	LogRequestBody bool
	// LogResponseHeaders 是否记录响应头
	LogResponseHeaders bool
	// LogResponseBody 是否记录响应体
	LogResponseBody bool
	// MaskSensitiveData 是否对敏感数据脱敏
	MaskSensitiveData bool
	// MaxBodyBytes 最大记录的 body 大小
	MaxBodyBytes int64
	// RequestIDKey context 中 Request ID 的 key
	RequestIDKey string
}

// DefaultHTTPLoggerConfig 返回默认配置
func DefaultHTTPLoggerConfig() HTTPLoggerConfig {
	return HTTPLoggerConfig{
		Tag:                defaultHTTPLogTag,
		SkipPaths:          []string{"/health", "/healthz", "/metrics", "/favicon.ico"},
		LogRequestHeaders:  true,
		LogRequestBody:     true,
		LogResponseHeaders: true,
		LogResponseBody:    true,
		MaskSensitiveData:  true,
		MaxBodyBytes:       defaultMaxBodySize,
		RequestIDKey:       "X-Request-ID",
	}
}

// HTTPLogger 返回 Gin HTTP 日志中间件
func HTTPLogger() gin.HandlerFunc {
	return HTTPLoggerWithConfig(DefaultHTTPLoggerConfig())
}

// HTTPLoggerWithConfig 返回带配置的 Gin HTTP 日志中间件
func HTTPLoggerWithConfig(config HTTPLoggerConfig) gin.HandlerFunc {
	cfg := config.withDefaults()
	skipPaths := buildSkipPathsMap(cfg.SkipPaths)

	return func(c *gin.Context) {
		if _, ok := skipPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		requestID := c.GetString(cfg.RequestIDKey)

		// === 1. 创建请求范围的 Logger 并注入 context ===
		reqLogger := NewRequestLogger(c.Request.Context(),
			log.String(FieldMethod, c.Request.Method),
			log.String(FieldPath, c.Request.URL.Path),
			log.String(FieldClientIP, c.ClientIP()),
			log.String(FieldRequestID, requestID),
		)
		ctx := WithLogger(c.Request.Context(), reqLogger)
		c.Request = c.Request.WithContext(ctx)

		// === 2. 记录请求开始信息 ===
		logHTTPRequestStart(c, cfg, requestID)

		// 读取并缓存请求体
		var requestBody []byte
		if cfg.LogRequestBody && c.Request.Body != nil {
			requestBody = readAndRestoreHTTPRequestBody(c, cfg.MaxBodyBytes)
		}

		// 包装 ResponseWriter 以捕获响应
		writer := newHTTPBodyCaptureWriter(c.Writer, cfg.LogResponseBody, cfg.MaxBodyBytes)
		c.Writer = writer

		// 处理请求
		c.Next()

		// === 3. 记录请求结束信息 ===
		statusCode := writer.Status()
		latency := time.Since(start)
		responseBody := writer.Body()

		logHTTPRequestEnd(c, cfg, requestID, latency, statusCode, requestBody, responseBody)
	}
}

func (cfg HTTPLoggerConfig) withDefaults() HTTPLoggerConfig {
	result := cfg

	if result.Tag == "" {
		result.Tag = defaultHTTPLogTag
	}
	if result.MaxBodyBytes <= 0 {
		result.MaxBodyBytes = defaultMaxBodySize
	}
	if result.RequestIDKey == "" {
		result.RequestIDKey = "X-Request-ID"
	}

	return result
}

func buildSkipPathsMap(paths []string) map[string]struct{} {
	if len(paths) == 0 {
		return map[string]struct{}{}
	}

	skip := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		skip[path] = struct{}{}
	}

	return skip
}

type httpBodyCaptureWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	limitBytes int64
	capture    bool
}

func newHTTPBodyCaptureWriter(w gin.ResponseWriter, capture bool, limit int64) *httpBodyCaptureWriter {
	var buffer *bytes.Buffer
	if capture {
		buffer = &bytes.Buffer{}
	}

	return &httpBodyCaptureWriter{
		ResponseWriter: w,
		body:           buffer,
		statusCode:     http.StatusOK,
		limitBytes:     limit,
		capture:        capture,
	}
}

func (w *httpBodyCaptureWriter) Write(data []byte) (int, error) {
	if w.capture && w.body != nil && len(data) > 0 {
		if w.limitBytes <= 0 || int64(w.body.Len()) < w.limitBytes {
			remaining := len(data)
			if w.limitBytes > 0 {
				remaining = int(minInt64Value(w.limitBytes-int64(w.body.Len()), int64(len(data))))
			}
			if remaining > 0 {
				w.body.Write(data[:remaining])
			}
		}
	}

	return w.ResponseWriter.Write(data)
}

func (w *httpBodyCaptureWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *httpBodyCaptureWriter) Status() int {
	return w.statusCode
}

func (w *httpBodyCaptureWriter) Body() []byte {
	if !w.capture || w.body == nil {
		return nil
	}
	return w.body.Bytes()
}

func minInt64Value(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// readAndRestoreHTTPRequestBody 读取请求体并恢复到请求中
func readAndRestoreHTTPRequestBody(c *gin.Context, maxSize int64) []byte {
	if c.Request.Body == nil {
		return nil
	}

	// 限制读取大小
	reader := io.LimitReader(c.Request.Body, maxSize)
	body, err := io.ReadAll(reader)
	if err != nil {
		log.Warnw("Failed to read request body", "error", err)
		return nil
	}

	// 恢复请求体
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	return body
}

// logHTTPRequestStart 记录请求开始信息
func logHTTPRequestStart(c *gin.Context, config HTTPLoggerConfig, requestID string) {
	fields := []log.Field{
		log.String("event", EventRequestStart),
		log.String("request_id", requestID),
		log.String("method", c.Request.Method),
		log.String("path", c.Request.URL.Path),
		log.String("query", c.Request.URL.RawQuery),
		log.String("client_ip", c.ClientIP()),
		log.String("user_agent", c.Request.UserAgent()),
		log.Time("timestamp", time.Now()),
	}

	// 记录请求头
	if config.LogRequestHeaders {
		headers := make(map[string]string)
		for name, values := range c.Request.Header {
			if len(values) > 0 {
				value := values[0]
				if config.MaskSensitiveData && isSensitiveHTTPHeader(name) {
					value = maskHTTPSensitiveValue(value)
				}
				headers[name] = value
			}
		}
		fields = append(fields, log.Any("request_headers", headers))
	}

	// 添加分布式追踪字段
	fields = append(fields, log.TraceFields(c.Request.Context())...)

	log.HTTP("HTTP Request Started", fields...)
}

// logHTTPRequestEnd 记录请求结束信息
func logHTTPRequestEnd(c *gin.Context, config HTTPLoggerConfig, requestID string, latency time.Duration, statusCode int, requestBody, responseBody []byte) {
	fields := []log.Field{
		log.String("event", EventRequestEnd),
		log.String("request_id", requestID),
		log.String("method", c.Request.Method),
		log.String("path", c.Request.URL.Path),
		log.Int("status_code", statusCode),
		log.Int64("duration_ms", latency.Milliseconds()),
		log.Int("response_size", len(responseBody)),
		log.Time("timestamp", time.Now()),
	}

	// 记录请求体
	if config.LogRequestBody && len(requestBody) > 0 {
		bodyStr := string(requestBody)
		if config.MaskSensitiveData {
			bodyStr = maskHTTPSensitiveJSON(bodyStr)
		}
		fields = append(fields, log.String("request_body", bodyStr))
	}

	// 记录响应头
	if config.LogResponseHeaders {
		headers := make(map[string]string)
		for name, values := range c.Writer.Header() {
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}
		fields = append(fields, log.Any("response_headers", headers))
	}

	// 记录响应体
	if config.LogResponseBody && len(responseBody) > 0 {
		bodyStr := string(responseBody)
		if config.MaskSensitiveData {
			bodyStr = maskHTTPSensitiveJSON(bodyStr)
		}
		fields = append(fields, log.String("response_body", bodyStr))
	}

	// 记录错误信息
	if len(c.Errors) > 0 {
		fields = append(fields, log.String("errors", c.Errors.String()))
	}

	// 添加分布式追踪字段
	fields = append(fields, log.TraceFields(c.Request.Context())...)

	// 根据状态码选择日志级别
	if statusCode >= http.StatusInternalServerError {
		log.HTTPError("HTTP Request Completed with Server Error", fields...)
	} else if statusCode >= http.StatusBadRequest {
		log.HTTPWarn("HTTP Request Completed with Client Error", fields...)
	} else {
		log.HTTPDebug("HTTP Request Completed Successfully", fields...)
	}
}

// isSensitiveHTTPHeader 判断是否为敏感的请求头
func isSensitiveHTTPHeader(name string) bool {
	name = strings.ToLower(name)
	return name == "authorization" || name == "cookie" || name == "x-api-key"
}

// maskHTTPSensitiveValue 对敏感值进行脱敏处理
func maskHTTPSensitiveValue(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "***" + value[len(value)-4:]
}

// maskHTTPSensitiveJSON 对 JSON 字符串中的敏感字段进行脱敏
func maskHTTPSensitiveJSON(jsonStr string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	maskHTTPSensitiveInData(data)

	masked, err := json.Marshal(data)
	if err != nil {
		return jsonStr
	}

	return string(masked)
}

// maskHTTPSensitiveInData 递归处理数据结构中的敏感字段
func maskHTTPSensitiveInData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if isSensitiveHTTPField(key) {
				if str, ok := value.(string); ok {
					v[key] = maskHTTPSensitiveValue(str)
				}
			} else {
				v[key] = maskHTTPSensitiveInData(value)
			}
		}
	case []interface{}:
		for i, item := range v {
			v[i] = maskHTTPSensitiveInData(item)
		}
	}
	return data
}

// isSensitiveHTTPField 判断字段名是否为敏感字段
func isSensitiveHTTPField(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if lower == sensitive {
			return true
		}
	}
	return false
}
