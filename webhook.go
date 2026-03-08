package main

import (
	"context"
	"embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/soulteary/webhook/internal/audit"
	"github.com/soulteary/webhook/internal/configui"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/i18n"
	"github.com/soulteary/webhook/internal/link"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/monitor"
	"github.com/soulteary/webhook/internal/openapi"
	"github.com/soulteary/webhook/internal/pidfile"
	"github.com/soulteary/webhook/internal/platform"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/soulteary/webhook/internal/server"
	"github.com/soulteary/webhook/internal/tracing"
	"github.com/soulteary/webhook/internal/version"
)

var (
	signals    chan os.Signal
	pidFile    *pidfile.PIDFile
	httpServer *server.Server
)

//go:embed locales/*.yaml
var WebhookLocales embed.FS

func NeedEchoVersionInfo(appFlags flags.AppFlags) {
	if appFlags.ShowVersion {
		i18n.Println(i18n.MSG_WEBHOOK_VERSION, version.Version)
		os.Exit(0)
	}
}

// prepareHooksFilesAndValidate 加锁检查并填充默认 HooksFiles，然后执行配置验证，供 NeedValidateConfig 与 main 共用。
// 当 -hooks-dir 已设置时，允许 HooksFiles 为空（目录扫描结果或待监控的空目录）。
func prepareHooksFilesAndValidate(appFlags flags.AppFlags) *flags.ValidationResult {
	rules.LockHooksFiles()
	if len(rules.HooksFiles) == 0 && appFlags.HooksDir == "" {
		rules.HooksFiles = append(rules.HooksFiles, "hooks.json")
	}
	rules.UnlockHooksFiles()
	return flags.Validate(appFlags)
}

func NeedValidateConfig(appFlags flags.AppFlags) {
	if appFlags.ValidateConfig {
		validationResult := prepareHooksFilesAndValidate(appFlags)
		if validationResult.HasErrors() {
			fmt.Fprintf(os.Stderr, "%s\n", i18n.Sprintf(i18n.MSG_CONFIG_VALIDATION_FAILED, len(validationResult.Errors)))
			for _, err := range validationResult.Errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", err.Error())
			}
			os.Exit(1)
		}
		fmt.Println(i18n.Sprintf(i18n.MSG_CONFIG_VALIDATION_PASSED))
		os.Exit(0)
	}
}

func CheckPrivilegesParamsCorrect(appFlags flags.AppFlags) {
	if (appFlags.SetUID != 0 || appFlags.SetGID != 0) && (appFlags.SetUID == 0 || appFlags.SetGID == 0) {
		i18n.Println(i18n.MSG_SETUID_OR_SETGID_ERROR)
		os.Exit(1)
	}
}

func GetNetAddr(appFlags flags.AppFlags, logQueue *[]string) (string, *net.Listener) {
	addr := fmt.Sprintf("%s:%d", appFlags.Host, appFlags.Port)
	// Open listener early so we can drop privileges.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		*logQueue = append(*logQueue, i18n.Sprintf(i18n.ERR_SERVER_LISTENING_PORT, err))
	}
	return addr, &ln
}

func DropPrivileges(appFlags flags.AppFlags, logQueue *[]string) {
	if appFlags.SetUID != 0 {
		err := platform.DropPrivileges(appFlags.SetUID, appFlags.SetGID)
		if err != nil {
			*logQueue = append(*logQueue, i18n.Sprintf(i18n.ERR_SERVER_LISTENING_PRIVILEGES, err))
		}
	}
}

func SetupLogger(appFlags flags.AppFlags, logQueue *[]string) error {
	// 初始化日志系统
	err := logger.Init(appFlags.Verbose, appFlags.Debug, appFlags.LogPath, false)
	if err != nil {
		*logQueue = append(*logQueue, i18n.Sprintf(i18n.ERR_SERVER_OPENING_LOG_FILE, appFlags.LogPath, err))
		return err
	}
	return nil
}

