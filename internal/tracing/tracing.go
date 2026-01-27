package tracing

import (
	"context"
	"net/http"

	loggerkit "github.com/soulteary/logger-kit"
	tracingkit "github.com/soulteary/tracing-kit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/soulteary/webhook/internal/middleware"
)

// TracingConfig 配置追踪功能
type TracingConfig struct {
	// Enabled 是否启用追踪
	Enabled bool
	// ServiceName 服务名称
	ServiceName string
	// ServiceVersion 服务版本
	ServiceVersion string
	// OTLPEndpoint OTLP 导出端点（如 localhost:4318）
	OTLPEndpoint string
}

var (
	// globalConfig 全局追踪配置（供测试 resetTracingState 使用）
	globalConfig TracingConfig //nolint:unused // used by tracing_test.resetTracingState
	// tracingEnabled 是否启用追踪
	tracingEnabled bool
)

// Init 初始化追踪系统
// 如果配置了 OTLPEndpoint，将使用 tracing-kit 初始化 OpenTelemetry tracer
func Init(config TracingConfig) error {
	globalConfig = config
	tracingEnabled = config.Enabled

	if !config.Enabled {
		return nil
	}

	// 如果配置了 OTLP 端点，初始化 OpenTelemetry tracer
	if config.OTLPEndpoint != "" {
		_, err := tracingkit.InitTracer(config.ServiceName, config.ServiceVersion, config.OTLPEndpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

// Shutdown 优雅关闭追踪系统
func Shutdown(ctx context.Context) error {
	if !tracingEnabled {
		return nil
	}
	return tracingkit.Shutdown(ctx)
}

// IsEnabled 返回是否启用追踪
func IsEnabled() bool {
	// 检查 tracing-kit 是否真正启用（即 OTLP 端点已配置）
	if tracingkit.IsEnabled() {
		return true
	}
	// 否则返回本地配置的状态
	return tracingEnabled
}

// StartSpan 开始一个新的追踪 span
// 返回带有 span 的 context 和用于结束 span 的函数
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, func()) {
	if !tracingEnabled {
		return ctx, func() {}
	}

	// 使用 tracing-kit 创建 span
	newCtx, span := tracingkit.StartSpan(ctx, name, opts...)
	return newCtx, func() {
		span.End()
	}
}

// StartSpanWithSpan 开始一个新的追踪 span，返回 span 对象
// 用于需要在 span 上设置属性或记录错误的场景
func StartSpanWithSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !tracingEnabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracingkit.StartSpan(ctx, name, opts...)
}

// SetSpanAttributes 在 span 上设置字符串属性
func SetSpanAttributes(span trace.Span, attrs map[string]string) {
	tracingkit.SetSpanAttributes(span, attrs)
}

// SetSpanAttributesFromMap 在 span 上设置混合类型属性
func SetSpanAttributesFromMap(span trace.Span, attrs map[string]interface{}) {
	tracingkit.SetSpanAttributesFromMap(span, attrs)
}

// RecordError 在 span 上记录错误
func RecordError(span trace.Span, err error) {
	tracingkit.RecordError(span, err)
}

// SetSpanStatus 设置 span 状态
func SetSpanStatus(span trace.Span, code codes.Code, description string) {
	tracingkit.SetSpanStatus(span, code, description)
}

// GetSpanFromContext 从 context 获取 span
func GetSpanFromContext(ctx context.Context) trace.Span {
	return tracingkit.GetSpanFromContext(ctx)
}

// InjectTraceContext 将追踪上下文注入到 HTTP 请求头中
func InjectTraceContext(ctx context.Context, header http.Header) {
	if !tracingEnabled {
		return
	}

	// 注入请求 ID 到响应头（优先 logger-kit，否则本包 middleware）
	requestID := loggerkit.RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = middleware.GetReqID(ctx)
	}
	if requestID != "" {
		header.Set("X-Request-Id", requestID)
	}

	// 使用 tracing-kit 注入 W3C Trace Context
	if tracingkit.IsEnabled() {
		headers := make(map[string]string)
		tracingkit.InjectTraceContext(ctx, headers)
		for k, v := range headers {
			header.Set(k, v)
		}
	}
}

// ExtractTraceContext 从 HTTP 请求头中提取追踪上下文
func ExtractTraceContext(r *http.Request) context.Context {
	ctx := r.Context()

	// 使用 tracing-kit 提取 W3C Trace Context
	if tracingkit.IsEnabled() {
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		ctx = tracingkit.ExtractTraceContext(ctx, headers)
	}

	return ctx
}

// WithTraceContext 为 context 添加追踪信息
func WithTraceContext(ctx context.Context) context.Context {
	if !tracingEnabled {
		return ctx
	}
	// 返回带有当前 span 的 context（如果存在）
	return ctx
}
