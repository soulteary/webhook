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

