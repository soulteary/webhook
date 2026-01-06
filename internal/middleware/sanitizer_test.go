package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通字符串",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "包含password的字符串",
			input:    "password123",
			expected: "***",
		},
		{
			name:     "包含token的字符串",
			input:    "token=abc123",
			expected: "token=***",
		},
		{
			name:     "键值对格式",
			input:    "key=value",
			expected: "key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "普通头",
			key:      "Content-Type",
			value:    "application/json",
			expected: "application/json",
		},
		{
			name:     "Authorization头",
			key:      "Authorization",
			value:    "Bearer token123",
			expected: "***",
		},
		{
			name:     "API-Key头",
			key:      "X-API-Key",
			value:    "secret123",
			expected: "***",
		},
		{
			name:     "Password头",
			key:      "X-Password",
			value:    "mypassword",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHeader(tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通JSON",
			input:    `{"name":"test","value":123}`,
			expected: `{"name":"test","value":123}`,
		},
		{
			name:     "包含password字段",
			input:    `{"username":"user","password":"secret123"}`,
			expected: `{"password":"***","username":"user"}`,
		},
		{
			name:     "包含token字段",
			input:    `{"token":"abc123","data":"test"}`,
			expected: `{"data":"test","token":"***"}`,
		},
		{
			name:     "嵌套JSON",
			input:    `{"user":{"name":"test","password":"secret"}}`,
			expected: `{"user":{"name":"test","password":"***"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeJSON(tt.input)
			// JSON字段顺序可能不同，所以需要特殊处理
			if tt.name == "普通JSON" {
				// 普通JSON不应该包含敏感信息，应该保持原样
				assert.Contains(t, result, "test")
				assert.Contains(t, result, "123")
				assert.NotContains(t, result, "***")
			} else {
				// 包含敏感字段的JSON应该被脱敏
				assert.Contains(t, result, "***")
				// 验证敏感字段被脱敏
				if tt.name == "包含password字段" {
					assert.Contains(t, result, "password")
					assert.Contains(t, result, "username")
				} else if tt.name == "包含token字段" {
					assert.Contains(t, result, "token")
					assert.Contains(t, result, "data")
				} else if tt.name == "嵌套JSON" {
					assert.Contains(t, result, "user")
					assert.Contains(t, result, "password")
				}
			}
		})
	}
}

func TestSanitizeQueryString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通查询字符串",
			input:    "name=test&value=123",
			expected: "name=test&value=123",
		},
		{
			name:     "包含password参数",
			input:    "username=user&password=secret123",
			expected: "username=user&password=***",
		},
		{
			name:     "包含token参数",
			input:    "token=abc123&data=test",
			expected: "token=***&data=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeQueryString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通错误消息",
			input:    "file not found",
			expected: "file not found",
		},
		{
			name:     "包含password的错误",
			input:    "error: password=secret123",
			expected: "error: password=***",
		},
		{
			name:     "包含token的错误",
			input:    "auth failed: token=abc123",
			expected: "auth failed: token=***",
		},
		{
			name:     "包含Bearer token的错误",
			input:    "Authorization: Bearer abc123def456",
			expected: "Authorization: ***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeRequestBody(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		includeBody bool
		expected    string
	}{
		{
			name:        "不包含请求体",
			contentType: "application/json",
			body:        []byte(`{"password":"secret"}`),
			includeBody: false,
			expected:    "",
		},
		{
			name:        "JSON请求体",
			contentType: "application/json",
			body:        []byte(`{"username":"user","password":"secret"}`),
			includeBody: true,
			expected:    "",
		},
		{
			name:        "表单请求体",
			contentType: "application/x-www-form-urlencoded",
			body:        []byte("username=user&password=secret"),
			includeBody: true,
			expected:    "username=user&password=***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeRequestBody(tt.contentType, tt.body, tt.includeBody)
			if tt.name == "不包含请求体" {
				assert.Empty(t, result)
			} else if tt.name == "表单请求体" {
				assert.Equal(t, tt.expected, result)
			} else {
				// JSON结果可能顺序不同，只检查是否包含脱敏标记
				assert.Contains(t, result, "***")
			}
		})
	}
}
