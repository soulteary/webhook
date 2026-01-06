package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHookExecutor_DefaultValues(t *testing.T) {
	tests := []struct {
		name            string
		maxConcurrent   int
		defaultTimeout  time.Duration
		expectedMax     int
		expectedTimeout time.Duration
	}{
		{
			name:            "valid values",
			maxConcurrent:   5,
			defaultTimeout:  10 * time.Second,
			expectedMax:     5,
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "zero max concurrent uses default",
			maxConcurrent:   0,
			defaultTimeout:  20 * time.Second,
			expectedMax:     DefaultMaxConcurrentHooks,
			expectedTimeout: 20 * time.Second,
		},
		{
			name:            "negative max concurrent uses default",
			maxConcurrent:   -1,
			defaultTimeout:  15 * time.Second,
			expectedMax:     DefaultMaxConcurrentHooks,
			expectedTimeout: 15 * time.Second,
		},
		{
			name:            "zero timeout uses default",
			maxConcurrent:   3,
			defaultTimeout:  0,
			expectedMax:     3,
			expectedTimeout: DefaultHookTimeout,
		},
		{
			name:            "negative timeout uses default",
			maxConcurrent:   3,
			defaultTimeout:  -1 * time.Second,
			expectedMax:     3,
			expectedTimeout: DefaultHookTimeout,
		},
		{
			name:            "both zero use defaults",
			maxConcurrent:   0,
			defaultTimeout:  0,
			expectedMax:     DefaultMaxConcurrentHooks,
			expectedTimeout: DefaultHookTimeout,
		},
	}

	// Mock executor function for testing
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		return "", nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewHookExecutorWithFunc(tt.maxConcurrent, tt.defaultTimeout, mockExecutorFunc)
			assert.Equal(t, tt.expectedMax, executor.GetMaxConcurrent())
			assert.Equal(t, tt.expectedTimeout, executor.GetDefaultTimeout())
		})
	}
}

func TestHookExecutor_Execute_ConcurrencyControl(t *testing.T) {
	maxConcurrent := 2

	// Mock executor function that simulates work
	var mu sync.Mutex
	concurrentCount := 0
	maxSeenConcurrent := 0

	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		mu.Lock()
		concurrentCount++
		if concurrentCount > maxSeenConcurrent {
			maxSeenConcurrent = concurrentCount
		}
		mu.Unlock()

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		concurrentCount--
		mu.Unlock()

		return "done", nil
	}

	executor := NewHookExecutorWithFunc(maxConcurrent, 5*time.Second, mockExecutorFunc)

	// Launch more goroutines than maxConcurrent
	numGoroutines := 5
	var wg sync.WaitGroup
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			h := &hook.Hook{ID: "test"}
			r := &hook.Request{ID: "test-request"}
			result, err := executor.Execute(ctx, h, r, nil, 1*time.Second)
			if err == nil {
				results <- result
			}
		}()
	}

	wg.Wait()
	close(results)

	// Verify that max concurrent executions never exceeded the limit
	assert.LessOrEqual(t, maxSeenConcurrent, maxConcurrent, "max concurrent executions should not exceed limit")
}

func TestHookExecutor_Execute_ExecutionTimeout(t *testing.T) {
	// Mock executor function that takes longer than timeout
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		// Wait for context to be cancelled
		<-ctx.Done()
		return "", ctx.Err()
	}

	executor := NewHookExecutorWithFunc(1, 100*time.Millisecond, mockExecutorFunc)

	ctx := context.Background()
	h := &hook.Hook{ID: "test"}
	r := &hook.Request{ID: "test-request"}

	result, err := executor.Execute(ctx, h, r, nil, 1*time.Second)

	// Should get timeout error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	assert.Empty(t, result)
}

func TestHookExecutor_Execute_AcquisitionTimeout(t *testing.T) {
	// Mock executor function that blocks
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		// Block until context is cancelled
		<-ctx.Done()
		return "", ctx.Err()
	}

	// Create executor with maxConcurrent = 1
	executor := NewHookExecutorWithFunc(1, 5*time.Second, mockExecutorFunc)

	ctx := context.Background()
	h := &hook.Hook{ID: "test"}
	r := &hook.Request{ID: "test-request"}

	// First execution should succeed (acquires semaphore)
	go func() {
		executor.Execute(ctx, h, r, nil, 1*time.Second)
	}()

	// Give first execution time to acquire semaphore
	time.Sleep(10 * time.Millisecond)

	// Second execution should timeout trying to acquire semaphore
	acquisitionTimeout := 100 * time.Millisecond
	result, err := executor.Execute(ctx, h, r, nil, acquisitionTimeout)

	// Should get acquisition timeout error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many concurrent hooks")
	assert.Empty(t, result)
}

