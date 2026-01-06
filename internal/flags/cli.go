package flags

import (
	"flag"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
)

func ParseCLI(flags AppFlags) AppFlags {
	var (
		Host               = flag.String("ip", DEFAULT_HOST, "ip the webhook should serve hooks on")
		Port               = flag.Int("port", DEFAULT_PORT, "port the webhook should serve hooks on")
		Verbose            = flag.Bool("verbose", DEFAULT_ENABLE_VERBOSE, "show verbose output")
		LogPath            = flag.String("logfile", DEFAULT_LOG_PATH, "send log output to a file; implicitly enables verbose logging")
		Debug              = flag.Bool("debug", DEFAULT_ENABLE_DEBUG, "show debug output")
		NoPanic            = flag.Bool("nopanic", DEFAULT_ENABLE_NO_PANIC, "do not panic if hooks cannot be loaded when webhook is not running in verbose mode")
		HotReload          = flag.Bool("hotreload", DEFAULT_ENABLE_HOT_RELOAD, "watch hooks file for changes and reload them automatically")
		HooksURLPrefix     = flag.String("urlprefix", DEFAULT_URL_PREFIX, "url prefix to use for served hooks (protocol://yourserver:port/PREFIX/:hook-id)")
		AsTemplate         = flag.Bool("template", DEFAULT_ENABLE_PARSE_TEMPLATE, "parse hooks file as a Go template")
		UseXRequestID      = flag.Bool("x-request-id", DEFAULT_ENABLE_X_REQUEST_ID, "use X-Request-Id header, if present, as request ID")
		XRequestIDLimit    = flag.Int("x-request-id-limit", DEFAULT_X_REQUEST_ID_LIMIT, "truncate X-Request-Id header to limit; default no limit")
		MaxMultipartMem    = flag.Int64("max-multipart-mem", DEFAULT_MAX_MPART_MEM, "maximum memory in bytes for parsing multipart form data before disk caching")
		MaxRequestBodySize = flag.Int64("max-request-body-size", DEFAULT_MAX_REQUEST_BODY_SIZE, "maximum size in bytes for request body (default 10MB)")
		SetGID             = flag.Int("setgid", DEFAULT_GID, "set group ID after opening listening port; must be used with setuid")
		SetUID             = flag.Int("setuid", DEFAULT_UID, "set user ID after opening listening port; must be used with setgid")
		HttpMethods        = flag.String("http-methods", DEFAULT_HTTP_METHODS, `set default allowed HTTP methods (ie. "POST"); separate methods with comma`)
		PidPath            = flag.String("pidfile", DEFAULT_PID_FILE, "create PID file at the given path")

		Lang    = flag.String("lang", DEFAULT_LANG, "set the language code for the webhook")
		I18nDir = flag.String("lang-dir", DEFAULT_I18N_DIR, "set the directory for the i18n files")

		HookTimeoutSeconds   = flag.Int("hook-timeout-seconds", DEFAULT_HOOK_TIMEOUT_SECONDS, "default timeout in seconds for hook execution (default 30)")
		MaxConcurrentHooks   = flag.Int("max-concurrent-hooks", DEFAULT_MAX_CONCURRENT_HOOKS, "maximum number of concurrent hook executions (default 10)")
		HookExecutionTimeout = flag.Int("hook-execution-timeout", DEFAULT_HOOK_EXECUTION_TIMEOUT, "timeout in seconds for acquiring execution slot when max concurrent hooks reached (default 5)")
		AllowAutoChmod       = flag.Bool("allow-auto-chmod", DEFAULT_ALLOW_AUTO_CHMOD, "allow automatically modifying file permissions when permission denied (SECURITY RISK: default false)")

		// Security flags
		AllowedCommandPaths = flag.String("allowed-command-paths", DEFAULT_ALLOWED_COMMAND_PATHS, "comma-separated list of allowed command paths (directories or files) for command execution whitelist; empty means no whitelist check")
		MaxArgLength        = flag.Int("max-arg-length", DEFAULT_MAX_ARG_LENGTH, "maximum length for a single command argument in bytes (default 1MB)")
		MaxTotalArgsLength  = flag.Int("max-total-args-length", DEFAULT_MAX_TOTAL_ARGS_LENGTH, "maximum total length for all command arguments in bytes (default 10MB)")
		MaxArgsCount        = flag.Int("max-args-count", DEFAULT_MAX_ARGS_COUNT, "maximum number of command arguments (default 1000)")
		StrictMode          = flag.Bool("strict-mode", DEFAULT_STRICT_MODE, "strict mode: reject arguments containing potentially dangerous characters (default false)")

		// Rate limiting flags
		RateLimitEnabled = flag.Bool("rate-limit-enabled", DEFAULT_RATE_LIMIT_ENABLED, "enable rate limiting (default false)")
		RateLimitRPS     = flag.Int("rate-limit-rps", DEFAULT_RATE_LIMIT_RPS, "rate limit requests per second (default 100)")
		RateLimitBurst   = flag.Int("rate-limit-burst", DEFAULT_RATE_LIMIT_BURST, "rate limit burst size (default 10)")

		// Logging flags
		LogRequestBody = flag.Bool("log-request-body", DEFAULT_LOG_REQUEST_BODY, "log request body in debug mode (default false, SECURITY: may expose sensitive data)")

		ShowVersion     = flag.Bool("version", false, "display webhook version and quit")
		ResponseHeaders hook.ResponseHeaders
	)

	// 加锁读取 HooksFiles
	rules.RLockHooksFiles()
	var hooksFiles hook.HooksFiles
	hooksFiles = make(hook.HooksFiles, len(rules.HooksFiles))
	copy(hooksFiles, rules.HooksFiles)
	rules.RUnlockHooksFiles()
	flag.Var(&hooksFiles, "hooks", "path to the json file containing defined hooks the webhook should serve, use multiple times to load from different files")
	flag.Var(&ResponseHeaders, "header", "response header to return, specified in format name=value, use multiple times to set multiple headers")

	flag.Parse()

	if *Host != DEFAULT_HOST {
		flags.Host = *Host
	}

	if *Port != DEFAULT_PORT {
		flags.Port = *Port
	}

	if *Verbose != DEFAULT_ENABLE_VERBOSE {
		flags.Verbose = *Verbose
	}

	if *LogPath != DEFAULT_LOG_PATH {
		flags.LogPath = *LogPath
	}

	if *Debug != DEFAULT_ENABLE_DEBUG {
		flags.Debug = *Debug
	}

	if *NoPanic != DEFAULT_ENABLE_NO_PANIC {
		flags.NoPanic = *NoPanic
	}

	if *HotReload != DEFAULT_ENABLE_HOT_RELOAD {
		flags.HotReload = *HotReload
	}

	if *HooksURLPrefix != DEFAULT_URL_PREFIX {
		flags.HooksURLPrefix = *HooksURLPrefix
	}

	if *AsTemplate != DEFAULT_ENABLE_PARSE_TEMPLATE {
		flags.AsTemplate = *AsTemplate
	}

	if *UseXRequestID != DEFAULT_ENABLE_X_REQUEST_ID {
		flags.UseXRequestID = *UseXRequestID
	}

	if *XRequestIDLimit != DEFAULT_X_REQUEST_ID_LIMIT {
		flags.XRequestIDLimit = *XRequestIDLimit
	}

	if *MaxMultipartMem != DEFAULT_MAX_MPART_MEM {
		flags.MaxMultipartMem = *MaxMultipartMem
	}

	if *MaxRequestBodySize != DEFAULT_MAX_REQUEST_BODY_SIZE {
		flags.MaxRequestBodySize = *MaxRequestBodySize
	}

	if *SetGID != DEFAULT_GID {
		flags.SetGID = *SetGID
	}

	if *SetUID != DEFAULT_UID {
		flags.SetUID = *SetUID
	}

	if *HttpMethods != DEFAULT_HTTP_METHODS {
		flags.HttpMethods = *HttpMethods
	}

	if *PidPath != DEFAULT_PID_FILE {
		flags.PidPath = *PidPath
	}

	if *ShowVersion {
		flags.ShowVersion = true
	}

	if len(hooksFiles) > 0 {
		flags.HooksFiles = append(flags.HooksFiles, hooksFiles...)
	}
	// 加写锁更新 HooksFiles
	rules.LockHooksFiles()
	rules.HooksFiles = flags.HooksFiles
	rules.UnlockHooksFiles()

	if len(ResponseHeaders) > 0 {
		flags.ResponseHeaders = ResponseHeaders
	}

	if *Lang != DEFAULT_LANG {
		flags.Lang = *Lang
	}

	if *I18nDir != DEFAULT_I18N_DIR {
		flags.I18nDir = *I18nDir
	}

	if *HookTimeoutSeconds != DEFAULT_HOOK_TIMEOUT_SECONDS {
		flags.HookTimeoutSeconds = *HookTimeoutSeconds
	}

	if *MaxConcurrentHooks != DEFAULT_MAX_CONCURRENT_HOOKS {
		flags.MaxConcurrentHooks = *MaxConcurrentHooks
	}

	if *HookExecutionTimeout != DEFAULT_HOOK_EXECUTION_TIMEOUT {
		flags.HookExecutionTimeout = *HookExecutionTimeout
	}

	if *AllowAutoChmod != DEFAULT_ALLOW_AUTO_CHMOD {
		flags.AllowAutoChmod = *AllowAutoChmod
	}

	// Security settings
	if *AllowedCommandPaths != DEFAULT_ALLOWED_COMMAND_PATHS {
		flags.AllowedCommandPaths = *AllowedCommandPaths
	}
	if *MaxArgLength != DEFAULT_MAX_ARG_LENGTH {
		flags.MaxArgLength = *MaxArgLength
	}
	if *MaxTotalArgsLength != DEFAULT_MAX_TOTAL_ARGS_LENGTH {
		flags.MaxTotalArgsLength = *MaxTotalArgsLength
	}
	if *MaxArgsCount != DEFAULT_MAX_ARGS_COUNT {
		flags.MaxArgsCount = *MaxArgsCount
	}
	if *StrictMode != DEFAULT_STRICT_MODE {
		flags.StrictMode = *StrictMode
	}

	// Rate limiting settings
	if *RateLimitEnabled != DEFAULT_RATE_LIMIT_ENABLED {
		flags.RateLimitEnabled = *RateLimitEnabled
	}
	if *RateLimitRPS != DEFAULT_RATE_LIMIT_RPS {
		flags.RateLimitRPS = *RateLimitRPS
	}
	if *RateLimitBurst != DEFAULT_RATE_LIMIT_BURST {
		flags.RateLimitBurst = *RateLimitBurst
	}

	// Logging settings
	if *LogRequestBody != DEFAULT_LOG_REQUEST_BODY {
		flags.LogRequestBody = *LogRequestBody
	}

	return flags
}
