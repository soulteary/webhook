package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/hook"
)

// When every execution slot is busy, an async hook must answer 503 instead of
// 200-then-drop, so the sender knows to retry.
func TestExecuteAsyncHook_RejectsWhenSaturated(t *testing.T) {
	started := make(chan struct{}, 2)
	unblock := make(chan struct{})
	executor := NewHookExecutorWithFunc(1, time.Minute, func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		started <- struct{}{}
		<-unblock
		return "", nil
	})
	h := &hook.Hook{ID: "async", ResponseMessage: "accepted"}
	req := &hook.Request{ID: "r"}

	first := httptest.NewRecorder()
	executeAsyncHook(first, context.Background(), h, req, executor, 50*time.Millisecond, "r1", "async", time.Now())
	if first.Code != http.StatusOK || first.Body.String() != "accepted" {
		t.Fatalf("first request: status %d body %q", first.Code, first.Body.String())
	}
	<-started

	second := httptest.NewRecorder()
	executeAsyncHook(second, context.Background(), h, req, executor, 50*time.Millisecond, "r2", "async", time.Now())
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("saturated request: status %d, want 503", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("saturated response has no Retry-After header")
	}

	close(unblock)
	GetAsyncHookWaitGroup().Wait()
	select {
	case <-started:
		t.Fatal("rejected request was executed anyway")
	default:
	}

	// The slot is released once the first command ends.
	third := httptest.NewRecorder()
	executeAsyncHook(third, context.Background(), h, req, executor, 50*time.Millisecond, "r3", "async", time.Now())
	if third.Code != http.StatusOK {
		t.Fatalf("request after release: status %d", third.Code)
	}
	GetAsyncHookWaitGroup().Wait()
}
