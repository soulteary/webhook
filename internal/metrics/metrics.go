package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metricskit "github.com/soulteary/metrics-kit"
)

// 为了向后兼容，保留原有的全局变量
var (
	// HookExecutions 记录 hook 执行总数，按 hook_id 和 status 分类
	HookExecutions *prometheus.CounterVec

	// HookDuration 记录 hook 执行时间，按 hook_id 分类
	HookDuration *prometheus.HistogramVec

	// HTTPRequests 记录 HTTP 请求总数，按 method、status_code 和 path 分类
	HTTPRequests *prometheus.CounterVec

	// HTTPRequestDuration 记录 HTTP 请求处理时间，按 method 和 path 分类
	HTTPRequestDuration *prometheus.HistogramVec

	// ConcurrentHooks 记录当前并发执行的 hook 数量，按 hook_id 分类
	ConcurrentHooks *prometheus.GaugeVec

	// SystemMemoryBytes 记录系统内存使用量（字节）
	SystemMemoryBytes *prometheus.GaugeVec

	// SystemCPUPercent 记录系统 CPU 使用率（百分比）
	SystemCPUPercent prometheus.Gauge

	// SystemGoroutines 记录当前 goroutine 数量
	SystemGoroutines prometheus.Gauge

	// 新增指标（使用 metrics-kit 构建器模式）
	// SignatureVerify 签名验证指标
	SignatureVerify *prometheus.CounterVec

	// RateLimitHits 限流命中指标
	RateLimitHits *prometheus.CounterVec

	// TriggerRules 触发规则评估指标
	TriggerRules *prometheus.CounterVec

	// 用于跟踪并发 hook 执行的计数器
	concurrentHooksMap = make(map[string]int)
	concurrentHooksMu  sync.Mutex

	// metricsInitialized 确保指标只初始化一次
	metricsOnce sync.Once
)

func init() {
	initMetrics()
}

// initMetrics 初始化所有指标（使用 metrics-kit 构建器模式）
func initMetrics() {
	metricsOnce.Do(func() {
		// 创建 metrics-kit registry（用于新指标）
		registry := metricskit.NewRegistry("webhook")

		// Hook 执行总数（使用 metrics-kit 构建器）
		HookExecutions = registry.Counter("executions_total").
			Help("Total number of hook executions").
			Labels("hook_id", "status").
			BuildVec()

		// Hook 执行时间（使用 metrics-kit 构建器和预定义桶）
		HookDuration = registry.Histogram("execution_duration_seconds").
			Help("Hook execution duration in seconds").
			Labels("hook_id").
			Buckets(metricskit.HTTPDurationBuckets()).
			BuildVec()

		// HTTP 请求总数
		HTTPRequests = registry.Counter("http_requests_total").
			Help("Total number of HTTP requests").
			Labels("method", "status_code", "path").
			BuildVec()

		// HTTP 请求处理时间
		HTTPRequestDuration = registry.Histogram("http_request_duration_seconds").
			Help("HTTP request duration in seconds").
			Labels("method", "path").
			Buckets(metricskit.HTTPDurationBuckets()).
			BuildVec()

		// 并发 hook 数量
		ConcurrentHooks = registry.Gauge("concurrent_hooks").
			Help("Current number of concurrent hook executions").
			Labels("hook_id").
			BuildVec()

		// 系统内存指标
		SystemMemoryBytes = registry.WithSubsystem("system").
			Gauge("memory_bytes").
			Help("System memory usage in bytes").
			Labels("type").
			BuildVec()

		SystemCPUPercent = registry.WithSubsystem("system").
			Gauge("cpu_percent").
			Help("Approximate CPU usage percentage based on GC time").
			Build()

		SystemGoroutines = registry.WithSubsystem("system").
			Gauge("goroutines").
			Help("Current number of goroutines").
			Build()

		// 新增：签名验证指标
		SignatureVerify = registry.Counter("signature_verify_total").
			Help("Total number of signature verifications").
			Labels("result", "algorithm").
			BuildVec()

		// 新增：限流命中指标
		RateLimitHits = registry.Counter("rate_limit_hits_total").
			Help("Total number of rate limit hits").
			Labels("scope").
			BuildVec()

		// 新增：触发规则评估指标
		TriggerRules = registry.Counter("trigger_rules_total").
			Help("Total number of trigger rule evaluations").
			Labels("hook_id", "result").
			BuildVec()

		// 注册所有指标到默认 Prometheus registry
		prometheus.MustRegister(
			HookExecutions,
			HookDuration,
			HTTPRequests,
			HTTPRequestDuration,
			ConcurrentHooks,
			SystemMemoryBytes,
			SystemCPUPercent,
			SystemGoroutines,
			SignatureVerify,
			RateLimitHits,
			TriggerRules,
		)
	})
}

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

