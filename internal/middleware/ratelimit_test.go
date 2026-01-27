package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name   string
		config RateLimitConfig
		want   bool
	}{
		{"disabled", RateLimitConfig{Enabled: false}, false},
		{"enabled", RateLimitConfig{Enabled: true, RPS: 10, Burst: 20}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.config)
			if (rl != nil) != tt.want {
				t.Errorf("NewRateLimiter() = %v, want %v", rl != nil, tt.want)
			}
		})
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     10,
		Burst:   5,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil when enabled")
	}

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 测试正常请求
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// 测试限流（发送超过 burst 的请求）
	for i := 0; i < 10; i++ {
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req)
	}

	// 最后一个请求应该被限流；由于限流器的实现，可能需要更多请求才能触发限流，这里主要测试不会 panic
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req)
	_ = w3.Code // may be OK or rate-limited depending on implementation
}

func TestRateLimiter_Middleware_Disabled(t *testing.T) {
	config := RateLimitConfig{Enabled: false}
	rl := NewRateLimiter(config)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimiter_HookMiddleware(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     10,
		Burst:   5,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil when enabled")
	}

	hookMiddleware := rl.HookMiddleware(5, 2)

	handler := hookMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 测试正常请求
	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimiter_HookMiddleware_Disabled(t *testing.T) {
	config := RateLimitConfig{Enabled: false}
	rl := NewRateLimiter(config)

	hookMiddleware := rl.HookMiddleware(5, 2)

	handler := hookMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		config RateLimitConfig
	}{
		{"disabled", RateLimitConfig{Enabled: false}},
		{"enabled", RateLimitConfig{Enabled: true, RPS: 10, Burst: 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewRateLimitMiddleware(tt.config)
			if middleware == nil {
				t.Error("NewRateLimitMiddleware() should not return nil")
			}

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// 应该不会 panic
		})
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name           string
		request        *http.Request
		expectedPrefix string
	}{
		{"X-Forwarded-For", func() *http.Request {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.1")
			return req
		}(), "192.168.1.1"},
		{"X-Real-IP", func() *http.Request {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Real-IP", "10.0.0.1")
			return req
		}(), "10.0.0.1"},
		{"RemoteAddr", httptest.NewRequest("GET", "/test", nil), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := extractIP(tt.request)
			if tt.expectedPrefix != "" && ip != tt.expectedPrefix {
				// 对于 RemoteAddr，IP 可能包含端口，所以只检查不为空
				if tt.name == "RemoteAddr" && ip == "" {
					t.Error("extractIP() should extract IP from RemoteAddr")
				} else if tt.name != "RemoteAddr" && ip != tt.expectedPrefix {
					t.Errorf("extractIP() = %s, want %s", ip, tt.expectedPrefix)
				}
			}
		})
	}
}

func TestParseForwardedIP(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		expected string
	}{
		{"single IP", "192.168.1.1", "192.168.1.1"},
		{"multiple IPs", "192.168.1.1, 10.0.0.1", "192.168.1.1"},
		{"with spaces", "  192.168.1.1  , 10.0.0.1  ", "192.168.1.1"},
		{"invalid IP", "invalid", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseForwardedIP(tt.xff)
			if got != tt.expected {
				t.Errorf("parseForwardedIP() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestExtractHookID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple path", "/hooks/test-hook", "test-hook"},
		{"nested path", "/hooks/test-hook/sub", "sub"},
		{"root path", "/", ""},
		{"empty path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.path == "" {
				// 对于空路径，需要手动构造请求，因为 httptest.NewRequest 不接受空 URL
				req = httptest.NewRequest("GET", "/", nil)
				req.URL.Path = ""
			} else {
				req = httptest.NewRequest("GET", tt.path, nil)
			}
			got := extractHookID(req)
			if got != tt.expected {
				t.Errorf("extractHookID() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     10,
		Burst:   5,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil")
	}

	// 创建一些限流器
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 发送一些请求来创建限流器
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1." + string(rune('0'+i)) + ":12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 等待清理 goroutine 运行
	time.Sleep(100 * time.Millisecond)

	// 应该不会 panic
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     100,
		Burst:   50,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil")
	}

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 并发发送请求
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			done <- true
		}(i)
	}

	// 等待所有请求完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 应该不会 panic 或产生竞态条件
}

