package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/soulteary/webhook/internal/logger"
	"golang.org/x/time/rate"
)

// RateLimiter 提供基于 IP 和基于 hook 的限流功能
type RateLimiter struct {
	// 全局限流器（基于 IP）
	globalLimiter *rate.Limiter

	// 基于 IP 的限流器映射
	ipLimiters map[string]*rate.Limiter

	// 基于 hook 的限流器映射
	hookLimiters map[string]*rate.Limiter

	// 保护并发访问的互斥锁
	mu sync.RWMutex

	// 清理过期限流器的定时器
	cleanupInterval time.Duration

	// 限流器过期时间
	limiterTTL time.Duration
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled bool // 是否启用限流
	RPS     int  // 每秒请求数
	Burst   int  // 突发请求数
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if !config.Enabled {
		return nil
	}

	rl := &RateLimiter{
		globalLimiter:   rate.NewLimiter(rate.Limit(config.RPS), config.Burst),
		ipLimiters:      make(map[string]*rate.Limiter),
		hookLimiters:    make(map[string]*rate.Limiter),
		cleanupInterval: 5 * time.Minute,
		limiterTTL:      10 * time.Minute,
	}

	// 启动清理 goroutine
	go rl.cleanup()

	return rl
}

// cleanup 定期清理过期的限流器（简化版本，实际可以基于最后使用时间）
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// 简化清理：如果限流器数量过多，清理一部分
		// 实际实现可以基于最后使用时间
		if len(rl.ipLimiters) > 1000 {
			// 清理一半的限流器（简单的策略）
			count := 0
			for ip := range rl.ipLimiters {
				if count >= len(rl.ipLimiters)/2 {
					break
				}
				delete(rl.ipLimiters, ip)
				count++
			}
		}
		if len(rl.hookLimiters) > 1000 {
			count := 0
			for hookID := range rl.hookLimiters {
				if count >= len(rl.hookLimiters)/2 {
					break
				}
				delete(rl.hookLimiters, hookID)
				count++
			}
		}
		rl.mu.Unlock()
	}
}

// getIPLimiter 获取或创建基于 IP 的限流器
func (rl *RateLimiter) getIPLimiter(ip string, rps int, burst int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.ipLimiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rps), burst)
		rl.ipLimiters[ip] = limiter
	}
	return limiter
}

// getHookLimiter 获取或创建基于 hook 的限流器
func (rl *RateLimiter) getHookLimiter(hookID string, rps int, burst int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.hookLimiters[hookID]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rps), burst)
		rl.hookLimiters[hookID] = limiter
	}
	return limiter
}

// extractIP 从请求中提取客户端 IP
func extractIP(r *http.Request) string {
	// 优先检查 X-Forwarded-For 头（适用于反向代理场景）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For 可能包含多个 IP，取第一个
		if ip := parseForwardedIP(xff); ip != "" {
			return ip
		}
	}

	// 检查 X-Real-IP 头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// parseForwardedIP 解析 X-Forwarded-For 头中的 IP
func parseForwardedIP(xff string) string {
	// X-Forwarded-For 格式: "client, proxy1, proxy2"
	// 取第一个 IP（去除空格）
	xff = strings.TrimSpace(xff)
	if xff == "" {
		return ""
	}

	// 查找第一个逗号
	idx := strings.Index(xff, ",")
	if idx > 0 {
		ip := strings.TrimSpace(xff[:idx])
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			return parsedIP.String()
		}
	}

	// 没有逗号，整个字符串就是 IP
	parsedIP := net.ParseIP(xff)
	if parsedIP != nil {
		return parsedIP.String()
	}
	return ""
}

// extractHookID 从请求中提取 hook ID
// 注意：这个函数需要在路由匹配之后调用才能获取到 hook ID
// 在中间件中，我们无法直接访问 chi.URLParam，所以这个函数主要用于 HookMiddleware
func extractHookID(r *http.Request) string {
	// 尝试从 URL 路径中提取
	// 实际使用中，hook ID 应该在 handler 中通过 chi.URLParam(r, "id") 获取
	// 这里提供一个简单的实现作为后备
	path := r.URL.Path
	if path == "" || path == "/" {
		return ""
	}
	// 移除前导斜杠并获取最后一部分
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Middleware 返回限流中间件
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	if rl == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 先检查全局限流
		if !rl.globalLimiter.Allow() {
			requestID := GetReqID(r.Context())
			logger.Warnf("[%s] global rate limit exceeded from %s", requestID, extractIP(r))
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// 检查基于 IP 的限流
		ip := extractIP(r)
		// 获取全局限流器的配置
		globalLimit := rl.globalLimiter.Limit()
		globalBurst := rl.globalLimiter.Burst()
		// rate.Limit 是 float64 类型，转换为 int
		ipLimiter := rl.getIPLimiter(ip, int(globalLimit), globalBurst)
		if !ipLimiter.Allow() {
			requestID := GetReqID(r.Context())
			logger.Warnf("[%s] IP rate limit exceeded for %s", requestID, ip)
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// HookMiddleware 返回基于 hook 的限流中间件
// 这个中间件需要在知道 hook ID 之后使用
func (rl *RateLimiter) HookMiddleware(rps int, burst int) func(next http.Handler) http.Handler {
	if rl == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从请求上下文中获取 hook ID（需要在 handler 中设置）
			// 或者从 URL 路径中提取
			hookID := extractHookID(r)

			if hookID != "" {
				hookLimiter := rl.getHookLimiter(hookID, rps, burst)
				if !hookLimiter.Allow() {
					requestID := GetReqID(r.Context())
					logger.Warnf("[%s] hook rate limit exceeded for hook %s", requestID, hookID)
					w.Header().Set("Retry-After", "1")
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewRateLimitMiddleware 创建限流中间件（简化版本，仅基于 IP）
func NewRateLimitMiddleware(config RateLimitConfig) func(next http.Handler) http.Handler {
	rl := NewRateLimiter(config)
	if rl == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return rl.Middleware
}
