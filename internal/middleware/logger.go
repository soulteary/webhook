package middleware

import (
	"net/http"

	loggerkit "github.com/soulteary/logger-kit"
	"github.com/soulteary/webhook/internal/logger"
)

// NewLogger creates a new logging middleware using logger-kit.
func NewLogger() func(next http.Handler) http.Handler {
	if logger.DefaultLogger == nil {
		// 如果 logger 未初始化，使用默认配置
		_ = logger.Init(true, false, "", false)
	}

	cfg := loggerkit.DefaultMiddlewareConfig()
	cfg.Logger = logger.DefaultLogger
	cfg.IncludeRequestID = true
	cfg.IncludeLatency = true
	cfg.SkipPaths = []string{"/health", "/livez", "/readyz", "/metrics"}

	return loggerkit.Middleware(cfg)
}
