package middleware

import (
	"net/http"

	middlewarekit "github.com/soulteary/middleware-kit"
)

// SecurityConfig 安全中间件配置
type SecurityConfig struct {
	// Enabled 是否启用安全头
	Enabled bool

	// Strict 是否使用严格模式（更多安全头）
	Strict bool

	// HSTS 是否启用 HTTP Strict Transport Security
	HSTS bool

	// HSTSMaxAge HSTS 最大有效期（秒）
	HSTSMaxAge int

	// HSTSIncludeSubDomains HSTS 是否包含子域名
	HSTSIncludeSubDomains bool

	// ContentSecurityPolicy 自定义 CSP 策略
	ContentSecurityPolicy string

	// CrossOriginResourcePolicy 跨域资源策略
	CrossOriginResourcePolicy string

	// CustomHeaders 自定义头部
	CustomHeaders map[string]string
}

// DefaultSecurityConfig 返回默认安全配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		Enabled:               true,
		Strict:                false,
		HSTS:                  false,
		HSTSMaxAge:            31536000, // 1 年
		HSTSIncludeSubDomains: true,
	}
}

// StrictSecurityConfig 返回严格安全配置
func StrictSecurityConfig() SecurityConfig {
	return SecurityConfig{
		Enabled:                   true,
		Strict:                    true,
		HSTS:                      true,
		HSTSMaxAge:                31536000, // 1 年
		HSTSIncludeSubDomains:     true,
		CrossOriginResourcePolicy: "same-origin",
	}
}

// SecurityHeaders 返回安全头中间件
// 使用 middleware-kit 的 SecurityHeadersStd 实现
func SecurityHeaders(cfg SecurityConfig) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	var kitCfg middlewarekit.SecurityHeadersConfig
	if cfg.Strict {
		kitCfg = middlewarekit.StrictSecurityHeadersConfig()
	} else {
		kitCfg = middlewarekit.DefaultSecurityHeadersConfig()
	}

	// 应用 HSTS 配置
	if cfg.HSTS {
		hsts := "max-age=" + itoa(cfg.HSTSMaxAge)
		if cfg.HSTSIncludeSubDomains {
			hsts += "; includeSubDomains"
		}
		kitCfg.StrictTransportSecurity = hsts
	}

	// 应用自定义 CSP
	if cfg.ContentSecurityPolicy != "" {
		kitCfg.ContentSecurityPolicy = cfg.ContentSecurityPolicy
	}

	// 应用跨域资源策略
	if cfg.CrossOriginResourcePolicy != "" {
		kitCfg.CrossOriginResourcePolicy = cfg.CrossOriginResourcePolicy
	}

	// 应用自定义头部
	if len(cfg.CustomHeaders) > 0 {
		if kitCfg.CustomHeaders == nil {
			kitCfg.CustomHeaders = make(map[string]string)
		}
		for k, v := range cfg.CustomHeaders {
			kitCfg.CustomHeaders[k] = v
		}
	}

	return middlewarekit.SecurityHeadersStd(kitCfg)
}

// NoCacheHeaders 返回禁用缓存的中间件
// 使用 middleware-kit 的 NoCacheHeadersStd 实现
func NoCacheHeaders() func(http.Handler) http.Handler {
	return middlewarekit.NoCacheHeadersStd()
}

// BodyLimit 返回请求体大小限制中间件
// 使用 middleware-kit 的 BodyLimitStd 实现
func BodyLimit(maxSize int64) func(http.Handler) http.Handler {
	return middlewarekit.BodyLimitStd(middlewarekit.BodyLimitConfig{
		MaxSize: maxSize,
	})
}

// BodyLimitWithConfig 返回带配置的请求体大小限制中间件
func BodyLimitWithConfig(cfg middlewarekit.BodyLimitConfig) func(http.Handler) http.Handler {
	return middlewarekit.BodyLimitStd(cfg)
}

// TrustedProxyConfig 可信代理配置（re-export from middleware-kit）
type TrustedProxyConfig = middlewarekit.TrustedProxyConfig

// NewTrustedProxyConfig 创建可信代理配置
func NewTrustedProxyConfig(proxies []string) *TrustedProxyConfig {
	return middlewarekit.NewTrustedProxyConfig(proxies)
}

// GetClientIPWithConfig 使用指定配置获取客户端 IP
func GetClientIPWithConfig(r *http.Request, cfg *TrustedProxyConfig) string {
	return middlewarekit.GetClientIP(r, cfg)
}

// MaskEmail 对邮箱地址进行脱敏
func MaskEmail(email string) string {
	return middlewarekit.MaskEmail(email)
}

// MaskPhone 对手机号进行脱敏
func MaskPhone(phone string) string {
	return middlewarekit.MaskPhone(phone)
}

// itoa 简单的整数转字符串
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	neg := false
	if i < 0 {
		neg = true
		i = -i
	}

	var buf [20]byte
	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	if neg {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
