package errors

import (
	"net/http"
	"testing"
)

// TestDefaultCoderHTTPStatus 测试默认的 HTTPStatus 行为
func TestDefaultCoderHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		coder          defaultCoder
		expectedStatus int
	}{
		{
			name: "未设置 HTTP 状态码，应返回 200",
			coder: defaultCoder{
				C:    10001,
				HTTP: 0,
				Ext:  "用户不存在",
				Ref:  "http://example.com",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "设置了 HTTP 状态码 404，应返回 404",
			coder: defaultCoder{
				C:    10002,
				HTTP: http.StatusNotFound,
				Ext:  "资源不存在",
				Ref:  "http://example.com",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "设置了 HTTP 状态码 400，应返回 400",
			coder: defaultCoder{
				C:    10003,
				HTTP: http.StatusBadRequest,
				Ext:  "参数错误",
				Ref:  "http://example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.coder.HTTPStatus()
			if status != tt.expectedStatus {
				t.Errorf("HTTPStatus() = %d, want %d", status, tt.expectedStatus)
			}
		})
	}
}

// TestUnknownCoderHTTPStatus 测试未知错误的 HTTP 状态码
func TestUnknownCoderHTTPStatus(t *testing.T) {
	status := unknownCoder.HTTPStatus()
	if status != http.StatusInternalServerError {
		t.Errorf("unknownCoder.HTTPStatus() = %d, want %d", status, http.StatusInternalServerError)
	}
}

// TestParseCoderHTTPStatus 测试 ParseCoder 返回的 HTTP 状态码
func TestParseCoderHTTPStatus(t *testing.T) {
	// 注册一个测试错误码
	testCode := 99999
	testCoder := defaultCoder{
		C:    testCode,
		HTTP: 0, // 不设置，期望返回 200
		Ext:  "测试错误",
		Ref:  "http://test.com",
	}
	Register(testCoder)

	// 创建带错误码的错误
	err := WithCode(testCode, "test error")

	// 解析错误码
	coder := ParseCoder(err)
	if coder == nil {
		t.Fatal("ParseCoder returned nil")
	}

	// 验证 HTTP 状态码
	status := coder.HTTPStatus()
	if status != http.StatusOK {
		t.Errorf("ParseCoder().HTTPStatus() = %d, want %d", status, http.StatusOK)
	}
}

// TestParseCoderForUnknownError 测试未知错误的解析
func TestParseCoderForUnknownError(t *testing.T) {
	// 创建一个没有错误码的普通错误
	err := New("unknown error")

	// 解析错误码
	coder := ParseCoder(err)
	if coder == nil {
		t.Fatal("ParseCoder returned nil")
	}

	// 验证返回的是 unknownCoder
	status := coder.HTTPStatus()
	if status != http.StatusInternalServerError {
		t.Errorf("ParseCoder().HTTPStatus() for unknown error = %d, want %d", status, http.StatusInternalServerError)
	}

	if coder.Code() != unknownCoder.Code() {
		t.Errorf("ParseCoder().Code() for unknown error = %d, want %d", coder.Code(), unknownCoder.Code())
	}
}
