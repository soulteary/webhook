package flags

import (
	"fmt"
	"strings"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/soulteary/webhook/internal/hook"
)

func ParseEnvs() AppFlags {
	var flags AppFlags
	flags.Host = fn.GetEnvStr(ENV_KEY_HOST, DEFAULT_HOST)
	flags.Port = fn.GetEnvInt(ENV_KEY_PORT, DEFAULT_PORT)
	flags.Verbose = fn.GetEnvBool(ENV_KEY_VERBOSE, DEFAULT_ENABLE_VERBOSE)
	flags.LogPath = fn.GetEnvStr(ENV_KEY_LOG_PATH, DEFAULT_LOG_PATH)
	flags.Debug = fn.GetEnvBool(ENV_KEY_DEBUG, DEFAULT_ENABLE_DEBUG)
	flags.NoPanic = fn.GetEnvBool(ENV_KEY_NO_PANIC, DEFAULT_ENABLE_NO_PANIC)
	flags.HotReload = fn.GetEnvBool(ENV_KEY_HOT_RELOAD, DEFAULT_ENABLE_HOT_RELOAD)
	flags.HooksURLPrefix = fn.GetEnvStr(ENV_KEY_HOOKS_URLPREFIX, DEFAULT_URL_PREFIX)
	flags.AsTemplate = fn.GetEnvBool(ENV_KEY_TEMPLATE, DEFAULT_ENABLE_PARSE_TEMPLATE)
	flags.UseXRequestID = fn.GetEnvBool(ENV_KEY_X_REQUEST_ID, DEFAULT_ENABLE_X_REQUEST_ID)
	flags.XRequestIDLimit = fn.GetEnvInt(ENV_KEY_X_REQUEST_ID, DEFAULT_X_REQUEST_ID_LIMIT)
	flags.MaxMultipartMem = int64(fn.GetEnvInt(ENV_KEY_MAX_MPART_MEM, DEFAULT_MAX_MPART_MEM))
	flags.MaxRequestBodySize = int64(fn.GetEnvInt(ENV_KEY_MAX_REQUEST_BODY_SIZE, DEFAULT_MAX_REQUEST_BODY_SIZE))
	flags.SetGID = fn.GetEnvInt(ENV_KEY_GID, DEFAULT_GID)
	flags.SetUID = fn.GetEnvInt(ENV_KEY_UID, DEFAULT_UID)
	flags.HttpMethods = fn.GetEnvStr(ENV_KEY_HTTP_METHODS, DEFAULT_HTTP_METHODS)
	flags.PidPath = fn.GetEnvStr(ENV_KEY_PID_FILE, DEFAULT_PID_FILE)

	// init i18n, set lang and i18n dir
	flags.Lang = fn.GetEnvStr(ENV_KEY_LANG, DEFAULT_LANG)
	flags.I18nDir = fn.GetEnvStr(ENV_KEY_I18N, DEFAULT_I18N_DIR)

	// hook execution configuration
	flags.HookTimeoutSeconds = fn.GetEnvInt(ENV_KEY_HOOK_TIMEOUT_SECONDS, DEFAULT_HOOK_TIMEOUT_SECONDS)
	flags.MaxConcurrentHooks = fn.GetEnvInt(ENV_KEY_MAX_CONCURRENT_HOOKS, DEFAULT_MAX_CONCURRENT_HOOKS)
	flags.HookExecutionTimeout = fn.GetEnvInt(ENV_KEY_HOOK_EXECUTION_TIMEOUT, DEFAULT_HOOK_EXECUTION_TIMEOUT)
	flags.AllowAutoChmod = fn.GetEnvBool(ENV_KEY_ALLOW_AUTO_CHMOD, DEFAULT_ALLOW_AUTO_CHMOD)

	// Security settings
	flags.AllowedCommandPaths = fn.GetEnvStr(ENV_KEY_ALLOWED_COMMAND_PATHS, DEFAULT_ALLOWED_COMMAND_PATHS)
	flags.MaxArgLength = fn.GetEnvInt(ENV_KEY_MAX_ARG_LENGTH, DEFAULT_MAX_ARG_LENGTH)
	flags.MaxTotalArgsLength = fn.GetEnvInt(ENV_KEY_MAX_TOTAL_ARGS_LENGTH, DEFAULT_MAX_TOTAL_ARGS_LENGTH)
	flags.MaxArgsCount = fn.GetEnvInt(ENV_KEY_MAX_ARGS_COUNT, DEFAULT_MAX_ARGS_COUNT)
	flags.StrictMode = fn.GetEnvBool(ENV_KEY_STRICT_MODE, DEFAULT_STRICT_MODE)

	// Rate limiting settings
	flags.RateLimitEnabled = fn.GetEnvBool(ENV_KEY_RATE_LIMIT_ENABLED, DEFAULT_RATE_LIMIT_ENABLED)
	flags.RateLimitRPS = fn.GetEnvInt(ENV_KEY_RATE_LIMIT_RPS, DEFAULT_RATE_LIMIT_RPS)
	flags.RateLimitBurst = fn.GetEnvInt(ENV_KEY_RATE_LIMIT_BURST, DEFAULT_RATE_LIMIT_BURST)

	// Logging settings
	flags.LogRequestBody = fn.GetEnvBool(ENV_KEY_LOG_REQUEST_BODY, DEFAULT_LOG_REQUEST_BODY)

	hooks := strings.Split(fn.GetEnvStr(ENV_KEY_HOOKS, ""), ",")
	var hooksFiles hook.HooksFiles
	for _, hook := range hooks {
		err := hooksFiles.Set(hook)
		if err != nil {
			fmt.Println("Error parsing hooks from environment variable: ", err)
		}
	}
	if len(hooksFiles) > 0 {
		flags.HooksFiles = hooksFiles
	}
	return flags
}
