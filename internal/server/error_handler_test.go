package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/security"
)

func TestNewHTTPError(t *testing.T) {
	err := errors.New("test error")
	httpErr := NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test message", err)

	if httpErr == nil {
		t.Fatal("NewHTTPError() should not return nil")
	}

	if httpErr.Type != ErrorTypeClient {
		t.Errorf("Type = %s, want %s", httpErr.Type, ErrorTypeClient)
	}

	if httpErr.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", httpErr.Status, http.StatusBadRequest)
	}

	if httpErr.Message != "test message" {
		t.Errorf("Message = %s, want test message", httpErr.Message)
	}

	if httpErr.Err != err {
		t.Error("Err should be set correctly")
	}
}

func TestHTTPError_WithRequestID(t *testing.T) {
	httpErr := NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil)
	httpErr = httpErr.WithRequestID("req-123")

	if httpErr.RequestID != "req-123" {
		t.Errorf("RequestID = %s, want req-123", httpErr.RequestID)
	}
}

func TestHTTPError_WithHookID(t *testing.T) {
	httpErr := NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil)
	httpErr = httpErr.WithHookID("hook-456")

	if httpErr.HookID != "hook-456" {
		t.Errorf("HookID = %s, want hook-456", httpErr.HookID)
	}
}

func TestHTTPError_Error(t *testing.T) {
	tests := []struct {
		name    string
		httpErr *HTTPError
		want    string
	}{
		{"with error", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", errors.New("underlying error")), "underlying error"},
		{"without error", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test message", nil), "test message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.httpErr.Error()
			if got != tt.want {
				t.Errorf("Error() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantType   ErrorType
		wantStatus int
	}{
		{"nil error", nil, "", 0},
		{"HTTPError", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil), ErrorTypeClient, http.StatusBadRequest},
		{"context deadline exceeded", context.DeadlineExceeded, ErrorTypeTimeout, http.StatusRequestTimeout},
		{"context canceled", context.Canceled, ErrorTypeTimeout, http.StatusRequestTimeout},
		{"parameter node error", &hook.ParameterNodeError{Key: "test"}, ErrorTypeClient, http.StatusBadRequest},
		{"signature error", &hook.SignatureError{Signature: "invalid"}, ErrorTypeClient, http.StatusUnauthorized},
		{"command validation error", security.NewCommandValidationError("path", "test", "/usr/bin/ls", nil), ErrorTypeServer, http.StatusInternalServerError},
		{"permission denied", errors.New("permission denied"), ErrorTypeClient, http.StatusBadRequest},
		{"not found", errors.New("not found"), ErrorTypeClient, http.StatusBadRequest},
		{"invalid", errors.New("invalid request"), ErrorTypeClient, http.StatusBadRequest},
		{"generic error", errors.New("generic error"), ErrorTypeServer, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpErr := ClassifyError(tt.err, "req-123", "hook-456")
			if tt.err == nil {
				if httpErr != nil {
					t.Error("ClassifyError() should return nil for nil error")
				}
				return
			}

			if httpErr == nil {
				t.Fatal("ClassifyError() should not return nil for non-nil error")
			}

			if httpErr.Type != tt.wantType {
				t.Errorf("Type = %s, want %s", httpErr.Type, tt.wantType)
			}

			if httpErr.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", httpErr.Status, tt.wantStatus)
			}

			if httpErr.RequestID != "req-123" {
				t.Errorf("RequestID = %s, want req-123", httpErr.RequestID)
			}

			if httpErr.HookID != "hook-456" {
				t.Errorf("HookID = %s, want hook-456", httpErr.HookID)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantJSON   bool
	}{
		{"nil error", nil, 0, false},
		{"client error", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil), http.StatusBadRequest, true},
		{"server error", NewHTTPError(ErrorTypeServer, http.StatusInternalServerError, "test", nil), http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)

			HandleError(w, r, tt.err, "req-123", "hook-456")

			if tt.err == nil {
				// 对于 nil error，HandleError 应该直接返回，不写入任何响应
				// 检查响应体是否为空，以及是否没有设置 Content-Type
				if w.Body.Len() != 0 {
					t.Error("HandleError() should not write response body for nil error")
				}
				if w.Header().Get("Content-Type") != "" {
					t.Error("HandleError() should not set Content-Type header for nil error")
				}
				return
			}

			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantJSON {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}

				if resp.RequestID != "req-123" {
					t.Errorf("RequestID = %s, want req-123", resp.RequestID)
				}

				if resp.HookID != "hook-456" {
					t.Errorf("HookID = %s, want hook-456", resp.HookID)
				}
			}
		})
	}
}

func TestHandleErrorPlain(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test message", nil)

	HandleErrorPlain(w, err, "req-123", "hook-456")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %s, want text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if body != "test message" {
		t.Errorf("Body = %s, want test message", body)
	}
}

func TestHandleErrorWithCustomMessage(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "original", nil)

	HandleErrorWithCustomMessage(w, err, "req-123", "hook-456", "custom message", http.StatusForbidden)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusForbidden)
	}

	body := w.Body.String()
	if body != "custom message" {
		t.Errorf("Body = %s, want custom message", body)
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"client error", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil), true},
		{"server error", NewHTTPError(ErrorTypeServer, http.StatusInternalServerError, "test", nil), false},
		{"timeout error", NewHTTPError(ErrorTypeTimeout, http.StatusRequestTimeout, "test", nil), false},
		{"regular error", errors.New("test"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsClientError(tt.err)
			if got != tt.want {
				t.Errorf("IsClientError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"server error", NewHTTPError(ErrorTypeServer, http.StatusInternalServerError, "test", nil), true},
		{"client error", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil), false},
		{"timeout error", NewHTTPError(ErrorTypeTimeout, http.StatusRequestTimeout, "test", nil), false},
		{"regular error", errors.New("test"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsServerError(tt.err)
			if got != tt.want {
				t.Errorf("IsServerError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"timeout error", NewHTTPError(ErrorTypeTimeout, http.StatusRequestTimeout, "test", nil), true},
		{"client error", NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test", nil), false},
		{"server error", NewHTTPError(ErrorTypeServer, http.StatusInternalServerError, "test", nil), false},
		{"regular error", errors.New("test"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTimeoutError(tt.err)
			if got != tt.want {
				t.Errorf("IsTimeoutError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	err := NewHTTPError(ErrorTypeClient, http.StatusBadRequest, "test message", errors.New("underlying"))
	HandleError(w, r, err, "req-123", "hook-456")

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if resp.Error != "Bad Request" {
		t.Errorf("Error = %s, want Bad Request", resp.Error)
	}

	if resp.Message != "test message" {
		t.Errorf("Message = %s, want test message", resp.Message)
	}

	if resp.RequestID != "req-123" {
		t.Errorf("RequestID = %s, want req-123", resp.RequestID)
	}

	if resp.HookID != "hook-456" {
		t.Errorf("HookID = %s, want hook-456", resp.HookID)
	}
}
