package flags

import (
	"fmt"
	"strings"

	"github.com/soulteary/cli-kit/env"
	"github.com/soulteary/webhook/internal/hook"
)

func ParseEnvs() AppFlags {
	var flags AppFlags
	flags.Host = env.GetTrimmed(ENV_KEY_HOST, DEFAULT_HOST)
	flags.Port = env.GetInt(ENV_KEY_PORT, DEFAULT_PORT)
	flags.Verbose = env.GetBool(ENV_KEY_VERBOSE, DEFAULT_ENABLE_VERBOSE)
	flags.LogPath = env.GetTrimmed(ENV_KEY_LOG_PATH, DEFAULT_LOG_PATH)
	flags.Debug = env.GetBool(ENV_KEY_DEBUG, DEFAULT_ENABLE_DEBUG)
	flags.NoPanic = env.GetBool(ENV_KEY_NO_PANIC, DEFAULT_ENABLE_NO_PANIC)
	flags.HotReload = env.GetBool(ENV_KEY_HOT_RELOAD, DEFAULT_ENABLE_HOT_RELOAD)
	flags.HooksURLPrefix = env.GetTrimmed(ENV_KEY_HOOKS_URLPREFIX, DEFAULT_URL_PREFIX)
	flags.AsTemplate = env.GetBool(ENV_KEY_TEMPLATE, DEFAULT_ENABLE_PARSE_TEMPLATE)
	flags.UseXRequestID = env.GetBool(ENV_KEY_X_REQUEST_ID, DEFAULT_ENABLE_X_REQUEST_ID)
	flags.XRequestIDLimit = env.GetInt(ENV_KEY_X_REQUEST_ID, DEFAULT_X_REQUEST_ID_LIMIT)
	flags.MaxMultipartMem = env.GetInt64(ENV_KEY_MAX_MPART_MEM, int64(DEFAULT_MAX_MPART_MEM))
	flags.MaxRequestBodySize = env.GetInt64(ENV_KEY_MAX_REQUEST_BODY_SIZE, int64(DEFAULT_MAX_REQUEST_BODY_SIZE))
	flags.SetGID = env.GetInt(ENV_KEY_GID, DEFAULT_GID)
	flags.SetUID = env.GetInt(ENV_KEY_UID, DEFAULT_UID)
	flags.HttpMethods = env.GetTrimmed(ENV_KEY_HTTP_METHODS, DEFAULT_HTTP_METHODS)
	flags.PidPath = env.GetTrimmed(ENV_KEY_PID_FILE, DEFAULT_PID_FILE)

	// init i18n, set lang and i18n dir
	flags.Lang = env.GetTrimmed(ENV_KEY_LANG, DEFAULT_LANG)
	flags.I18nDir = env.GetTrimmed(ENV_KEY_I18N, DEFAULT_I18N_DIR)

	// hook execution configuration
	flags.HookTimeoutSeconds = env.GetInt(ENV_KEY_HOOK_TIMEOUT_SECONDS, DEFAULT_HOOK_TIMEOUT_SECONDS)
	flags.MaxConcurrentHooks = env.GetInt(ENV_KEY_MAX_CONCURRENT_HOOKS, DEFAULT_MAX_CONCURRENT_HOOKS)
	flags.HookExecutionTimeout = env.GetInt(ENV_KEY_HOOK_EXECUTION_TIMEOUT, DEFAULT_HOOK_EXECUTION_TIMEOUT)
	flags.AllowAutoChmod = env.GetBool(ENV_KEY_ALLOW_AUTO_CHMOD, DEFAULT_ALLOW_AUTO_CHMOD)

	// Security settings
	flags.AllowedCommandPaths = env.GetTrimmed(ENV_KEY_ALLOWED_COMMAND_PATHS, DEFAULT_ALLOWED_COMMAND_PATHS)
	flags.MaxArgLength = env.GetInt(ENV_KEY_MAX_ARG_LENGTH, DEFAULT_MAX_ARG_LENGTH)
	flags.MaxTotalArgsLength = env.GetInt(ENV_KEY_MAX_TOTAL_ARGS_LENGTH, DEFAULT_MAX_TOTAL_ARGS_LENGTH)
	flags.MaxArgsCount = env.GetInt(ENV_KEY_MAX_ARGS_COUNT, DEFAULT_MAX_ARGS_COUNT)
	flags.StrictMode = env.GetBool(ENV_KEY_STRICT_MODE, DEFAULT_STRICT_MODE)

	// Rate limiting settings
	flags.RateLimitEnabled = env.GetBool(ENV_KEY_RATE_LIMIT_ENABLED, DEFAULT_RATE_LIMIT_ENABLED)
	flags.RateLimitRPS = env.GetInt(ENV_KEY_RATE_LIMIT_RPS, DEFAULT_RATE_LIMIT_RPS)
	flags.RateLimitBurst = env.GetInt(ENV_KEY_RATE_LIMIT_BURST, DEFAULT_RATE_LIMIT_BURST)

	// Logging settings
	flags.LogRequestBody = env.GetBool(ENV_KEY_LOG_REQUEST_BODY, DEFAULT_LOG_REQUEST_BODY)

	// HTTP server timeout settings
	flags.ReadHeaderTimeoutSeconds = env.GetInt(ENV_KEY_READ_HEADER_TIMEOUT_SECONDS, DEFAULT_READ_HEADER_TIMEOUT_SECONDS)
	flags.ReadTimeoutSeconds = env.GetInt(ENV_KEY_READ_TIMEOUT_SECONDS, DEFAULT_READ_TIMEOUT_SECONDS)
	flags.WriteTimeoutSeconds = env.GetInt(ENV_KEY_WRITE_TIMEOUT_SECONDS, DEFAULT_WRITE_TIMEOUT_SECONDS)
	flags.IdleTimeoutSeconds = env.GetInt(ENV_KEY_IDLE_TIMEOUT_SECONDS, DEFAULT_IDLE_TIMEOUT_SECONDS)
	flags.MaxHeaderBytes = env.GetInt(ENV_KEY_MAX_HEADER_BYTES, DEFAULT_MAX_HEADER_BYTES)

	hooks := strings.Split(env.GetTrimmed(ENV_KEY_HOOKS, ""), ",")
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
