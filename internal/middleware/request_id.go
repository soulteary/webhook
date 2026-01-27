package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// contextKey 用于 context 键，避免与基础类型冲突。
type requestIDContextKey struct{}

// RequestIDKey 是存入 context 的请求 ID 键，供 GetReqID 等使用。
var RequestIDKey = requestIDContextKey{}

// GetReqID 从 context 中读取请求 ID，若不存在或类型非 string 则返回空字符串。
func GetReqID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestIDOption 用于配置 RequestID 中间件。
type RequestIDOption func(*RequestIDOptions)

// RequestIDOptions 是 RequestID 中间件的配置。
type RequestIDOptions struct {
	useXRequestID  bool
	requestIDLimit int
}

// UseRequestID 返回是否使用请求头中的 X-Request-Id。
func (o *RequestIDOptions) UseRequestID() bool {
	return o != nil && o.useXRequestID
}

func newRequestIDOptions(opts ...RequestIDOption) *RequestIDOptions {
	o := &RequestIDOptions{}
	for _, f := range opts {
		f(o)
	}
	return o
}

// UseXRequestIDHeaderOption 设置是否优先从 X-Request-Id 请求头读取。
func UseXRequestIDHeaderOption(use bool) RequestIDOption {
	return func(o *RequestIDOptions) {
		o.useXRequestID = use
	}
}

// XRequestIDLimitOption 限制从请求头读取的 X-Request-Id 长度，0 表示不限制。
func XRequestIDLimitOption(limit int) RequestIDOption {
	return func(o *RequestIDOptions) {
		o.requestIDLimit = limit
	}
}

const xRequestIDHeader = "X-Request-Id"

// RequestID 返回注入请求 ID 的中间件：从上下文或请求头读取、或生成新 ID，并写入 context 与响应头。
func RequestID(opts ...RequestIDOption) func(next http.Handler) http.Handler {
	o := newRequestIDOptions(opts...)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := GetReqID(r.Context())
			if id == "" && o.UseRequestID() {
				id = r.Header.Get(xRequestIDHeader)
				if o.requestIDLimit > 0 && len(id) > o.requestIDLimit {
					id = id[:o.requestIDLimit]
				}
			}
			if id == "" {
				id = generateRequestID()
			}
			ctx := context.WithValue(r.Context(), RequestIDKey, id)
			r = r.WithContext(ctx)
			w.Header().Set(xRequestIDHeader, id)
			next.ServeHTTP(w, r)
		})
	}
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
