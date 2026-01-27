package main

import (
	"context"
	"embed"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/soulteary/webhook/internal/audit"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/i18n"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/monitor"
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

//go:embed locales/*.toml
var WebhookLocales embed.FS

func NeedEchoVersionInfo(appFlags flags.AppFlags) {
	if appFlags.ShowVersion {
		i18n.Println(i18n.MSG_WEBHOOK_VERSION, version.Version)
		os.Exit(0)
	}
}

func NeedValidateConfig(appFlags flags.AppFlags) {
	if appFlags.ValidateConfig {
		// 加锁检查和更新 HooksFiles（与 main 函数中的逻辑一致）
		rules.LockHooksFiles()
		if len(rules.HooksFiles) == 0 {
			rules.HooksFiles = append(rules.HooksFiles, "hooks.json")
		}
		rules.UnlockHooksFiles()

		// 验证配置
		validationResult := flags.Validate(appFlags)
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

func main() {
	appFlags := flags.Parse()

	if err := i18n.InitLocaleByFiles(appFlags.I18nDir, WebhookLocales); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize i18n: %v\n", err)
		os.Exit(1)
	}
	i18n.SetGlobalLocale(appFlags.Lang)

	// check if we need to echo version info and quit app
	NeedEchoVersionInfo(appFlags)
	// check if we need to validate config and quit app
	NeedValidateConfig(appFlags)
	// check if the privileges params are correct, or exit(1)
	CheckPrivilegesParamsCorrect(appFlags)

	if appFlags.Debug || appFlags.LogPath != "" {
		appFlags.Verbose = true
	}

	// 加锁检查和更新 HooksFiles
	rules.LockHooksFiles()
	if len(rules.HooksFiles) == 0 {
		rules.HooksFiles = append(rules.HooksFiles, "hooks.json")
	}
	rules.UnlockHooksFiles()

	// 验证配置
	validationResult := flags.Validate(appFlags)
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

	if !appFlags.Verbose && !appFlags.NoPanic && rules.LenLoadedHooks() == 0 {
		logger.Fatalln(i18n.Sprintf(i18n.ERR_COULD_NOT_LOAD_ANY_HOOKS))
	}

	if appFlags.HotReload {
		monitor.ApplyWatcher(appFlags)
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
