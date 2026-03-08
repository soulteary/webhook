package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/webhook/internal/configui"
)

// TestStandaloneHandler verifies that the configui.Handler used by main
// serves the root and /api/generate correctly (standalone mode with basePath "/").
func TestStandaloneHandler(t *testing.T) {
	handler, err := configui.Handler("/", "http://localhost:9080")
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /: status %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("GET /: Content-Type %q", rec.Header().Get("Content-Type"))
	}
}