// RecordSignatureVerify 记录签名验证结果
// result: "success", "failure", "error"
// algorithm: "sha256", "sha512", "sha1", "md5" 等
func RecordSignatureVerify(result, algorithm string) {
	if SignatureVerify != nil {
		SignatureVerify.WithLabelValues(result, algorithm).Inc()
	}
}

// RecordRateLimitHit 记录限流命中
// scope: "ip", "user", "hook", "global" 等
func RecordRateLimitHit(scope string) {
	if RateLimitHits != nil {
		RateLimitHits.WithLabelValues(scope).Inc()
	}
}

// RecordTriggerRuleEvaluation 记录触发规则评估结果
// result: "matched", "not_matched", "error"
func RecordTriggerRuleEvaluation(hookID, result string) {
	if TriggerRules != nil {
		TriggerRules.WithLabelValues(hookID, result).Inc()
	}
}

// UpdateSystemMetrics 更新系统指标（内存、CPU、goroutine）
func UpdateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 内存指标
	if SystemMemoryBytes != nil {
		SystemMemoryBytes.WithLabelValues("alloc").Set(float64(m.Alloc))
		SystemMemoryBytes.WithLabelValues("sys").Set(float64(m.Sys))
		SystemMemoryBytes.WithLabelValues("total").Set(float64(m.TotalAlloc))
		SystemMemoryBytes.WithLabelValues("heap").Set(float64(m.HeapAlloc))
		SystemMemoryBytes.WithLabelValues("stack").Set(float64(m.StackInuse))
	}

	// Goroutine 数量
	if SystemGoroutines != nil {
		SystemGoroutines.Set(float64(runtime.NumGoroutine()))
	}

	// CPU 使用率（简化版本，基于 GC 时间）
	// 注意：这是一个简化的实现，实际 CPU 使用率需要更复杂的计算
	// 可以使用 gopsutil 等库来获取更准确的 CPU 使用率
	if m.NumGC > 0 && SystemCPUPercent != nil {
		// 使用 GC 时间作为 CPU 使用率的近似值
		cpuPercent := float64(m.PauseTotalNs) / float64(time.Second.Nanoseconds()) * 100
		if cpuPercent > 100 {
			cpuPercent = 100
		}
		SystemCPUPercent.Set(cpuPercent)
	}
}

// StopFunc 是一个用于停止后台任务的函数类型
type StopFunc func()

// StartSystemMetricsCollector 启动系统指标收集器（定期更新）
// 返回一个停止函数，调用它可以停止收集器
func StartSystemMetricsCollector(interval time.Duration) StopFunc {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				UpdateSystemMetrics()
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
	}
}

// --- 新增：基于 metrics-kit 的指标辅助方法 ---

// SignatureMetrics 提供签名验证指标的便捷方法
type SignatureMetrics struct{}

// RecordSuccess 记录成功的签名验证
func (SignatureMetrics) RecordSuccess(algorithm string) {
	RecordSignatureVerify("success", algorithm)
}

// RecordFailure 记录失败的签名验证
func (SignatureMetrics) RecordFailure(algorithm string) {
	RecordSignatureVerify("failure", algorithm)
}

// RecordError 记录签名验证错误
func (SignatureMetrics) RecordError(algorithm string) {
	RecordSignatureVerify("error", algorithm)
}

// RateLimitMetrics 提供限流指标的便捷方法
type RateLimitMetrics struct{}

// RecordIPHit 记录 IP 限流命中
func (RateLimitMetrics) RecordIPHit() {
	RecordRateLimitHit("ip")
}

// RecordUserHit 记录用户限流命中
func (RateLimitMetrics) RecordUserHit() {
	RecordRateLimitHit("user")
}

// RecordHookHit 记录 Hook 限流命中
func (RateLimitMetrics) RecordHookHit() {
	RecordRateLimitHit("hook")
}

// RecordGlobalHit 记录全局限流命中
func (RateLimitMetrics) RecordGlobalHit() {
	RecordRateLimitHit("global")
}

// TriggerRuleMetrics 提供触发规则指标的便捷方法
type TriggerRuleMetrics struct{}

// RecordMatched 记录规则匹配
func (TriggerRuleMetrics) RecordMatched(hookID string) {
	RecordTriggerRuleEvaluation(hookID, "matched")
}

// RecordNotMatched 记录规则不匹配
func (TriggerRuleMetrics) RecordNotMatched(hookID string) {
	RecordTriggerRuleEvaluation(hookID, "not_matched")
}

// RecordError 记录规则评估错误
func (TriggerRuleMetrics) RecordError(hookID string) {
	RecordTriggerRuleEvaluation(hookID, "error")
}

// 全局便捷实例
var (
	Signature   SignatureMetrics
	RateLimit   RateLimitMetrics
	TriggerRule TriggerRuleMetrics
)