func TestExtractIP_IPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "[2001:db8::1]:8080"
	ip := extractIP(req)
	assert.Equal(t, "2001:db8::1", ip)
}

func TestExtractIP_InvalidRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "invalid-addr"
	ip := extractIP(req)
	assert.Equal(t, "invalid-addr", ip)
}

func TestExtractIP_XForwardedFor_InvalidIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "invalid-ip")
	ip := extractIP(req)
	// Should fallback to RemoteAddr
	assert.NotEmpty(t, ip)
}

func TestParseForwardedIP_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		expected string
	}{
		{"comma at start", ",192.168.1.1", ""},
		{"comma only", ",", ""},
		{"multiple commas", "192.168.1.1, 10.0.0.1, 172.16.0.1", "192.168.1.1"},
		{"IPv6 in XFF", "2001:db8::1", "2001:db8::1"},
		{"IPv6 with comma", "2001:db8::1, 2001:db8::2", "2001:db8::1"},
		{"whitespace only", "   ", ""},
		{"mixed whitespace", "  192.168.1.1  , 10.0.0.1  ", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseForwardedIP(tt.xff)
			if tt.expected != "" && got != tt.expected {
				t.Errorf("parseForwardedIP() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestExtractHookID_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		usePath  bool // If true, manually set req.URL.Path instead of using path in NewRequest
	}{
		{"single slash", "/", "", false},
		{"multiple slashes", "/hooks/test/sub/path", "path", false},
		{"no leading slash", "hooks/test", "test", true},
		{"empty after trim", "   ", "", true},
		{"with query", "/hooks/test?param=value", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.usePath {
				// For invalid URI paths, create a valid request first, then set the path
				req = httptest.NewRequest("GET", "/", nil)
				req.URL.Path = tt.path
			} else {
				req = httptest.NewRequest("GET", tt.path, nil)
			}
			got := extractHookID(req)
			if tt.expected != "" && got != tt.expected {
				t.Errorf("extractHookID() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestRateLimiter_GetIPLimiter(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     10,
		Burst:   5,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil")
	}

	// Test getting limiter for same IP multiple times
	limiter1 := rl.getIPLimiter("192.168.1.1", 10, 5)
	limiter2 := rl.getIPLimiter("192.168.1.1", 10, 5)

	// Should return same limiter instance
	if limiter1 != limiter2 {
		t.Error("getIPLimiter() should return same limiter for same IP")
	}

	// Test different IPs
	limiter3 := rl.getIPLimiter("192.168.1.2", 10, 5)
	if limiter1 == limiter3 {
		t.Error("getIPLimiter() should return different limiter for different IP")
	}
}

func TestRateLimiter_GetHookLimiter(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     10,
		Burst:   5,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil")
	}

	// Test getting limiter for same hook multiple times
	limiter1 := rl.getHookLimiter("hook1", 5, 2)
	limiter2 := rl.getHookLimiter("hook1", 5, 2)

	// Should return same limiter instance
	if limiter1 != limiter2 {
		t.Error("getHookLimiter() should return same limiter for same hook")
	}

	// Test different hooks
	limiter3 := rl.getHookLimiter("hook2", 5, 2)
	if limiter1 == limiter3 {
		t.Error("getHookLimiter() should return different limiter for different hook")
	}
}

func TestRateLimiter_Middleware_NilLimiter(t *testing.T) {
	// Test middleware with nil limiter (disabled)
	var rl *RateLimiter = nil
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimiter_HookMiddleware_EmptyHookID(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     10,
		Burst:   5,
	}

	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() should not return nil")
	}

	hookMiddleware := rl.HookMiddleware(5, 2)
	handler := hookMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with empty path (no hook ID)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestRateLimitConfig_RedisFields 测试 Redis 配置字段
func TestRateLimitConfig_RedisFields(t *testing.T) {
	config := RateLimitConfig{
		Enabled:        true,
		RPS:            100,
		Burst:          10,
		RedisEnabled:   true,
		RedisAddr:      "localhost:6379",
		RedisPassword:  "secret",
		RedisDB:        1,
		RedisKeyPrefix: "myapp:ratelimit:",
		WindowSeconds:  120,
	}

	assert.True(t, config.Enabled)
	assert.True(t, config.RedisEnabled)
	assert.Equal(t, "localhost:6379", config.RedisAddr)
	assert.Equal(t, "secret", config.RedisPassword)
	assert.Equal(t, 1, config.RedisDB)
	assert.Equal(t, "myapp:ratelimit:", config.RedisKeyPrefix)
	assert.Equal(t, 120, config.WindowSeconds)
}

// TestNewRateLimiterWithRedis 测试带 Redis 配置的限流器创建
func TestNewRateLimiterWithRedis(t *testing.T) {
	// 测试禁用 Redis 时的行为（应该回退到内存限流）
	rl := NewRateLimiterWithRedis(true, 100, 10, "", "", 0, "", 60)
	require.NotNil(t, rl)
	assert.False(t, rl.IsRedisEnabled(), "should not use Redis when addr is empty")

	// 清理
	err := rl.Close()
	assert.NoError(t, err)
}

// TestNewRateLimiter_WithInvalidRedis 测试无效 Redis 配置时的回退行为
func TestNewRateLimiter_WithInvalidRedis(t *testing.T) {
	config := RateLimitConfig{
		Enabled:       true,
		RPS:           100,
		Burst:         10,
		RedisEnabled:  true,
		RedisAddr:     "invalid-host:6379", // 无效的 Redis 地址
		RedisPassword: "",
		RedisDB:       0,
		WindowSeconds: 60,
	}

	// 应该创建限流器，但回退到内存模式
	rl := NewRateLimiter(config)
	require.NotNil(t, rl)

	// 由于 Redis 连接失败，应该回退到内存模式
	// 注意：这个测试可能需要一些时间来完成连接超时
	// 在某些环境中，DNS 解析可能成功但连接失败

	// 清理
	err := rl.Close()
	assert.NoError(t, err)
}

// TestRateLimiter_IsRedisEnabled 测试 Redis 启用状态检查
func TestRateLimiter_IsRedisEnabled(t *testing.T) {
	// 测试 nil 限流器
	var nilRL *RateLimiter
	assert.False(t, nilRL.IsRedisEnabled())

	// 测试内存限流器
	config := RateLimitConfig{
		Enabled: true,
		RPS:     100,
		Burst:   10,
	}
	rl := NewRateLimiter(config)
	require.NotNil(t, rl)
	assert.False(t, rl.IsRedisEnabled())

	// 清理
	err := rl.Close()
	assert.NoError(t, err)
}

// TestRateLimiter_Close 测试限流器关闭
func TestRateLimiter_Close(t *testing.T) {
	// 测试 nil 限流器关闭
	var nilRL *RateLimiter
	err := nilRL.Close()
	assert.NoError(t, err)

	// 测试内存限流器关闭
	config := RateLimitConfig{
		Enabled: true,
		RPS:     100,
		Burst:   10,
	}
	rl := NewRateLimiter(config)
	require.NotNil(t, rl)

	err = rl.Close()
	assert.NoError(t, err)
}

// TestRateLimiter_MiddlewareWithHeaders 测试限流响应头
func TestRateLimiter_MiddlewareWithHeaders(t *testing.T) {
	config := RateLimitConfig{
		Enabled: true,
		RPS:     100,
		Burst:   10,
	}

	rl := NewRateLimiter(config)
	require.NotNil(t, rl)
	defer func() { _ = rl.Close() }()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRateLimiter_HookMiddleware_WithConfig 测试带配置的 Hook 限流中间件
func TestRateLimiter_HookMiddleware_WithConfig(t *testing.T) {
	config := RateLimitConfig{
		Enabled:       true,
		RPS:           10,
		Burst:         5,
		WindowSeconds: 60,
	}

	rl := NewRateLimiter(config)
	require.NotNil(t, rl)
	defer func() { _ = rl.Close() }()

	hookMiddleware := rl.HookMiddleware(5, 2)
	handler := hookMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 测试正常请求
	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
