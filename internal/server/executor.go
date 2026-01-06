package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/soulteary/webhook/internal/hook"
)

const (
	// DefaultHookTimeout 默认 hook 执行超时时间
	DefaultHookTimeout = 30 * time.Second
	// DefaultMaxConcurrentHooks 默认最大并发执行的 hook 数量
	DefaultMaxConcurrentHooks = 10
	// HookExecutionTimeout 获取 semaphore 的超时时间
	HookExecutionTimeout = 5 * time.Second
)

// HookExecutor 管理 hook 执行的并发控制和超时
type HookExecutor struct {
	sem            chan struct{}
	maxConcurrent  int
	defaultTimeout time.Duration
	executorFunc   func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error)
}

// NewHookExecutor 已废弃，请使用 NewHookExecutorWithFunc
// 此函数会 panic，因为现在 handleHook 需要 appFlags 参数
func NewHookExecutor(maxConcurrent int, defaultTimeout time.Duration) *HookExecutor {
	panic("NewHookExecutor is deprecated. Use NewHookExecutorWithFunc instead, passing a function that wraps handleHook with appFlags")
}

// NewHookExecutorWithFunc 创建新的 HookExecutor 实例，允许自定义执行函数（主要用于测试）
func NewHookExecutorWithFunc(maxConcurrent int, defaultTimeout time.Duration, executorFunc func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error)) *HookExecutor {
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultMaxConcurrentHooks
	}
	if defaultTimeout <= 0 {
		defaultTimeout = DefaultHookTimeout
	}
	return &HookExecutor{
		sem:            make(chan struct{}, maxConcurrent),
		maxConcurrent:  maxConcurrent,
		defaultTimeout: defaultTimeout,
		executorFunc:   executorFunc,
	}
}

// Execute 执行 hook，带并发控制和超时
func (he *HookExecutor) Execute(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter, executionTimeout time.Duration) (string, error) {
	// 尝试获取 semaphore，带超时
	select {
	case he.sem <- struct{}{}:
		defer func() { <-he.sem }()
	case <-time.After(executionTimeout):
		return "", errors.New("too many concurrent hooks, execution timeout")
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// 创建带超时的 context
	timeout := he.defaultTimeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return he.executorFunc(execCtx, h, r, w)
}

// GetMaxConcurrent 获取最大并发数（用于测试）
func (he *HookExecutor) GetMaxConcurrent() int {
	return he.maxConcurrent
}

// GetDefaultTimeout 获取默认超时时间（用于测试）
func (he *HookExecutor) GetDefaultTimeout() time.Duration {
	return he.defaultTimeout
}
