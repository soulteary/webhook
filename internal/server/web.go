package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/mux"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/link"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/metrics"
	"github.com/soulteary/webhook/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server 管理 HTTP 服务器和优雅关闭
type Server struct {
	server   *http.Server
	listener net.Listener
	mu       sync.Mutex
	shutdown bool
}

// Launch 启动 HTTP 服务器并返回 Server 实例
func Launch(appFlags flags.AppFlags, addr string, ln net.Listener) *Server {
	r := mux.NewRouter()

	r.Use(middleware.RequestID(
		middleware.UseXRequestIDHeaderOption(appFlags.UseXRequestID),
		middleware.XRequestIDLimitOption(appFlags.XRequestIDLimit),
	))
	r.Use(middleware.NewLogger())
	r.Use(chimiddleware.Recoverer)

	// 添加限流中间件（如果启用）
	if appFlags.RateLimitEnabled {
		rateLimitConfig := middleware.RateLimitConfig{
			Enabled: appFlags.RateLimitEnabled,
			RPS:     appFlags.RateLimitRPS,
			Burst:   appFlags.RateLimitBurst,
		}
		r.Use(middleware.NewRateLimitMiddleware(rateLimitConfig))
		logger.Infof("rate limiting enabled: %d RPS, burst: %d", appFlags.RateLimitRPS, appFlags.RateLimitBurst)
	}

	if appFlags.Debug {
		dumperConfig := middleware.DumperConfig{
			IncludeRequestBody: appFlags.LogRequestBody,
		}
		r.Use(middleware.DumperWithConfig(logger.Writer(), dumperConfig))
	}

	// Clean up input
	appFlags.HttpMethods = strings.ToUpper(strings.ReplaceAll(appFlags.HttpMethods, " ", ""))

	hooksURL := link.MakeRoutePattern(&appFlags.HooksURLPrefix)

	// 健康检查端点
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		defer func() {
			duration := time.Since(startTime)
			metrics.RecordHTTPRequest(req.Method, "200", "/health", duration)
		}()
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Prometheus metrics 端点
	r.Handle("/metrics", promhttp.Handler())

	// 根路径
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		defer func() {
			duration := time.Since(startTime)
			metrics.RecordHTTPRequest(req.Method, "200", "/", duration)
		}()
		
		setResponseHeaders(w, appFlags.ResponseHeaders)
		fmt.Fprint(w, "OK")
	})

	// Create common HTTP server settings
	svr := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
	}

	s := &Server{
		server:   svr,
		listener: ln,
	}

	hookHandler := createHookHandler(appFlags, s)
	r.HandleFunc(hooksURL, hookHandler)

	// 启动系统指标收集器（每 10 秒更新一次）
	metrics.StartSystemMetricsCollector(10 * time.Second)

	// Serve HTTP in a goroutine
	go func() {
		logger.Infof("serving hooks on http://%s%s", addr, link.MakeHumanPattern(&appFlags.HooksURLPrefix))
		logger.Infof("health check endpoint: http://%s/health", addr)
		logger.Infof("metrics endpoint: http://%s/metrics", addr)
		if err := svr.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("server error: %v", err))
		}
	}()

	return s
}

// Shutdown 优雅关闭服务器
// 1. 停止接受新请求
// 2. 等待正在执行的 hook 完成
// 3. 设置最大等待时间
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return nil
	}
	s.shutdown = true
	s.mu.Unlock()

	// 停止接受新请求
	s.server.SetKeepAlivesEnabled(false)

	// 等待正在执行的 hook 完成
	done := make(chan error, 1)
	go func() {
		// 等待所有异步执行的 hook goroutine 完成
		GetAsyncHookWaitGroup().Wait()
		// 关闭 HTTP 服务器
		done <- s.server.Shutdown(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Errorf("error during server shutdown: %v", err)
		} else {
			logger.Info("server shutdown completed gracefully")
		}
		return err
	case <-ctx.Done():
		logger.Warnf("server shutdown timeout: %v", ctx.Err())
		return ctx.Err()
	}
}

// IsShuttingDown 检查服务器是否正在关闭
func (s *Server) IsShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdown
}
