package metrics

import (
	"testing"
	"time"
)

func TestRecordHookExecution(t *testing.T) {
	hookID := "test-hook-1"
	status := "success"
	duration := 100 * time.Millisecond

	// 这个测试主要确保函数不会 panic
	RecordHookExecution(hookID, status, duration)
	RecordHookExecution(hookID, "failure", duration)
}

func TestIncrementDecrementConcurrentHooks(t *testing.T) {
	hookID := "test-hook-1"

	// 测试增加
	IncrementConcurrentHooks(hookID)
	IncrementConcurrentHooks(hookID)

	// 测试减少
	DecrementConcurrentHooks(hookID)
	DecrementConcurrentHooks(hookID)

	// 测试减少到0后删除
	DecrementConcurrentHooks(hookID) // 应该不会 panic

	// 测试多个 hook
	IncrementConcurrentHooks("hook-1")
	IncrementConcurrentHooks("hook-2")
	DecrementConcurrentHooks("hook-1")
	DecrementConcurrentHooks("hook-2")
}

func TestRecordHTTPRequest(t *testing.T) {
	method := "GET"
	statusCode := "200"
	path := "/hooks/test"
	duration := 50 * time.Millisecond

	// 这个测试主要确保函数不会 panic
	RecordHTTPRequest(method, statusCode, path, duration)
	RecordHTTPRequest("POST", "404", "/hooks/notfound", duration)
}

func TestUpdateSystemMetrics(t *testing.T) {
	// 这个测试主要确保函数不会 panic
	UpdateSystemMetrics()

	// 多次调用确保稳定性
	for i := 0; i < 10; i++ {
		UpdateSystemMetrics()
	}
}

func TestStartSystemMetricsCollector(t *testing.T) {
	// 启动收集器
	stop := StartSystemMetricsCollector(100 * time.Millisecond)

	// 确保在测试结束时停止收集器
	t.Cleanup(func() {
		stop()
	})

	// 等待一段时间让收集器运行
	time.Sleep(200 * time.Millisecond)

	// 更新指标
	UpdateSystemMetrics()
}

func TestConcurrentMetrics(t *testing.T) {
	// 测试并发访问指标
	done := make(chan bool)
	concurrency := 10

	// 并发记录 hook 执行
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				RecordHookExecution("hook-1", "success", time.Millisecond)
				IncrementConcurrentHooks("hook-1")
				DecrementConcurrentHooks("hook-1")
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestMetricsWithDifferentLabels(t *testing.T) {
	// 测试不同标签的指标
	hookIDs := []string{"hook-1", "hook-2", "hook-3"}
	statuses := []string{"success", "failure", "timeout"}

	for _, hookID := range hookIDs {
		for _, status := range statuses {
			RecordHookExecution(hookID, status, time.Millisecond)
		}
	}

	// 测试 HTTP 请求的不同标签
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statusCodes := []string{"200", "404", "500"}
	paths := []string{"/hooks/test", "/hooks/other", "/health"}

	for _, method := range methods {
		for _, statusCode := range statusCodes {
			for _, path := range paths {
				RecordHTTPRequest(method, statusCode, path, time.Millisecond)
			}
		}
	}
}

func TestSystemMemoryMetrics(t *testing.T) {
	// 更新系统指标
	UpdateSystemMetrics()

	// 验证指标已设置（通过再次更新，确保不会 panic）
	UpdateSystemMetrics()
}

func TestGoroutineMetrics(t *testing.T) {
	// 更新系统指标
	UpdateSystemMetrics()

	// 启动一些 goroutine 来改变 goroutine 数量
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			time.Sleep(10 * time.Millisecond)
			done <- true
		}()
	}

	// 等待 goroutine 完成
	for i := 0; i < 5; i++ {
		<-done
	}

	// 再次更新指标
	UpdateSystemMetrics()
}
