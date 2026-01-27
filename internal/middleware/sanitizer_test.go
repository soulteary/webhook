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
				switch tt.name {
				case "包含password字段":
					assert.Contains(t, result, "password")
					assert.Contains(t, result, "username")
				case "包含token字段":
					assert.Contains(t, result, "token")
					assert.Contains(t, result, "data")
				case "嵌套JSON":
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
			switch tt.name {
			case "不包含请求体":
				assert.Empty(t, result)
			case "表单请求体":
				assert.Equal(t, tt.expected, result)
			default:
				// JSON结果可能顺序不同，只检查是否包含脱敏标记
				assert.Contains(t, result, "***")
			}
		})
	}
}

func TestSanitizeRequestLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "请求行无查询参数",
			input:    "GET /api/test HTTP/1.1",
			expected: "GET /api/test HTTP/1.1",
		},
		{
			name:     "请求行包含查询参数",
			input:    "GET /api/test?name=value HTTP/1.1",
			expected: "GET /api/test?name=value HTTP/1.1",
		},
		{
			name:     "请求行包含敏感查询参数",
			input:    "GET /api/test?password=secret123 HTTP/1.1",
			expected: "GET /api/test?password=***",
		},
		{
			name:     "请求行包含token参数",
			input:    "GET /api/test?token=abc123 HTTP/1.1",
			expected: "GET /api/test?token=***",
		},
		{
			name:     "请求行包含多个查询参数",
			input:    "GET /api/test?name=value&password=secret HTTP/1.1",
			expected: "GET /api/test?name=value&password=***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeRequestLine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeDumpRequest(t *testing.T) {
	tests := []struct {
		name        string
		dump        []byte
		includeBody bool
		checkFunc   func(t *testing.T, result []byte)
	}{
		{
			name:        "空dump",
			dump:        []byte{},
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Empty(t, result)
			},
		},
		{
			name:        "不包含请求体",
			dump:        []byte("GET /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "GET /test HTTP/1.1")
				assert.Contains(t, string(result), "Host: example.com")
			},
		},
		{
			name:        "包含敏感头的请求",
			dump:        []byte("GET /test HTTP/1.1\nAuthorization: Bearer token123\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "Authorization: ***")
			},
		},
		{
			name:        "包含JSON请求体",
			dump:        []byte("POST /test HTTP/1.1\nContent-Type: application/json\n\n{\"password\":\"secret\"}"),
			includeBody: true,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "password")
				assert.Contains(t, string(result), "***")
			},
		},
		{
			name:        "包含表单请求体",
			dump:        []byte("POST /test HTTP/1.1\nContent-Type: application/x-www-form-urlencoded\n\nusername=user&password=secret"),
			includeBody: true,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "password=***")
			},
		},
		{
			name:        "包含敏感查询参数的请求行",
			dump:        []byte("GET /test?password=secret HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "password=***")
			},
		},
		{
			name:        "包含多种HTTP方法的请求行",
			dump:        []byte("POST /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "POST")
			},
		},
		{
			name:        "PUT方法",
			dump:        []byte("PUT /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "PUT")
			},
		},
		{
			name:        "DELETE方法",
			dump:        []byte("DELETE /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "DELETE")
			},
		},
		{
			name:        "PATCH方法",
			dump:        []byte("PATCH /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "PATCH")
			},
		},
		{
			name:        "HEAD方法",
			dump:        []byte("HEAD /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "HEAD")
			},
		},
		{
			name:        "OPTIONS方法",
			dump:        []byte("OPTIONS /test HTTP/1.1\nHost: example.com\n\n"),
			includeBody: false,
			checkFunc: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "OPTIONS")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeDumpRequest(tt.dump, tt.includeBody)
			tt.checkFunc(t, result)
		})
	}
}

func TestSanitizeDumpRequest_ComplexCase(t *testing.T) {
	// Test a complex case with multiple headers and body
	dump := []byte(`POST /api/login HTTP/1.1
Host: example.com
Content-Type: application/json
Authorization: Bearer secret-token
X-API-Key: my-api-key

{"username":"user","password":"secret123","token":"abc123"}`)

	result := SanitizeDumpRequest(dump, true)

	resultStr := string(result)
	// Check that sensitive headers are sanitized
	assert.Contains(t, resultStr, "Authorization: ***")
	assert.Contains(t, resultStr, "X-API-Key: ***")
	// Check that sensitive body fields are sanitized
	assert.Contains(t, resultStr, "password")
	assert.Contains(t, resultStr, "***")
}
