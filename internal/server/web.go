package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	rediskit "github.com/soulteary/redis-kit/client"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	healthkit "github.com/soulteary/health-kit/v4"
	loggerkit "github.com/soulteary/logger-kit/v3"
	loggerfiber "github.com/soulteary/logger-kit/v3/fiberadapter"
	middlewarekit "github.com/soulteary/middleware-kit/v3"
	mwfiber "github.com/soulteary/middleware-kit/v3/fiberadapter"
	versionkit "github.com/soulteary/version-kit/v4"
	versionhttp "github.com/soulteary/version-kit/v4/httpadapter"
	"github.com/soulteary/webhook/internal/configui"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/link"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/metrics"
	"github.com/soulteary/webhook/internal/middleware"
	"github.com/soulteary/webhook/internal/openapi"
	"github.com/soulteary/webhook/internal/version"
)

// Server 管理 HTTP 服务器和优雅关闭
type Server struct {
	app      *fiber.App
	listener net.Listener
	mu       sync.Mutex
	shutdown bool
}

func registerReadOnlyRoute(app *fiber.App, path string, handler fiber.Handler) {
	app.Get(path, handler)
	app.Head(path, handler)
	app.All(path, func(c fiber.Ctx) error {
		startTime := time.Now()
		method := strings.Clone(c.Method())
		defer func() {
			metrics.RecordHTTPRequest(method, "405", path, time.Since(startTime))
		}()
		c.Set("Allow", "GET, HEAD")
		return c.Status(http.StatusMethodNotAllowed).SendString(http.StatusText(http.StatusMethodNotAllowed))
	})
}

// requestIDPolicy 在日志中间件之前约束客户端传入的 X-Request-Id：
// 未启用 -x-request-id 时丢弃客户端的值（由服务端生成），启用时按 limit 截断。
func requestIDPolicy(useHeader bool, limit int) fiber.Handler {
	const header = "X-Request-Id"
	return func(c fiber.Ctx) error {
		if !useHeader {
			c.Request().Header.Del(header)
			return c.Next()
		}
		if id := c.Get(header); limit > 0 && len(id) > limit {
			c.Request().Header.Set(header, id[:limit])
		}
		return c.Next()
	}
}

