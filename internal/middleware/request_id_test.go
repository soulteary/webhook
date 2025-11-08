package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	// Test without options
	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetReqID(r.Context())
		assert.NotEmpty(t, id)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestID_WithXRequestIDHeader(t *testing.T) {
	// Test with X-Request-Id header
	handler := RequestID(UseXRequestIDHeaderOption(true))(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := GetReqID(r.Context())
			assert.Equal(t, "custom-id", id)
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-Id", "custom-id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestID_WithLimit(t *testing.T) {
	// Test with limit option
	handler := RequestID(
		UseXRequestIDHeaderOption(true),
		XRequestIDLimitOption(5),
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := GetReqID(r.Context())
			assert.Equal(t, "12345", id)
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-Id", "123456789")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetReqID(t *testing.T) {
	// Test with nil context
	assert.Empty(t, GetReqID(nil))

	// Test with context without request ID
	ctx := context.Background()
	assert.Empty(t, GetReqID(ctx))

	// Test with context with request ID
	ctx = context.WithValue(ctx, RequestIDKey, "test-id")
	assert.Equal(t, "test-id", GetReqID(ctx))

	// Test with wrong type
	ctx = context.WithValue(ctx, RequestIDKey, 123)
	assert.Empty(t, GetReqID(ctx))
}

func TestUseXRequestIDHeaderOption(t *testing.T) {
	opt := UseXRequestIDHeaderOption(true)
	options := newRequestIDOptions(opt)
	assert.True(t, options.UseRequestID())

	opt = UseXRequestIDHeaderOption(false)
	options = newRequestIDOptions(opt)
	assert.False(t, options.UseRequestID())
}

func TestXRequestIDLimitOption(t *testing.T) {
	opt := XRequestIDLimitOption(10)
	options := newRequestIDOptions(opt)
	assert.Equal(t, 10, options.requestIDLimit)
}

func TestRequestIDOptions_UseRequestID(t *testing.T) {
	options := &RequestIDOptions{
		useXRequestID: true,
	}
	assert.True(t, options.UseRequestID())

	options.useXRequestID = false
	assert.False(t, options.UseRequestID())
}

func TestNewRequestIDOptions(t *testing.T) {
	// Test with no options
	options := newRequestIDOptions()
	assert.NotNil(t, options)
	assert.False(t, options.UseRequestID())
	assert.Equal(t, 0, options.requestIDLimit)

	// Test with multiple options
	options = newRequestIDOptions(
		UseXRequestIDHeaderOption(true),
		XRequestIDLimitOption(20),
	)
	assert.True(t, options.UseRequestID())
	assert.Equal(t, 20, options.requestIDLimit)
}
