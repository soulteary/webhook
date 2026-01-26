package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_Default(t *testing.T) {
	cfg := DefaultSecurityConfig()
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// 检查默认安全头
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}
	if got := rr.Header().Get("X-XSS-Protection"); got != "1; mode=block" {
		t.Errorf("X-XSS-Protection = %q, want %q", got, "1; mode=block")
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q, want %q", got, "strict-origin-when-cross-origin")
	}
}

func TestSecurityHeaders_Strict(t *testing.T) {
	cfg := StrictSecurityConfig()
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// 检查严格模式的安全头
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := rr.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Content-Security-Policy should be set in strict mode")
	}
	if got := rr.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("Strict-Transport-Security should be set in strict mode with HSTS enabled")
	}
	if got := rr.Header().Get("Cross-Origin-Resource-Policy"); got != "same-origin" {
		t.Errorf("Cross-Origin-Resource-Policy = %q, want %q", got, "same-origin")
	}
}

func TestSecurityHeaders_Disabled(t *testing.T) {
	cfg := SecurityConfig{Enabled: false}
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// 禁用时不应设置安全头
	if got := rr.Header().Get("X-Content-Type-Options"); got != "" {
		t.Errorf("X-Content-Type-Options should be empty when disabled, got %q", got)
	}
}

func TestSecurityHeaders_CustomHSTS(t *testing.T) {
	cfg := SecurityConfig{
		Enabled:               true,
		HSTS:                  true,
		HSTSMaxAge:            86400, // 1 天
		HSTSIncludeSubDomains: true,
	}
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := "max-age=86400; includeSubDomains"
	if got := rr.Header().Get("Strict-Transport-Security"); got != expected {
		t.Errorf("Strict-Transport-Security = %q, want %q", got, expected)
	}
}

func TestSecurityHeaders_CustomHeaders(t *testing.T) {
	cfg := SecurityConfig{
		Enabled: true,
		CustomHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Custom-Header"); got != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", got, "custom-value")
	}
}

func TestNoCacheHeaders(t *testing.T) {
	handler := NoCacheHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate, proxy-revalidate" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store, no-cache, must-revalidate, proxy-revalidate")
	}
	if got := rr.Header().Get("Pragma"); got != "no-cache" {
		t.Errorf("Pragma = %q, want %q", got, "no-cache")
	}
	if got := rr.Header().Get("Expires"); got != "0" {
		t.Errorf("Expires = %q, want %q", got, "0")
	}
}

func TestBodyLimit(t *testing.T) {
	// 创建一个限制为 10 字节的中间件
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		method         string
		contentLength  int64
		expectedStatus int
	}{
		{
			name:           "GET request should pass",
			method:         http.MethodGet,
			contentLength:  0,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST with small body should pass",
			method:         http.MethodPost,
			contentLength:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST with large body should be rejected",
			method:         http.MethodPost,
			contentLength:  100,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.ContentLength = tt.contentLength
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"john.doe@example.com", "jo***@example.com"},
		{"a@b.com", "a***@b.com"},
		{"ab@c.com", "ab***@c.com"},
		{"invalid", "***"},
		{"", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := MaskEmail(tt.input); got != tt.expected {
				t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+1234567890", "+12***7890"},
		{"12345678901", "123***8901"},
		{"12345", "***"},
		{"", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := MaskPhone(tt.input); got != tt.expected {
				t.Errorf("MaskPhone(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
		{31536000, "31536000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := itoa(tt.input); got != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
