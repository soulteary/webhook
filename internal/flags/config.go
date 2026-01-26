package flags

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/soulteary/cli-kit/configutil"
	"github.com/soulteary/cli-kit/env"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
)

// ParseConfig parses configuration from CLI flags and environment variables.
// Priority: CLI flag > Environment variable > Default value
func ParseConfig() AppFlags {
	fs := flag.NewFlagSet("webhook", flag.ExitOnError)

	// Define all flags
	fs.String("ip", DEFAULT_HOST, "ip the webhook should serve hooks on")
	fs.Int("port", DEFAULT_PORT, "port the webhook should serve hooks on")
	fs.Bool("verbose", DEFAULT_ENABLE_VERBOSE, "show verbose output")
	fs.String("logfile", DEFAULT_LOG_PATH, "send log output to a file; implicitly enables verbose logging")
	fs.Bool("debug", DEFAULT_ENABLE_DEBUG, "show debug output")
	fs.Bool("nopanic", DEFAULT_ENABLE_NO_PANIC, "do not panic if hooks cannot be loaded when webhook is not running in verbose mode")
	fs.Bool("hotreload", DEFAULT_ENABLE_HOT_RELOAD, "watch hooks file for changes and reload them automatically")
	fs.String("urlprefix", DEFAULT_URL_PREFIX, "url prefix to use for served hooks (protocol://yourserver:port/PREFIX/:hook-id)")
	fs.Bool("template", DEFAULT_ENABLE_PARSE_TEMPLATE, "parse hooks file as a Go template")
	fs.Bool("x-request-id", DEFAULT_ENABLE_X_REQUEST_ID, "use X-Request-Id header, if present, as request ID")
	fs.Int("x-request-id-limit", DEFAULT_X_REQUEST_ID_LIMIT, "truncate X-Request-Id header to limit; default no limit")
	fs.Int64("max-multipart-mem", DEFAULT_MAX_MPART_MEM, "maximum memory in bytes for parsing multipart form data before disk caching")
	fs.Int64("max-request-body-size", DEFAULT_MAX_REQUEST_BODY_SIZE, "maximum size in bytes for request body (default 10MB)")
	fs.Int("setgid", DEFAULT_GID, "set group ID after opening listening port; must be used with setuid")
	fs.Int("setuid", DEFAULT_UID, "set user ID after opening listening port; must be used with setgid")
	fs.String("http-methods", DEFAULT_HTTP_METHODS, `set default allowed HTTP methods (ie. "POST"); separate methods with comma`)
	fs.String("pidfile", DEFAULT_PID_FILE, "create PID file at the given path")

	fs.String("lang", DEFAULT_LANG, "set the language code for the webhook")
	fs.String("lang-dir", DEFAULT_I18N_DIR, "set the directory for the i18n files")

	fs.Int("hook-timeout-seconds", DEFAULT_HOOK_TIMEOUT_SECONDS, "default timeout in seconds for hook execution (default 30)")
	fs.Int("max-concurrent-hooks", DEFAULT_MAX_CONCURRENT_HOOKS, "maximum number of concurrent hook executions (default 10)")
	fs.Int("hook-execution-timeout", DEFAULT_HOOK_EXECUTION_TIMEOUT, "timeout in seconds for acquiring execution slot when max concurrent hooks reached (default 5)")
	fs.Bool("allow-auto-chmod", DEFAULT_ALLOW_AUTO_CHMOD, "allow automatically modifying file permissions when permission denied (SECURITY RISK: default false)")

	// Security flags
	fs.String("allowed-command-paths", DEFAULT_ALLOWED_COMMAND_PATHS, "comma-separated list of allowed command paths (directories or files) for command execution whitelist; empty means no whitelist check")
	fs.Int("max-arg-length", DEFAULT_MAX_ARG_LENGTH, "maximum length for a single command argument in bytes (default 1MB)")
	fs.Int("max-total-args-length", DEFAULT_MAX_TOTAL_ARGS_LENGTH, "maximum total length for all command arguments in bytes (default 10MB)")
	fs.Int("max-args-count", DEFAULT_MAX_ARGS_COUNT, "maximum number of command arguments (default 1000)")
	fs.Bool("strict-mode", DEFAULT_STRICT_MODE, "strict mode: reject arguments containing potentially dangerous characters (default false)")

	// Rate limiting flags
	fs.Bool("rate-limit-enabled", DEFAULT_RATE_LIMIT_ENABLED, "enable rate limiting (default false)")
	fs.Int("rate-limit-rps", DEFAULT_RATE_LIMIT_RPS, "rate limit requests per second (default 100)")
	fs.Int("rate-limit-burst", DEFAULT_RATE_LIMIT_BURST, "rate limit burst size (default 10)")

	// Logging flags
	fs.Bool("log-request-body", DEFAULT_LOG_REQUEST_BODY, "log request body in debug mode (default false, SECURITY: may expose sensitive data)")

	// HTTP server timeout flags
	fs.Int("read-header-timeout-seconds", DEFAULT_READ_HEADER_TIMEOUT_SECONDS, "timeout in seconds for reading request headers (default 5)")
	fs.Int("read-timeout-seconds", DEFAULT_READ_TIMEOUT_SECONDS, "timeout in seconds for reading request body (default 10)")
	fs.Int("write-timeout-seconds", DEFAULT_WRITE_TIMEOUT_SECONDS, "timeout in seconds for writing response (default 30)")
	fs.Int("idle-timeout-seconds", DEFAULT_IDLE_TIMEOUT_SECONDS, "timeout in seconds for idle connections (default 90)")
	fs.Int("max-header-bytes", DEFAULT_MAX_HEADER_BYTES, "maximum size in bytes for request headers (default 1MB)")

	// Tracing flags
	fs.Bool("tracing-enabled", DEFAULT_TRACING_ENABLED, "enable distributed tracing with OpenTelemetry (default false)")
	fs.String("otlp-endpoint", DEFAULT_OTLP_ENDPOINT, "OTLP exporter endpoint (e.g., localhost:4318)")
	fs.String("tracing-service-name", DEFAULT_TRACING_SVC_NAME, "service name for tracing (default 'webhook')")

	showVersion := fs.Bool("version", false, "display webhook version and quit")
	validateConfig := fs.Bool("validate-config", false, "validate configuration and exit")

	// Multi-value flags
	rules.RLockHooksFiles()
	var hooksFiles hook.HooksFiles
	hooksFiles = make(hook.HooksFiles, len(rules.HooksFiles))
	copy(hooksFiles, rules.HooksFiles)
	rules.RUnlockHooksFiles()
	fs.Var(&hooksFiles, "hooks", "path to the json file containing defined hooks the webhook should serve, use multiple times to load from different files")

	var responseHeaders hook.ResponseHeaders
	fs.Var(&responseHeaders, "header", "response header to return, specified in format name=value, use multiple times to set multiple headers")

	// Parse command line arguments
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Build config using configutil with priority: CLI > ENV > Default
	var flags AppFlags

	// Basic settings
	flags.Host = configutil.ResolveString(fs, "ip", ENV_KEY_HOST, DEFAULT_HOST, true)
	flags.Port = configutil.ResolveInt(fs, "port", ENV_KEY_PORT, DEFAULT_PORT, false)
	flags.Verbose = configutil.ResolveBool(fs, "verbose", ENV_KEY_VERBOSE, DEFAULT_ENABLE_VERBOSE)
	flags.LogPath = configutil.ResolveString(fs, "logfile", ENV_KEY_LOG_PATH, DEFAULT_LOG_PATH, true)
	flags.Debug = configutil.ResolveBool(fs, "debug", ENV_KEY_DEBUG, DEFAULT_ENABLE_DEBUG)
	flags.NoPanic = configutil.ResolveBool(fs, "nopanic", ENV_KEY_NO_PANIC, DEFAULT_ENABLE_NO_PANIC)
	flags.HotReload = configutil.ResolveBool(fs, "hotreload", ENV_KEY_HOT_RELOAD, DEFAULT_ENABLE_HOT_RELOAD)
	flags.HooksURLPrefix = configutil.ResolveString(fs, "urlprefix", ENV_KEY_HOOKS_URLPREFIX, DEFAULT_URL_PREFIX, true)
	flags.AsTemplate = configutil.ResolveBool(fs, "template", ENV_KEY_TEMPLATE, DEFAULT_ENABLE_PARSE_TEMPLATE)
	flags.UseXRequestID = configutil.ResolveBool(fs, "x-request-id", ENV_KEY_X_REQUEST_ID, DEFAULT_ENABLE_X_REQUEST_ID)
	flags.XRequestIDLimit = configutil.ResolveInt(fs, "x-request-id-limit", ENV_KEY_X_REQUEST_ID, DEFAULT_X_REQUEST_ID_LIMIT, true)
	flags.MaxMultipartMem = configutil.ResolveInt64(fs, "max-multipart-mem", ENV_KEY_MAX_MPART_MEM, int64(DEFAULT_MAX_MPART_MEM), true)
	flags.MaxRequestBodySize = configutil.ResolveInt64(fs, "max-request-body-size", ENV_KEY_MAX_REQUEST_BODY_SIZE, int64(DEFAULT_MAX_REQUEST_BODY_SIZE), true)
	flags.SetGID = configutil.ResolveInt(fs, "setgid", ENV_KEY_GID, DEFAULT_GID, true)
	flags.SetUID = configutil.ResolveInt(fs, "setuid", ENV_KEY_UID, DEFAULT_UID, true)
	flags.HttpMethods = configutil.ResolveString(fs, "http-methods", ENV_KEY_HTTP_METHODS, DEFAULT_HTTP_METHODS, true)
	flags.PidPath = configutil.ResolveString(fs, "pidfile", ENV_KEY_PID_FILE, DEFAULT_PID_FILE, true)

	// i18n settings
	flags.Lang = configutil.ResolveString(fs, "lang", ENV_KEY_LANG, DEFAULT_LANG, true)
	flags.I18nDir = configutil.ResolveString(fs, "lang-dir", ENV_KEY_I18N, DEFAULT_I18N_DIR, true)

	// Hook execution configuration
	flags.HookTimeoutSeconds = configutil.ResolveInt(fs, "hook-timeout-seconds", ENV_KEY_HOOK_TIMEOUT_SECONDS, DEFAULT_HOOK_TIMEOUT_SECONDS, true)
	flags.MaxConcurrentHooks = configutil.ResolveInt(fs, "max-concurrent-hooks", ENV_KEY_MAX_CONCURRENT_HOOKS, DEFAULT_MAX_CONCURRENT_HOOKS, false)
	flags.HookExecutionTimeout = configutil.ResolveInt(fs, "hook-execution-timeout", ENV_KEY_HOOK_EXECUTION_TIMEOUT, DEFAULT_HOOK_EXECUTION_TIMEOUT, true)
	flags.AllowAutoChmod = configutil.ResolveBool(fs, "allow-auto-chmod", ENV_KEY_ALLOW_AUTO_CHMOD, DEFAULT_ALLOW_AUTO_CHMOD)

	// Security settings
	flags.AllowedCommandPaths = configutil.ResolveString(fs, "allowed-command-paths", ENV_KEY_ALLOWED_COMMAND_PATHS, DEFAULT_ALLOWED_COMMAND_PATHS, true)
	flags.MaxArgLength = configutil.ResolveInt(fs, "max-arg-length", ENV_KEY_MAX_ARG_LENGTH, DEFAULT_MAX_ARG_LENGTH, false)
	flags.MaxTotalArgsLength = configutil.ResolveInt(fs, "max-total-args-length", ENV_KEY_MAX_TOTAL_ARGS_LENGTH, DEFAULT_MAX_TOTAL_ARGS_LENGTH, false)
	flags.MaxArgsCount = configutil.ResolveInt(fs, "max-args-count", ENV_KEY_MAX_ARGS_COUNT, DEFAULT_MAX_ARGS_COUNT, false)
	flags.StrictMode = configutil.ResolveBool(fs, "strict-mode", ENV_KEY_STRICT_MODE, DEFAULT_STRICT_MODE)

	// Rate limiting settings
	flags.RateLimitEnabled = configutil.ResolveBool(fs, "rate-limit-enabled", ENV_KEY_RATE_LIMIT_ENABLED, DEFAULT_RATE_LIMIT_ENABLED)
	flags.RateLimitRPS = configutil.ResolveInt(fs, "rate-limit-rps", ENV_KEY_RATE_LIMIT_RPS, DEFAULT_RATE_LIMIT_RPS, false)
	flags.RateLimitBurst = configutil.ResolveInt(fs, "rate-limit-burst", ENV_KEY_RATE_LIMIT_BURST, DEFAULT_RATE_LIMIT_BURST, false)

	// Logging settings
	flags.LogRequestBody = configutil.ResolveBool(fs, "log-request-body", ENV_KEY_LOG_REQUEST_BODY, DEFAULT_LOG_REQUEST_BODY)

	// HTTP server timeout settings
	flags.ReadHeaderTimeoutSeconds = configutil.ResolveInt(fs, "read-header-timeout-seconds", ENV_KEY_READ_HEADER_TIMEOUT_SECONDS, DEFAULT_READ_HEADER_TIMEOUT_SECONDS, true)
	flags.ReadTimeoutSeconds = configutil.ResolveInt(fs, "read-timeout-seconds", ENV_KEY_READ_TIMEOUT_SECONDS, DEFAULT_READ_TIMEOUT_SECONDS, true)
	flags.WriteTimeoutSeconds = configutil.ResolveInt(fs, "write-timeout-seconds", ENV_KEY_WRITE_TIMEOUT_SECONDS, DEFAULT_WRITE_TIMEOUT_SECONDS, true)
	flags.IdleTimeoutSeconds = configutil.ResolveInt(fs, "idle-timeout-seconds", ENV_KEY_IDLE_TIMEOUT_SECONDS, DEFAULT_IDLE_TIMEOUT_SECONDS, true)
	flags.MaxHeaderBytes = configutil.ResolveInt(fs, "max-header-bytes", ENV_KEY_MAX_HEADER_BYTES, DEFAULT_MAX_HEADER_BYTES, false)

	// Tracing settings
	flags.TracingEnabled = configutil.ResolveBool(fs, "tracing-enabled", ENV_KEY_TRACING_ENABLED, DEFAULT_TRACING_ENABLED)
	flags.OTLPEndpoint = configutil.ResolveString(fs, "otlp-endpoint", ENV_KEY_OTLP_ENDPOINT, DEFAULT_OTLP_ENDPOINT, true)
	flags.TracingServiceName = configutil.ResolveString(fs, "tracing-service-name", ENV_KEY_TRACING_SVC_NAME, DEFAULT_TRACING_SVC_NAME, true)

	// Special flags
	flags.ShowVersion = *showVersion
	flags.ValidateConfig = *validateConfig

	// Handle multi-value flags with ENV fallback
	if len(hooksFiles) > 0 {
		flags.HooksFiles = hooksFiles
	} else {
		// Try environment variable
		hooksEnv := env.GetTrimmed(ENV_KEY_HOOKS, "")
		if hooksEnv != "" {
			hooks := strings.Split(hooksEnv, ",")
			for _, hookPath := range hooks {
				hookPath = strings.TrimSpace(hookPath)
				if hookPath != "" {
					if err := flags.HooksFiles.Set(hookPath); err != nil {
						fmt.Fprintf(os.Stderr, "Error parsing hooks from environment variable: %v\n", err)
					}
				}
			}
		}
	}

	// Update global HooksFiles
	rules.LockHooksFiles()
	rules.HooksFiles = flags.HooksFiles
	rules.UnlockHooksFiles()

	if len(responseHeaders) > 0 {
		flags.ResponseHeaders = responseHeaders
	}

	return flags
}
