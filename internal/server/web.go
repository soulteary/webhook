package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	healthkit "github.com/soulteary/health-kit"
	loggerkit "github.com/soulteary/logger-kit"
	middlewarekit "github.com/soulteary/middleware-kit"
	versionkit "github.com/soulteary/version-kit"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/link"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/metrics"
	"github.com/soulteary/webhook/internal/middleware"
	"github.com/soulteary/webhook/internal/version"
)

// Server 管理 HTTP 服务器和优雅关闭
type Server struct {
	app      *fiber.App
	listener net.Listener
	mu       sync.Mutex
	shutdown bool
}

// Launch 启动 HTTP 服务器并返回 Server 实例（基于 fiber.App）
func Launch(appFlags flags.AppFlags, addr string, ln net.Listener) *Server {
	// Clean up input
	appFlags.HttpMethods = strings.ToUpper(strings.ReplaceAll(appFlags.HttpMethods, " ", ""))

	bodyLimit := int(appFlags.MaxRequestBodySize)
	if bodyLimit <= 0 {
		bodyLimit = flags.DEFAULT_MAX_REQUEST_BODY_SIZE
	}
	readHeaderTimeout := time.Duration(appFlags.ReadHeaderTimeoutSeconds) * time.Second
	if readHeaderTimeout == 0 {
		readHeaderTimeout = 5 * time.Second
	}
	readTimeout := time.Duration(appFlags.ReadTimeoutSeconds) * time.Second
	if readTimeout == 0 {
		readTimeout = 10 * time.Second
	}
	writeTimeout := time.Duration(appFlags.WriteTimeoutSeconds) * time.Second
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}
	idleTimeout := time.Duration(appFlags.IdleTimeoutSeconds) * time.Second
	if idleTimeout == 0 {
		idleTimeout = 90 * time.Second
	}

	app := fiber.New(fiber.Config{
		BodyLimit:             bodyLimit,
		ReadTimeout:           readTimeout,
		WriteTimeout:          writeTimeout,
		IdleTimeout:           idleTimeout,
		ReadBufferSize:        0,
		WriteBufferSize:       0,
		DisableStartupMessage: true,
	})

	// 安全头中间件（middleware-kit Fiber 版）
	var kitSecurityCfg middlewarekit.SecurityHeadersConfig
	if appFlags.StrictMode {
		kitSecurityCfg = middlewarekit.StrictSecurityHeadersConfig()
	} else {
		kitSecurityCfg = middlewarekit.DefaultSecurityHeadersConfig()
	}
	app.Use(middlewarekit.SecurityHeaders(kitSecurityCfg))

	// logger-kit Fiber 中间件
	if logger.DefaultLogger == nil {
		logger.Init(true, false, "", false)
	}
	loggerCfg := loggerkit.DefaultMiddlewareConfig()
	loggerCfg.Logger = logger.DefaultLogger
	loggerCfg.IncludeRequestID = true
	loggerCfg.RequestIDHeader = "X-Request-Id"
	if appFlags.UseXRequestID {
		loggerCfg.GenerateRequestID = nil
	}
	app.Use(loggerkit.FiberMiddleware(loggerCfg))
	app.Use(recover.New())

	// 限流中间件（适配 Std 中间件）
	if appFlags.RateLimitEnabled {
		rateLimitConfig := middleware.RateLimitConfig{
			Enabled:        appFlags.RateLimitEnabled,
			RPS:            appFlags.RateLimitRPS,
			Burst:          appFlags.RateLimitBurst,
			RedisEnabled:   appFlags.RedisEnabled,
			RedisAddr:      appFlags.RedisAddr,
			RedisPassword:  appFlags.RedisPassword,
			RedisDB:        appFlags.RedisDB,
			RedisKeyPrefix: appFlags.RedisKeyPrefix,
			WindowSeconds:  appFlags.RateLimitWindowSec,
		}
		app.Use(adaptor.HTTPMiddleware(middleware.NewRateLimitMiddleware(rateLimitConfig)))
		if appFlags.RedisEnabled {
			logger.Infof("rate limiting enabled with Redis: %d RPS, burst: %d, window: %ds, Redis: %s",
				appFlags.RateLimitRPS, appFlags.RateLimitBurst, appFlags.RateLimitWindowSec, appFlags.RedisAddr)
		} else {
			logger.Infof("rate limiting enabled (in-memory): %d RPS, burst: %d", appFlags.RateLimitRPS, appFlags.RateLimitBurst)
		}
	}

	if appFlags.Debug {
		dumperConfig := middleware.DumperConfig{
			IncludeRequestBody: appFlags.LogRequestBody,
		}
		app.Use(adaptor.HTTPMiddleware(middleware.DumperWithConfig(logger.Writer(), dumperConfig)))
	}

	// 健康检查聚合器
	healthConfig := healthkit.DefaultConfig().
		WithServiceName("webhook").
		WithTimeout(5 * time.Second).
		WithDetails(true).
		WithChecks(true)
	healthAggregator := healthkit.NewAggregator(healthConfig)

	var serverRef *Server

	healthAggregator.AddChecker(healthkit.NewCustomChecker("service", func(ctx context.Context) error {
		if serverRef != nil && serverRef.IsShuttingDown() {
			return fmt.Errorf("server is shutting down")
		}
		return nil
	}).WithMetadata(map[string]any{
		"component": "webhook-server",
	}))

	if appFlags.RedisEnabled {
		healthAggregator.AddChecker(healthkit.NewCustomChecker("redis", func(ctx context.Context) error {
			return nil
		}).WithMetadata(map[string]any{
			"addr": appFlags.RedisAddr,
		}))
	}

	// health / livez / readyz / version / metrics / 根路径：HTTP -> Fiber 适配器
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := healthkit.Handler(healthAggregator)
		handler(w, r)
		duration := time.Since(startTime)
		result := healthAggregator.Check(r.Context())
		statusCode := healthkit.HTTPStatusCode(result.Status)
		metrics.RecordHTTPRequest(r.Method, fmt.Sprintf("%d", statusCode), "/health", duration)
	}
	app.All("/health", adaptor.HTTPHandlerFunc(healthHandler))

	livezHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := healthkit.LivenessHandler("webhook")
		handler(w, r)
		metrics.RecordHTTPRequest(r.Method, "200", "/livez", time.Since(startTime))
	}
	app.All("/livez", adaptor.HTTPHandlerFunc(livezHandler))

	readyzHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := healthkit.ReadinessHandler(healthAggregator)
		handler(w, r)
		duration := time.Since(startTime)
		result := healthAggregator.Check(r.Context())
		statusCode := healthkit.HTTPStatusCode(result.Status)
		metrics.RecordHTTPRequest(r.Method, fmt.Sprintf("%d", statusCode), "/readyz", duration)
	}
	app.All("/readyz", adaptor.HTTPHandlerFunc(readyzHandler))

	versionInfo := version.GetVersionInfo()
	versionConfig := versionkit.HandlerConfig{
		Info:           versionInfo,
		Pretty:         true,
		IncludeHeaders: true,
		HeaderPrefix:   "X-",
	}
	versionHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := versionkit.Handler(versionConfig)
		handler(w, r)
		metrics.RecordHTTPRequest(r.Method, "200", "/version", time.Since(startTime))
	}
	app.All("/version", adaptor.HTTPHandlerFunc(versionHandler))

	app.All("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		defer func() {
			metrics.RecordHTTPRequest(r.Method, "200", "/", time.Since(startTime))
		}()
		setResponseHeaders(w, appFlags.ResponseHeaders)
		fmt.Fprint(w, "OK")
	}
	app.All("/", adaptor.HTTPHandlerFunc(rootHandler))

	s := &Server{
		app:      app,
		listener: ln,
	}
	serverRef = s

	// Hook 路由：通过适配器调用现有 createHookHandler
	hookHandler := createHookHandler(appFlags, s)
	hookBase := link.MakeBaseURL(&appFlags.HooksURLPrefix)
	if hookBase == "" {
		hookBase = "/hooks"
	}
	app.All(hookBase+"/:id", adaptor.HTTPHandlerFunc(hookHandler))
	app.All(hookBase+"/:id/*", adaptor.HTTPHandlerFunc(hookHandler))

	metrics.StartSystemMetricsCollector(10 * time.Second)

	go func() {
		logger.Infof("serving hooks on http://%s%s", addr, link.MakeHumanPattern(&appFlags.HooksURLPrefix))
		logger.Infof("health check endpoints: http://%s/health, http://%s/livez, http://%s/readyz", addr, addr, addr)
		logger.Infof("version endpoint: http://%s/version", addr)
		logger.Infof("metrics endpoint: http://%s/metrics", addr)
		if err := app.Listener(ln); err != nil {
			logger.Error(fmt.Sprintf("server error: %v", err))
		}
	}()

	return s
}

// Shutdown 优雅关闭服务器：先等异步 hook WaitGroup，再关闭 Fiber
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return nil
	}
	s.shutdown = true
	s.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		GetAsyncHookWaitGroup().Wait()
		done <- s.app.Shutdown()
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
