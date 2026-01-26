package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tracingkit "github.com/soulteary/tracing-kit"
	"go.opentelemetry.io/otel/codes"

	"github.com/soulteary/webhook/internal/middleware"
)

// resetTracingState 重置追踪状态，用于测试之间的隔离
func resetTracingState() {
	tracingEnabled = false
	globalConfig = TracingConfig{}
	tracingkit.TeardownTestTracer()
}

func TestInit(t *testing.T) {
	defer resetTracingState()

	config := TracingConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	err := Init(config)
	if err != nil {
		t.Errorf("Init() should not return error: %v", err)
	}

	if !IsEnabled() {
		t.Error("IsEnabled() should return true after Init with Enabled=true")
	}

	// 测试禁用追踪
	resetTracingState()
	config.Enabled = false
	err = Init(config)
	if err != nil {
		t.Errorf("Init() should not return error: %v", err)
	}

	if IsEnabled() {
		t.Error("IsEnabled() should return false after Init with Enabled=false")
	}
}

func TestInitWithOTLPEndpoint(t *testing.T) {
	defer resetTracingState()

	// 测试带有 OTLP 端点的初始化（使用无效端点，不会实际连接）
	config := TracingConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4318",
	}

	err := Init(config)
	if err != nil {
		t.Errorf("Init() with OTLP endpoint should not return error: %v", err)
	}

	// tracing-kit 应该被启用
	if !tracingkit.IsEnabled() {
		t.Error("tracing-kit should be enabled after Init with OTLPEndpoint")
	}
}

func TestIsEnabled(t *testing.T) {
	defer resetTracingState()

	// 测试默认状态（未初始化）
	resetTracingState()
	_ = Init(TracingConfig{Enabled: false})
	if IsEnabled() {
		t.Error("IsEnabled() should return false when not enabled")
	}

	resetTracingState()
	_ = Init(TracingConfig{Enabled: true})
	if !IsEnabled() {
		t.Error("IsEnabled() should return true when enabled")
	}
}

func TestStartSpan(t *testing.T) {
	defer resetTracingState()

	_ = Init(TracingConfig{Enabled: true})

	ctx := context.Background()
	newCtx, finish := StartSpan(ctx, "test-span")

	if newCtx == nil {
		t.Error("StartSpan() should return a context")
	}

	// finish 函数应该可以安全调用
	finish()

	// 测试禁用追踪时
	resetTracingState()
	_ = Init(TracingConfig{Enabled: false})
	ctx2 := context.Background()
	newCtx2, finish2 := StartSpan(ctx2, "test-span")

	if newCtx2 != ctx2 {
		t.Error("StartSpan() should return original context when tracing is disabled")
	}

	finish2() // 应该不会 panic
}

func TestStartSpanWithSpan(t *testing.T) {
	defer resetTracingState()

	// 使用 test tracer 设置
	tp, _ := tracingkit.SetupTestTracer(t)
	defer tracingkit.ShutdownTracerProvider(tp)

	tracingEnabled = true
	globalConfig = TracingConfig{Enabled: true}

	ctx := context.Background()
	newCtx, span := StartSpanWithSpan(ctx, "test-span")

	if newCtx == nil {
		t.Error("StartSpanWithSpan() should return a context")
	}

	if span == nil {
		t.Error("StartSpanWithSpan() should return a span")
	}

	// 测试设置属性
	SetSpanAttributes(span, map[string]string{
		"test.key": "test-value",
	})

	SetSpanAttributesFromMap(span, map[string]interface{}{
		"test.int":    42,
		"test.bool":   true,
		"test.float":  3.14,
		"test.string": "hello",
	})

	// 测试设置状态
	SetSpanStatus(span, codes.Ok, "success")

	span.End()
}

func TestRecordError(t *testing.T) {
	defer resetTracingState()

	// 使用 test tracer 设置
	tp, _ := tracingkit.SetupTestTracer(t)
	defer tracingkit.ShutdownTracerProvider(tp)

	tracingEnabled = true

	ctx := context.Background()
	_, span := StartSpanWithSpan(ctx, "test-span")

	// 测试记录错误
	testErr := context.DeadlineExceeded
	RecordError(span, testErr)

	span.End()
}