// runConfigUIOnly starts only the Config UI HTTP server (no webhook server). Used when -config-ui is set and no -hooks are provided.
func runConfigUIOnly(appFlags flags.AppFlags) {
	port := strconv.Itoa(appFlags.Port)
	webhookBaseURL := "http://localhost:" + port
	hookBase := link.MakeBaseURL(&appFlags.HooksURLPrefix)
	if hookBase == "" {
		hookBase = "/hooks"
	}
	handler, err := configui.Handler("/", webhookBaseURL, appFlags.HooksDir, hookBase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config-ui init: %v\n", err)
		os.Exit(1)
	}
	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		}
	}()
	fmt.Printf("Webhook Config UI: http://localhost%s\n", addr)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	appFlags := flags.Parse()

	if err := i18n.InitLocaleByFiles(appFlags.I18nDir, WebhookLocales); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize i18n: %v\n", err)
		os.Exit(1)
	}
	i18n.SetGlobalLocale(appFlags.Lang)

	// check if we need to echo version info and quit app
	NeedEchoVersionInfo(appFlags)

	// Config UI only mode: no hooks and config-ui enabled — start only Config UI server
	if appFlags.ConfigUIEnabled && len(appFlags.HooksFiles) == 0 {
		runConfigUIOnly(appFlags)
		return
	}

	// check if we need to validate config and quit app
	NeedValidateConfig(appFlags)
	// check if the privileges params are correct, or exit(1)
	CheckPrivilegesParamsCorrect(appFlags)

	if appFlags.Debug || appFlags.LogPath != "" {
		appFlags.Verbose = true
	}

	validationResult := prepareHooksFilesAndValidate(appFlags)
	if validationResult.HasErrors() {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.Sprintf(i18n.MSG_CONFIG_VALIDATION_FAILED, len(validationResult.Errors)))
		for _, err := range validationResult.Errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", err.Error())
		}
		os.Exit(1)
	}

	// logQueue is a queue for log messages encountered during startup. We need
	// to queue the messages so that we can handle any privilege dropping and
	// log file opening prior to writing our first log message.
	var logQueue []string

	// set up net listener and get listening address
	addr, ln := GetNetAddr(appFlags, &logQueue)
	// drop privileges
	DropPrivileges(appFlags, &logQueue)
	// setup logger
	err := SetupLogger(appFlags, &logQueue)
	if err != nil {
		// 如果日志初始化失败，使用标准输出输出错误信息
		for i := range logQueue {
			fmt.Fprintln(os.Stderr, logQueue[i])
		}
		os.Exit(1)
	}

	if len(logQueue) != 0 {
		for i := range logQueue {
			logger.Error(logQueue[i])
		}
		os.Exit(1)
	}

	// Create pidfile
	if appFlags.PidPath != "" {
		var err error

		pidFile, err = pidfile.New(appFlags.PidPath)
		if err != nil {
			logger.Fatalf("%s %v", i18n.ERR_CREATING_PID_FILE, err)
		}

		defer func() {
			// NOTE(moorereason): my testing shows that this doesn't work with
			// ^C, so we also do a Remove in the signal handler elsewhere.
			if nerr := pidFile.Remove(); nerr != nil {
				logger.Error(fmt.Sprintf("%v", nerr))
			}
		}()
	}

	logger.Info(i18n.Sprintf(i18n.MSG_SERVER_IS_STARTING, version.Version))

	// 初始化追踪系统
	if appFlags.TracingEnabled {
		tracingConfig := tracing.TracingConfig{
			Enabled:        appFlags.TracingEnabled,
			ServiceName:    appFlags.TracingServiceName,
			ServiceVersion: version.Version,
			OTLPEndpoint:   appFlags.OTLPEndpoint,
		}
		if err := tracing.Init(tracingConfig); err != nil {
			logger.Warnf("failed to initialize tracing: %v", err)
		} else {
			logger.Infof("tracing enabled: service=%s, endpoint=%s", appFlags.TracingServiceName, appFlags.OTLPEndpoint)
		}
	}

	// 初始化审计日志系统
	if appFlags.AuditEnabled {
		if err := audit.Init(appFlags); err != nil {
			logger.Warnf("failed to initialize audit logging: %v", err)
		}
	}

	// load and parse hooks
	rules.ParseAndLoadHooks(appFlags.AsTemplate)

	// 使用 -hooks-dir 时允许暂时无 hook（空目录或待监控）
	if !appFlags.Verbose && !appFlags.NoPanic && rules.LenLoadedHooks() == 0 && appFlags.HooksDir == "" {
		logger.Fatalln(i18n.Sprintf(i18n.ERR_COULD_NOT_LOAD_ANY_HOOKS))
	}

	// -hotreload 或 -hooks-dir 时启动监控（-hooks-dir 时始终监控目录）
	if appFlags.HotReload || appFlags.HooksDir != "" {
		monitor.ApplyWatcher(appFlags)
	}

	if appFlags.OpenAPIEnabled && appFlags.OpenAPIPrint {
		spec, err := openapi.Spec(appFlags, "http://"+addr)
		if err != nil {
			logger.Warnf("openapi spec generation failed: %v", err)
		} else {
			fmt.Print(string(spec))
		}
	}

	// 启动服务器
	httpServer = server.Launch(appFlags, addr, *ln)

	// 设置优雅关闭回调
	shutdownFn := func() {
		// 关闭审计日志系统
		if audit.IsEnabled() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := audit.Shutdown(ctx); err != nil {
				logger.Warnf("error shutting down audit logging: %v", err)
			}
		}

		// 关闭追踪系统
		if tracing.IsEnabled() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracing.Shutdown(ctx); err != nil {
				logger.Warnf("error shutting down tracing: %v", err)
			} else {
				logger.Info("tracing shutdown completed")
			}
		}

		if httpServer != nil {
			// 设置最大等待时间为 30 秒
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(ctx); err != nil {
				logger.Errorf("error during graceful shutdown: %v", err)
			}
		}
	}

	// set os signal watcher with shutdown callback
	if appFlags.AsTemplate {
		signals = platform.SetupSignalsWithShutdown(signals, rules.ReloadAllHooksAsTemplate, shutdownFn, pidFile, nil)
	} else {
		signals = platform.SetupSignalsWithShutdown(signals, rules.ReloadAllHooksNotAsTemplate, shutdownFn, pidFile, nil)
	}

	// 等待服务器关闭
	select {}
}
