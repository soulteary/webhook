package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	assert.NotNil(t, logger)

	// Create a test handler
	handler := logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-id"))
	w := httptest.NewRecorder()

	// Execute the handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_NewLogEntry(t *testing.T) {
	logger := &Logger{}
	req := httptest.NewRequest("GET", "/test", nil)

	entry := logger.NewLogEntry(req)
	assert.NotNil(t, entry)

	logEntry, ok := entry.(*LogEntry)
	assert.True(t, ok)
	assert.NotNil(t, logEntry.req)
	assert.NotNil(t, logEntry.buf)
}

func TestLogEntry_Write(t *testing.T) {
	logger := &Logger{}
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-123"))

	entry := logger.NewLogEntry(req).(*LogEntry)
	assert.NotNil(t, entry)

	// Test Write with request ID
	entry.Write(http.StatusOK, 100, http.Header{}, 0, nil)

	// Test Write without request ID
	req2 := httptest.NewRequest("POST", "/test2", nil)
	entry2 := logger.NewLogEntry(req2).(*LogEntry)
	entry2.Write(http.StatusCreated, 200, http.Header{}, 0, nil)
}

func TestLogEntry_Panic(t *testing.T) {
	logger := &Logger{}
	req := httptest.NewRequest("GET", "/test", nil)

	entry := logger.NewLogEntry(req).(*LogEntry)
	stack := []byte("test stack trace")

	// Test Panic method
	entry.Panic("test panic", stack)
}
