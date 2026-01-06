package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HookExecutions 记录 hook 执行总数，按 hook_id 和 status 分类
	HookExecutions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_executions_total",
			Help: "Total number of hook executions",
		},
		[]string{"hook_id", "status"},
	)

	// HookDuration 记录 hook 执行时间，按 hook_id 分类
	HookDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_execution_duration_seconds",
			Help:    "Hook execution duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms 到 ~32s
		},
		[]string{"hook_id"},
	)

	// HTTPRequests 记录 HTTP 请求总数，按 method、status_code 和 path 分类
	HTTPRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "status_code", "path"},
	)

	// HTTPRequestDuration 记录 HTTP 请求处理时间，按 method 和 path 分类
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms 到 ~32s
		},
		[]string{"method", "path"},
	)

	// ConcurrentHooks 记录当前并发执行的 hook 数量，按 hook_id 分类
	ConcurrentHooks = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_concurrent_hooks",
			Help: "Current number of concurrent hook executions",
		},
		[]string{"hook_id"},
	)

	// SystemMemoryBytes 记录系统内存使用量（字节）
	SystemMemoryBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_system_memory_bytes",
			Help: "System memory usage in bytes",
		},
		[]string{"type"}, // type: "alloc", "sys", "total", "heap", "stack"
	)

	// SystemCPUPercent 记录系统 CPU 使用率（百分比）
	SystemCPUPercent = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webhook_system_cpu_percent",
			Help: "System CPU usage percentage",
		},
	)

	// SystemGoroutines 记录当前 goroutine 数量
	SystemGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webhook_system_goroutines",
			Help: "Current number of goroutines",
		},
	)

	// 用于跟踪并发 hook 执行的计数器
	concurrentHooksMap = make(map[string]int)
	concurrentHooksMu  sync.Mutex
)

// RecordHookExecution 记录 hook 执行
func RecordHookExecution(hookID, status string, duration time.Duration) {
	HookExecutions.WithLabelValues(hookID, status).Inc()
	HookDuration.WithLabelValues(hookID).Observe(duration.Seconds())
}

// IncrementConcurrentHooks 增加并发 hook 计数
func IncrementConcurrentHooks(hookID string) {
	concurrentHooksMu.Lock()
	defer concurrentHooksMu.Unlock()
	concurrentHooksMap[hookID]++
	ConcurrentHooks.WithLabelValues(hookID).Set(float64(concurrentHooksMap[hookID]))
}

// DecrementConcurrentHooks 减少并发 hook 计数
func DecrementConcurrentHooks(hookID string) {
	concurrentHooksMu.Lock()
	defer concurrentHooksMu.Unlock()
	if count, exists := concurrentHooksMap[hookID]; exists && count > 0 {
		concurrentHooksMap[hookID]--
		ConcurrentHooks.WithLabelValues(hookID).Set(float64(concurrentHooksMap[hookID]))
		if concurrentHooksMap[hookID] == 0 {
			delete(concurrentHooksMap, hookID)
		}
	}
}

// RecordHTTPRequest 记录 HTTP 请求
func RecordHTTPRequest(method, statusCode, path string, duration time.Duration) {
	HTTPRequests.WithLabelValues(method, statusCode, path).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// UpdateSystemMetrics 更新系统指标（内存、CPU、goroutine）
func UpdateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 内存指标
	SystemMemoryBytes.WithLabelValues("alloc").Set(float64(m.Alloc))
	SystemMemoryBytes.WithLabelValues("sys").Set(float64(m.Sys))
	SystemMemoryBytes.WithLabelValues("total").Set(float64(m.TotalAlloc))
	SystemMemoryBytes.WithLabelValues("heap").Set(float64(m.HeapAlloc))
	SystemMemoryBytes.WithLabelValues("stack").Set(float64(m.StackInuse))

	// Goroutine 数量
	SystemGoroutines.Set(float64(runtime.NumGoroutine()))

	// CPU 使用率（简化版本，基于 GC 时间）
	// 注意：这是一个简化的实现，实际 CPU 使用率需要更复杂的计算
	// 可以使用 gopsutil 等库来获取更准确的 CPU 使用率
	if m.NumGC > 0 {
		// 使用 GC 时间作为 CPU 使用率的近似值
		cpuPercent := float64(m.PauseTotalNs) / float64(time.Second.Nanoseconds()) * 100
		if cpuPercent > 100 {
			cpuPercent = 100
		}
		SystemCPUPercent.Set(cpuPercent)
	}
}

// StartSystemMetricsCollector 启动系统指标收集器（定期更新）
func StartSystemMetricsCollector(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			UpdateSystemMetrics()
		}
	}()
}