// newRedisHealthCheck 返回真正向 Redis 发 PING 的健康检查；连接按需建立并复用。
func newRedisHealthCheck(appFlags flags.AppFlags) func(ctx context.Context) error {
	var (
		mu     sync.Mutex
		client *redis.Client
	)
	return func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if client == nil {
			cfg := rediskit.DefaultConfig().
				WithAddr(appFlags.RedisAddr).
				WithPassword(appFlags.RedisPassword).
				WithDB(appFlags.RedisDB)
			c, err := rediskit.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("redis unavailable: %w", err)
			}
			client = c
		}
		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis ping failed: %w", err)
		}
		return nil
	}
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
	// fasthttp 没有单独的请求头读取超时，请求头与请求体共用 ReadTimeout。
	// 用户显式改过这个参数时明确告知，而不是静默忽略。
	if appFlags.ReadHeaderTimeoutSeconds != flags.DEFAULT_READ_HEADER_TIMEOUT_SECONDS && readHeaderTimeout < time.Duration(appFlags.ReadTimeoutSeconds)*time.Second {
		logger.Warnf("-read-header-timeout-seconds=%d has no effect: request headers are bounded by -read-timeout-seconds", appFlags.ReadHeaderTimeoutSeconds)
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

	// fasthttp 用读缓冲区大小限制请求头。默认值（1MB）时保留 fasthttp 的 4KB 缓冲，
	// 避免每个连接都分配 1MB；只有显式配置了其他值才生效。
	readBufferSize := 0
	if appFlags.MaxHeaderBytes > 0 && appFlags.MaxHeaderBytes != flags.DEFAULT_MAX_HEADER_BYTES {
		readBufferSize = appFlags.MaxHeaderBytes
	}

	app := fiber.New(fiber.Config{
		BodyLimit:       bodyLimit,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ReadBufferSize:  readBufferSize,
		WriteBufferSize: 0,
	})

	// 安全头中间件（middleware-kit Fiber 版）
	var kitSecurityCfg middlewarekit.SecurityHeadersConfig
	if appFlags.StrictMode {
		kitSecurityCfg = middlewarekit.StrictSecurityHeadersConfig()
	} else {
		kitSecurityCfg = middlewarekit.DefaultSecurityHeadersConfig()
	}
	app.Use(mwfiber.SecurityHeaders(kitSecurityCfg))

	// logger-kit Fiber 中间件
	if logger.DefaultLogger == nil {
		_ = logger.Init(true, false, "", false)
	}
	loggerCfg := loggerkit.DefaultMiddlewareConfig()
	loggerCfg.Logger = logger.DefaultLogger
	loggerCfg.IncludeRequestID = true
	loggerCfg.RequestIDHeader = "X-Request-Id"
	if appFlags.UseXRequestID {
		loggerCfg.GenerateRequestID = nil
	}
	app.Use(requestIDPolicy(appFlags.UseXRequestID, appFlags.XRequestIDLimit))
	app.Use(loggerfiber.Middleware(loggerfiber.Config{MiddlewareConfig: loggerCfg}))
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
			PerIPRPS:       appFlags.RateLimitPerIPRPS,
			PerIPBurst:     appFlags.RateLimitPerIPBurst,
			TrustedProxies: strings.Split(appFlags.TrustedProxies, ","),
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
		healthAggregator.AddChecker(healthkit.NewCustomChecker("redis", newRedisHealthCheck(appFlags)).WithMetadata(map[string]any{
			"addr": appFlags.RedisAddr,
		}))
	}

	// health / livez / readyz / version / metrics / 根路径：HTTP -> Fiber 适配器
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := healthkit.Handler(healthAggregator)
		// The fasthttp adapter's request context is backed by server-owned state.
		// Health checks may derive child contexts, so detach them before shutdown
		// can recycle or mutate that state.
		detachedRequest := r.Clone(context.Background())
		handler(w, detachedRequest)
		duration := time.Since(startTime)
		result := healthAggregator.Check(context.Background())
		statusCode := healthkit.HTTPStatusCode(result.Status)
		metrics.RecordHTTPRequest(r.Method, fmt.Sprintf("%d", statusCode), "/health", duration)
	}
	registerReadOnlyRoute(app, "/health", adaptor.HTTPHandlerFunc(healthHandler))

	livezHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := healthkit.LivenessHandler("webhook")
		handler(w, r)
		metrics.RecordHTTPRequest(r.Method, "200", "/livez", time.Since(startTime))
	}
	registerReadOnlyRoute(app, "/livez", adaptor.HTTPHandlerFunc(livezHandler))

	readyzHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := healthkit.ReadinessHandler(healthAggregator)
		detachedRequest := r.Clone(context.Background())
		handler(w, detachedRequest)
		duration := time.Since(startTime)
		result := healthAggregator.Check(context.Background())
		statusCode := healthkit.HTTPStatusCode(result.Status)
		metrics.RecordHTTPRequest(r.Method, fmt.Sprintf("%d", statusCode), "/readyz", duration)
	}
	registerReadOnlyRoute(app, "/readyz", adaptor.HTTPHandlerFunc(readyzHandler))

	versionInfo := version.GetVersionInfo()
	versionConfig := versionkit.HandlerConfig{
		Info:           versionInfo,
		Pretty:         true,
		IncludeHeaders: true,
		HeaderPrefix:   "X-",
		// version-kit v2.2.0 made the build details opt-in. /version is
		// documented in the OpenAPI spec with go_version, platform and
		// compiler as required, so keep serving them.
		IncludeBuildDetails: true,
	}
	versionHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		handler := versionhttp.Handler(versionConfig)
		handler(w, r)
		metrics.RecordHTTPRequest(r.Method, "200", "/version", time.Since(startTime))
	}
	registerReadOnlyRoute(app, "/version", adaptor.HTTPHandlerFunc(versionHandler))

	registerReadOnlyRoute(app, "/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		defer func() {
			metrics.RecordHTTPRequest(r.Method, "200", "/", time.Since(startTime))
		}()
		setResponseHeaders(w, appFlags.ResponseHeaders)
		_, _ = fmt.Fprint(w, "OK")
	}
	registerReadOnlyRoute(app, "/", adaptor.HTTPHandlerFunc(rootHandler))

	var openapiPathLogged string
	if appFlags.OpenAPIEnabled {
		openapiPath := strings.TrimSpace(appFlags.OpenAPIPath)
		if openapiPath == "" || !strings.HasPrefix(openapiPath, "/") {
			openapiPath = "/openapi"
		}
		hookBaseForReserved := link.MakeBaseURL(&appFlags.HooksURLPrefix)
		if hookBaseForReserved == "" {
			hookBaseForReserved = "/hooks"
		}
		reservedPaths := []string{"/", "/health", "/livez", "/readyz", "/version", "/metrics", hookBaseForReserved}
		isReserved := false
		for _, p := range reservedPaths {
			if openapiPath == p || (p != "/" && strings.HasPrefix(openapiPath, p+"/")) {
				isReserved = true
				break
			}
		}
		if isReserved {
			logger.Warnf("openapi-path %q conflicts with reserved path; skipping OpenAPI route", openapiPath)
		} else {
			specJSON, err := openapi.Spec(appFlags, "http://"+addr)
			if err != nil {
				logger.Warnf("openapi spec generation failed: %v", err)
			} else {
				body := specJSON
				app.Get(openapiPath, func(c fiber.Ctx) error {
					c.Set("Content-Type", "application/json; charset=utf-8")
					return c.Send(body)
				})
				openapiPathLogged = openapiPath
			}
		}
	}

	var configUIPathLogged string
	if appFlags.ConfigUIEnabled {
		configUIPath := strings.TrimSpace(appFlags.ConfigUIPath)
		if configUIPath == "" || !strings.HasPrefix(configUIPath, "/") {
			configUIPath = "/config-ui"
		}
		configUIPath = strings.TrimSuffix(configUIPath, "/")
		if configUIPath == "" {
			configUIPath = "/"
		}
		hookBaseForReserved := link.MakeBaseURL(&appFlags.HooksURLPrefix)
		if hookBaseForReserved == "" {
			hookBaseForReserved = "/hooks"
		}
		reservedPaths := []string{"/", "/health", "/livez", "/readyz", "/version", "/metrics", hookBaseForReserved}
		if openapiPathLogged != "" {
			reservedPaths = append(reservedPaths, openapiPathLogged)
		}
		isReserved := false
		for _, p := range reservedPaths {
			if configUIPath == p || (p != "/" && strings.HasPrefix(configUIPath, p+"/")) {
				isReserved = true
				break
			}
		}
		if isReserved {
			logger.Warnf("config-ui-path %q conflicts with reserved path; skipping Config UI route", configUIPath)
		} else {
			configUIHandler, err := configui.HandlerWithSaveToken(configUIPath, "http://"+addr, appFlags.HooksDir, hookBaseForReserved, appFlags.ConfigUIToken)
			if err != nil {
				logger.Warnf("config-ui handler init failed: %v", err)
			} else {
				app.All(configUIPath, adaptor.HTTPHandler(configUIHandler))
				app.All(configUIPath+"/*", adaptor.HTTPHandler(configUIHandler))
				configUIPathLogged = configUIPath
			}
		}
	}

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
		if openapiPathLogged != "" {
			logger.Infof("openapi spec: http://%s%s", addr, openapiPathLogged)
		}
		if configUIPathLogged != "" {
			logger.Infof("config UI: http://%s%s", addr, configUIPathLogged)
			if strings.TrimSpace(appFlags.ConfigUIToken) == "" {
				logger.Infof("config UI: save-to-directory API is disabled (set -config-ui-token or CONFIG_UI_TOKEN to enable it)")
			}
		}
		if err := app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
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
