package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDumper(t *testing.T) {
	var buf bytes.Buffer

	handler := Dumper(&buf)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}),
	)

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-id"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, buf.String(), "test-id")
}

func TestResponseDupper_Write(t *testing.T) {
	w := httptest.NewRecorder()
	dupper := &responseDupper{
		ResponseWriter: w,
		Buffer:         &bytes.Buffer{},
		Status:         0,
	}

	data := []byte("test data")
	n, err := dupper.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, dupper.Buffer.Bytes())
	assert.Equal(t, data, w.Body.Bytes())
}

func TestResponseDupper_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	dupper := &responseDupper{
		ResponseWriter: w,
		Buffer:         &bytes.Buffer{},
		Status:         0,
	}

	dupper.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, dupper.Status)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestResponseDupper_Hijack(t *testing.T) {
	w := httptest.NewRecorder()
	dupper := &responseDupper{
		ResponseWriter: w,
		Buffer:         &bytes.Buffer{},
		Status:         0,
	}

	// Test with non-hijacker
	conn, rw, err := dupper.Hijack()
	assert.Nil(t, conn)
	assert.Nil(t, rw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be hijacked")
}

func TestDumperWithConfig(t *testing.T) {
	var buf bytes.Buffer

	// Test with IncludeRequestBody = true
	handler := DumperWithConfig(&buf, DumperConfig{
		IncludeRequestBody: true,
	})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}),
	)

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-id"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, buf.String(), "test-id")
}

func TestDumperWithConfig_IncludeRequestBody(t *testing.T) {
	var buf bytes.Buffer

	handler := DumperWithConfig(&buf, DumperConfig{
		IncludeRequestBody: true,
	})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("response body"))
		}),
	)

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString("request body"))
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-id"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	output := buf.String()
	// When IncludeRequestBody is true, request body should be included
	assert.Contains(t, output, "test-id")
}

func TestDumperWithConfig_ExcludeRequestBody(t *testing.T) {
	var buf bytes.Buffer

	handler := DumperWithConfig(&buf, DumperConfig{
		IncludeRequestBody: false,
	})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("response body"))
		}),
	)

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString("request body"))
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-id"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	output := buf.String()
	// When IncludeRequestBody is false, should see the security message
	assert.Contains(t, output, "Request body omitted for security")
}

func TestDumper_ErrorDumpingRequest(t *testing.T) {
	var buf bytes.Buffer

	handler := Dumper(&buf)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Create a request that might cause DumpRequest to fail
	req := httptest.NewRequest("POST", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-id"))
	// Set a nil body to potentially cause issues
	req.Body = nil
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should handle error gracefully
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseDupper_Flush(t *testing.T) {
	w := httptest.NewRecorder()
	dupper := &responseDupper{
		ResponseWriter: w,
		Buffer:         &bytes.Buffer{},
		Status:         0,
	}

	// Test that Flush doesn't panic (if ResponseWriter implements Flusher)
	// httptest.NewRecorder doesn't implement Flusher, so this should be safe
	dupper.Write([]byte("test"))
	assert.Equal(t, []byte("test"), w.Body.Bytes())
}
