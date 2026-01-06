package tracing

import (
	"context"
	"net/http"

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
}

var (
	// globalConfig 全局追踪配置
	globalConfig TracingConfig
	// tracingEnabled 是否启用追踪
	tracingEnabled bool
)

// Init 初始化追踪系统
func Init(config TracingConfig) {
	globalConfig = config
	tracingEnabled = config.Enabled
}

// IsEnabled 返回是否启用追踪
func IsEnabled() bool {
	return tracingEnabled
}

// StartSpan 开始一个新的追踪 span（如果启用）
func StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if !tracingEnabled {
		return ctx, func() {}
	}
	// 如果未来需要集成 OpenTelemetry，可以在这里添加
	// 目前只返回原始 context 和空函数
	return ctx, func() {}
}

// InjectTraceContext 将追踪上下文注入到 HTTP 请求头中
func InjectTraceContext(ctx context.Context, header http.Header) {
	if !tracingEnabled {
		return
	}

	// 注入请求 ID 到响应头
	requestID := middleware.GetReqID(ctx)
	if requestID != "" {
		header.Set("X-Request-Id", requestID)
	}

	// 如果未来需要集成 OpenTelemetry，可以在这里添加 W3C Trace Context
	// 例如：header.Set("traceparent", traceParent)
}

// ExtractTraceContext 从 HTTP 请求头中提取追踪上下文
func ExtractTraceContext(r *http.Request) context.Context {
	ctx := r.Context()

	// 请求 ID 已经在 middleware.RequestID 中处理
	// 如果未来需要集成 OpenTelemetry，可以在这里提取 W3C Trace Context
	// 例如：traceParent := r.Header.Get("traceparent")

	return ctx
}

// WithTraceContext 为 context 添加追踪信息
func WithTraceContext(ctx context.Context) context.Context {
	if !tracingEnabled {
		return ctx
	}
	// 如果未来需要集成 OpenTelemetry，可以在这里添加 span context
	return ctx
}
