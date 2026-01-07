package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/webhook/internal/middleware"
)

func TestInit(t *testing.T) {
	config := TracingConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	Init(config)

	if !IsEnabled() {
		t.Error("IsEnabled() should return true after Init with Enabled=true")
	}

	// 测试禁用追踪
	config.Enabled = false
	Init(config)

	if IsEnabled() {
		t.Error("IsEnabled() should return false after Init with Enabled=false")
	}
}

func TestIsEnabled(t *testing.T) {
	// 测试默认状态（未初始化）
	Init(TracingConfig{Enabled: false})
	if IsEnabled() {
		t.Error("IsEnabled() should return false when not enabled")
	}

	Init(TracingConfig{Enabled: true})
	if !IsEnabled() {
		t.Error("IsEnabled() should return true when enabled")
	}
}

func TestStartSpan(t *testing.T) {
	Init(TracingConfig{Enabled: true})

	ctx := context.Background()
	newCtx, finish := StartSpan(ctx, "test-span")

	if newCtx == nil {
		t.Error("StartSpan() should return a context")
	}

	// finish 函数应该可以安全调用
	finish()

	// 测试禁用追踪时
	Init(TracingConfig{Enabled: false})
	ctx2 := context.Background()
	newCtx2, finish2 := StartSpan(ctx2, "test-span")

	if newCtx2 != ctx2 {
		t.Error("StartSpan() should return original context when tracing is disabled")
	}

	finish2() // 应该不会 panic
}

func TestInjectTraceContext(t *testing.T) {
	Init(TracingConfig{Enabled: true})

	// 创建带有请求 ID 的 context
	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "test-request-id")
	header := make(http.Header)

	InjectTraceContext(ctx, header)

	requestID := header.Get("X-Request-Id")
	if requestID != "test-request-id" {
		t.Errorf("InjectTraceContext() should inject request ID, got: %s", requestID)
	}

	// 测试禁用追踪时
	Init(TracingConfig{Enabled: false})
	header2 := make(http.Header)
	InjectTraceContext(ctx, header2)

	if header2.Get("X-Request-Id") != "" {
		t.Error("InjectTraceContext() should not inject when tracing is disabled")
	}

	// 测试没有请求 ID 的情况
	ctx3 := context.Background()
	header3 := make(http.Header)
	Init(TracingConfig{Enabled: true})
	InjectTraceContext(ctx3, header3)

	// 应该不会 panic，但可能没有请求 ID
}

func TestExtractTraceContext(t *testing.T) {
	Init(TracingConfig{Enabled: true})

	// 创建测试请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-Id", "test-request-id")

	ctx := ExtractTraceContext(req)

	if ctx == nil {
		t.Error("ExtractTraceContext() should return a context")
	}

	// 测试禁用追踪时
	Init(TracingConfig{Enabled: false})
	ctx2 := ExtractTraceContext(req)

	if ctx2 == nil {
		t.Error("ExtractTraceContext() should return a context even when tracing is disabled")
	}
}

func TestWithTraceContext(t *testing.T) {
	Init(TracingConfig{Enabled: true})

	ctx := context.Background()
	newCtx := WithTraceContext(ctx)

	if newCtx == nil {
		t.Error("WithTraceContext() should return a context")
	}

	// 测试禁用追踪时
	Init(TracingConfig{Enabled: false})
	ctx2 := context.Background()
	newCtx2 := WithTraceContext(ctx2)

	if newCtx2 != ctx2 {
		t.Error("WithTraceContext() should return original context when tracing is disabled")
	}
}

func TestTracingWithMiddleware(t *testing.T) {
	Init(TracingConfig{Enabled: true})

	// 创建带有请求 ID 中间件的请求
	handler := middleware.RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := middleware.GetReqID(ctx)

		// 注入追踪上下文
		header := w.Header()
		InjectTraceContext(ctx, header)

		if header.Get("X-Request-Id") != requestID {
			t.Error("Request ID should be injected into headers")
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证响应头包含请求 ID
	if w.Header().Get("X-Request-Id") == "" {
		t.Error("Response header should contain X-Request-Id")
	}
}

func TestTracingDisabled(t *testing.T) {
	Init(TracingConfig{Enabled: false})

	ctx := context.Background()

	// 所有函数在禁用时应该安全返回
	_, finish := StartSpan(ctx, "test")
	finish()

	header := make(http.Header)
	InjectTraceContext(ctx, header)

	newCtx := WithTraceContext(ctx)
	if newCtx != ctx {
		t.Error("WithTraceContext() should return original context when disabled")
	}
}
