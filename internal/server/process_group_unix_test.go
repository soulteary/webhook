//go:build !windows

package server

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/stretchr/testify/require"
)

// 超时后必须终止脚本派生的子进程，且不能因为子进程持有输出管道而一直阻塞。
func TestHandleHook_TimeoutKillsProcessTree(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "child.pid")
	scriptPath := filepath.Join(tempDir, "spawn.sh")
	// 后台子进程继承 stdout/stderr；脚本自身再等待它。
	script := "#!/bin/sh\nsleep 60 &\necho $! > " + pidFile + "\nwait\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0755))

	h := &hook.Hook{
		ID:                      "tree-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:    true,
	}
	const hookTimeout = 2 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), hookTimeout)
	defer cancel()

	start := time.Now()
	_, err := handleHook(ctx, h, &hook.Request{ID: "req"}, nil, flags.AppFlags{})
	elapsed := time.Since(start)
	require.Error(t, err)
	require.Less(t, elapsed, hookTimeout+3*time.Second, "handleHook must return promptly after the timeout")

	raw, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	// 信号 0 只探测进程是否存在；给内核一点时间回收。
	require.Eventually(t, func() bool {
		return syscall.Kill(pid, 0) != nil
	}, 2*time.Second, 50*time.Millisecond, "background child %d survived the hook timeout", pid)
}
