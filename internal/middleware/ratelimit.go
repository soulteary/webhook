package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	rediskit "github.com/soulteary/redis-kit/client"
	redisratelimit "github.com/soulteary/redis-kit/ratelimit"
	"github.com/soulteary/webhook/internal/logger"
	"golang.org/x/time/rate"
)

// RateLimiter 提供基于 IP 和基于 hook 的限流功能
// 支持内存限流（默认）和 Redis 分布式限流
type RateLimiter struct {
	// 全局限流器（基于 IP）- 内存模式
	globalLimiter *rate.Limiter

	// 基于 IP 的限流器映射 - 内存模式
	ipLimiters map[string]*rate.Limiter

	// 基于 hook 的限流器映射 - 内存模式
	hookLimiters map[string]*rate.Limiter

	// 保护并发访问的互斥锁
	mu sync.RWMutex

	// 清理过期限流器的定时器
	cleanupInterval time.Duration

	// 限流器过期时间
	limiterTTL time.Duration

	// Redis 分布式限流支持
	redisClient  *redis.Client
	redisLimiter *redisratelimit.RateLimiter
	useRedis     bool

	// 配置
	config RateLimitConfig
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled bool // 是否启用限流
	RPS     int  // 每秒请求数
	Burst   int  // 突发请求数

	// Redis 分布式限流配置
	RedisEnabled   bool   // 是否启用 Redis 限流
	RedisAddr      string // Redis 服务器地址
	RedisPassword  string // Redis 密码
	RedisDB        int    // Redis 数据库索引
	RedisKeyPrefix string // Redis 键前缀
	WindowSeconds  int    // 限流时间窗口（秒）
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
		config:          config,
	}

	// 尝试初始化 Redis 限流
	if config.RedisEnabled {
		if err := rl.initRedis(); err != nil {
			logger.Warnf("failed to initialize Redis rate limiter, falling back to in-memory: %v", err)
		} else {
			logger.Info("Redis distributed rate limiter initialized successfully")
		}
	}

	// 如果不使用 Redis，启动内存清理 goroutine
	if !rl.useRedis {
		go rl.cleanup()
	}

	return rl
}

// initRedis 初始化 Redis 连接
func (rl *RateLimiter) initRedis() error {
	cfg := rediskit.DefaultConfig().
		WithAddr(rl.config.RedisAddr).
		WithPassword(rl.config.RedisPassword).
		WithDB(rl.config.RedisDB)

	client, err := rediskit.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// 设置键前缀，如果未配置则使用默认值
	keyPrefix := rl.config.RedisKeyPrefix
	if keyPrefix == "" {
		keyPrefix = "webhook:ratelimit:"
	}

	rl.redisClient = client
	rl.redisLimiter = redisratelimit.NewRateLimiterWithPrefixes(client, keyPrefix, keyPrefix+"cooldown:")
	rl.useRedis = true

	return nil
}

// Close 关闭限流器，释放 Redis 连接
func (rl *RateLimiter) Close() error {
	if rl == nil {
		return nil
	}
	if rl.redisClient != nil {
		return rl.redisClient.Close()
	}
	return nil
}

// IsRedisEnabled 返回是否启用了 Redis 限流
func (rl *RateLimiter) IsRedisEnabled() bool {
	if rl == nil {
		return false
	}
	return rl.useRedis
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

// checkRedisLimit 使用 Redis 检查限流
func (rl *RateLimiter) checkRedisLimit(ctx context.Context, key string, limit int) (bool, int, time.Duration) {
	window := time.Duration(rl.config.WindowSeconds) * time.Second
	if window == 0 {
		window = 60 * time.Second // 默认 60 秒窗口
	}

	allowed, remaining, resetTime, err := rl.redisLimiter.CheckLimit(ctx, key, limit, window)
	if err != nil {
		logger.Warnf("Redis rate limit check failed, allowing request: %v", err)
		return true, limit, 0
	}

	retryAfter := time.Until(resetTime)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return allowed, remaining, retryAfter
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
		ip := extractIP(r)
		requestID := GetReqID(r.Context())

		if rl.useRedis {
			// 使用 Redis 分布式限流
			// 先检查全局限流
			globalKey := "global"
			globalLimit := rl.config.RPS * rl.config.WindowSeconds
			if globalLimit <= 0 {
				globalLimit = 100 * 60 // 默认每分钟 6000 请求
			}

			allowed, _, retryAfter := rl.checkRedisLimit(r.Context(), globalKey, globalLimit)
			if !allowed {
				logger.Warnf("[%s] global rate limit exceeded (Redis)", requestID)
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// 检查基于 IP 的限流
			ipKey := "ip:" + ip
			ipLimit := rl.config.RPS * rl.config.WindowSeconds
			if ipLimit <= 0 {
				ipLimit = 100 * 60 // 默认每分钟 6000 请求
			}

			allowed, remaining, retryAfter := rl.checkRedisLimit(r.Context(), ipKey, ipLimit)
			if !allowed {
				logger.Warnf("[%s] IP rate limit exceeded for %s (Redis)", requestID, ip)
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				w.Header().Set("X-RateLimit-Remaining", "0")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// 设置剩余请求数响应头
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		} else {
			// 使用内存限流
			// 先检查全局限流
			if !rl.globalLimiter.Allow() {
				logger.Warnf("[%s] global rate limit exceeded from %s", requestID, ip)
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// 检查基于 IP 的限流
			globalLimit := rl.globalLimiter.Limit()
			globalBurst := rl.globalLimiter.Burst()
			ipLimiter := rl.getIPLimiter(ip, int(globalLimit), globalBurst)
			if !ipLimiter.Allow() {
				logger.Warnf("[%s] IP rate limit exceeded for %s", requestID, ip)
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
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
				requestID := GetReqID(r.Context())

				if rl.useRedis {
					// 使用 Redis 分布式限流
					hookKey := "hook:" + hookID
					// 计算时间窗口内允许的请求数
					windowSeconds := rl.config.WindowSeconds
					if windowSeconds <= 0 {
						windowSeconds = 60
					}
					hookLimit := rps * windowSeconds

					allowed, remaining, retryAfter := rl.checkRedisLimit(r.Context(), hookKey, hookLimit)
					if !allowed {
						logger.Warnf("[%s] hook rate limit exceeded for hook %s (Redis)", requestID, hookID)
						w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
						w.Header().Set("X-RateLimit-Remaining", "0")
						http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
						return
					}
					w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
				} else {
					// 使用内存限流
					hookLimiter := rl.getHookLimiter(hookID, rps, burst)
					if !hookLimiter.Allow() {
						logger.Warnf("[%s] hook rate limit exceeded for hook %s", requestID, hookID)
						w.Header().Set("Retry-After", "1")
						http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
						return
					}
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

// NewRateLimiterWithRedis 创建带 Redis 支持的限流器（便捷方法）
func NewRateLimiterWithRedis(enabled bool, rps, burst int, redisAddr, redisPassword string, redisDB int, keyPrefix string, windowSeconds int) *RateLimiter {
	config := RateLimitConfig{
		Enabled:        enabled,
		RPS:            rps,
		Burst:          burst,
		RedisEnabled:   redisAddr != "",
		RedisAddr:      redisAddr,
		RedisPassword:  redisPassword,
		RedisDB:        redisDB,
		RedisKeyPrefix: keyPrefix,
		WindowSeconds:  windowSeconds,
	}
	return NewRateLimiter(config)
}