func TestHookExecutor_Execute_ContextCancellation(t *testing.T) {
	// Mock executor function that respects context
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}

	executor := NewHookExecutorWithFunc(1, 5*time.Second, mockExecutorFunc)

	ctx, cancel := context.WithCancel(context.Background())
	h := &hook.Hook{ID: "test"}
	r := &hook.Request{ID: "test-request"}

	// Cancel context before execution completes
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result, err := executor.Execute(ctx, h, r, nil, 1*time.Second)

	// Should get context cancelled error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Empty(t, result)
}

func TestHookExecutor_Execute_Success(t *testing.T) {
	expectedOutput := "test output"
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		return expectedOutput, nil
	}

	executor := NewHookExecutorWithFunc(1, 5*time.Second, mockExecutorFunc)

	ctx := context.Background()
	h := &hook.Hook{ID: "test"}
	r := &hook.Request{ID: "test-request"}

	result, err := executor.Execute(ctx, h, r, nil, 1*time.Second)

	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, result)
}

func TestHookExecutor_Execute_WithResponseWriter(t *testing.T) {
	var receivedWriter http.ResponseWriter
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		receivedWriter = w
		return "done", nil
	}

	executor := NewHookExecutorWithFunc(1, 5*time.Second, mockExecutorFunc)

	ctx := context.Background()
	h := &hook.Hook{ID: "test"}
	r := &hook.Request{ID: "test-request"}
	mockWriter := &executorMockResponseWriter{}

	result, err := executor.Execute(ctx, h, r, mockWriter, 1*time.Second)

	assert.NoError(t, err)
	assert.Equal(t, "done", result)
	assert.Equal(t, mockWriter, receivedWriter)
}

func TestHookExecutor_Execute_MultipleConcurrent(t *testing.T) {
	maxConcurrent := 3
	var wg sync.WaitGroup
	numExecutions := 10
	successCount := 0
	var mu sync.Mutex

	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "success", nil
	}

	executor := NewHookExecutorWithFunc(maxConcurrent, 5*time.Second, mockExecutorFunc)

	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			h := &hook.Hook{ID: "test"}
			r := &hook.Request{ID: "test-request"}
			_, err := executor.Execute(ctx, h, r, nil, 1*time.Second)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// All executions should succeed
	assert.Equal(t, numExecutions, successCount)
}

func TestHookExecutor_GetMaxConcurrent(t *testing.T) {
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		return "", nil
	}
	executor := NewHookExecutorWithFunc(5, 10*time.Second, mockExecutorFunc)
	assert.Equal(t, 5, executor.GetMaxConcurrent())
}

func TestHookExecutor_GetDefaultTimeout(t *testing.T) {
	timeout := 15 * time.Second
	mockExecutorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		return "", nil
	}
	executor := NewHookExecutorWithFunc(1, timeout, mockExecutorFunc)
	assert.Equal(t, timeout, executor.GetDefaultTimeout())
}

// executorMockResponseWriter is a helper for testing (renamed to avoid conflict with existing mockResponseWriter)
type executorMockResponseWriter struct {
	header http.Header
	code   int
	body   []byte
}

func (m *executorMockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *executorMockResponseWriter) Write(b []byte) (int, error) {
	m.body = append(m.body, b...)
	return len(b), nil
}

func (m *executorMockResponseWriter) WriteHeader(code int) {
	m.code = code
}

func TestNewHookExecutorWithFunc(t *testing.T) {
	called := false
	mockFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		called = true
		return "mocked", nil
	}

	executor := NewHookExecutorWithFunc(2, 5*time.Second, mockFunc)
	require.NotNil(t, executor)

	ctx := context.Background()
	h := &hook.Hook{ID: "test"}
	r := &hook.Request{ID: "test-request"}

	result, err := executor.Execute(ctx, h, r, nil, 1*time.Second)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "mocked", result)
}