func TestGetSpanFromContext(t *testing.T) {
	defer resetTracingState()

	// 使用 test tracer 设置
	tp, _ := tracingkit.SetupTestTracer(t)
	defer tracingkit.ShutdownTracerProvider(tp)

	tracingEnabled = true

	ctx := context.Background()
	newCtx, span := StartSpanWithSpan(ctx, "test-span")

	// 从 context 获取 span
	retrievedSpan := GetSpanFromContext(newCtx)
	if retrievedSpan == nil {
		t.Error("GetSpanFromContext() should return a span")
	}

	span.End()
}

func TestInjectTraceContext(t *testing.T) {
	defer resetTracingState()

	_ = Init(TracingConfig{Enabled: true})

	// 创建带有请求 ID 的 context
	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "test-request-id")
	header := make(http.Header)

	InjectTraceContext(ctx, header)

	requestID := header.Get("X-Request-Id")
	if requestID != "test-request-id" {
		t.Errorf("InjectTraceContext() should inject request ID, got: %s", requestID)
	}

	// 测试禁用追踪时
	resetTracingState()
	_ = Init(TracingConfig{Enabled: false})
	header2 := make(http.Header)
	InjectTraceContext(ctx, header2)

	if header2.Get("X-Request-Id") != "" {
		t.Error("InjectTraceContext() should not inject when tracing is disabled")
	}

	// 测试没有请求 ID 的情况
	ctx3 := context.Background()
	header3 := make(http.Header)
	resetTracingState()
	_ = Init(TracingConfig{Enabled: true})
	InjectTraceContext(ctx3, header3)

	// 应该不会 panic，但可能没有请求 ID
}

func TestInjectTraceContextWithOTLP(t *testing.T) {
	defer resetTracingState()

	// 使用 test tracer 设置
	tp, _ := tracingkit.SetupTestTracer(t)
	defer tracingkit.ShutdownTracerProvider(tp)

	tracingEnabled = true

	// 创建带有 span 的 context
	ctx, span := tracingkit.StartSpan(context.Background(), "test-span")
	defer span.End()

	header := make(http.Header)
	InjectTraceContext(ctx, header)

	// 应该包含 traceparent 头
	if header.Get("Traceparent") == "" {
		t.Error("InjectTraceContext() should inject traceparent header when OTLP is enabled")
	}
}

func TestExtractTraceContext(t *testing.T) {
	defer resetTracingState()

	_ = Init(TracingConfig{Enabled: true})

	// 创建测试请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-Id", "test-request-id")

	ctx := ExtractTraceContext(req)

	if ctx == nil {
		t.Error("ExtractTraceContext() should return a context")
	}

	// 测试禁用追踪时
	resetTracingState()
	_ = Init(TracingConfig{Enabled: false})
	ctx2 := ExtractTraceContext(req)

	if ctx2 == nil {
		t.Error("ExtractTraceContext() should return a context even when tracing is disabled")
	}
}

func TestExtractTraceContextWithOTLP(t *testing.T) {
	defer resetTracingState()

	// 使用 test tracer 设置
	tp, _ := tracingkit.SetupTestTracer(t)
	defer tracingkit.ShutdownTracerProvider(tp)

	tracingEnabled = true

	// 创建带有 traceparent 头的请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")

	ctx := ExtractTraceContext(req)

	if ctx == nil {
		t.Error("ExtractTraceContext() should return a context")
	}

	// 验证 span context 被正确提取
	span := GetSpanFromContext(ctx)
	if span == nil {
		t.Error("ExtractTraceContext() should extract span context from traceparent header")
	}
}

func TestWithTraceContext(t *testing.T) {
	defer resetTracingState()

	_ = Init(TracingConfig{Enabled: true})

	ctx := context.Background()
	newCtx := WithTraceContext(ctx)

	if newCtx == nil {
		t.Error("WithTraceContext() should return a context")
	}

	// 测试禁用追踪时
	resetTracingState()
	_ = Init(TracingConfig{Enabled: false})
	ctx2 := context.Background()
	newCtx2 := WithTraceContext(ctx2)

	if newCtx2 != ctx2 {
		t.Error("WithTraceContext() should return original context when tracing is disabled")
	}
}

func TestShutdown(t *testing.T) {
	defer resetTracingState()

	// 测试未初始化时关闭
	err := Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() should not return error when not initialized: %v", err)
	}

	// 测试初始化后关闭
	_ = Init(TracingConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4318",
	})

	err = Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() should not return error: %v", err)
	}
}

func TestTracingWithMiddleware(t *testing.T) {
	defer resetTracingState()

	_ = Init(TracingConfig{Enabled: true})

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
	defer resetTracingState()

	_ = Init(TracingConfig{Enabled: false})

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
