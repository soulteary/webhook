package flags

import "github.com/soulteary/webhook/internal/hook"

const (
	DEFAULT_HOST = "0.0.0.0"
	DEFAULT_PORT = 9000

	DEFAULT_LOG_PATH     = ""
	DEFAULT_URL_PREFIX   = "hooks"
	DEFAULT_HTTP_METHODS = ""
	DEFAULT_PID_FILE     = ""

	DEFAULT_ENABLE_VERBOSE        = false
	DEFAULT_ENABLE_DEBUG          = false
	DEFAULT_ENABLE_NO_PANIC       = false
	DEFAULT_ENABLE_HOT_RELOAD     = false
	DEFAULT_ENABLE_PARSE_TEMPLATE = false
	DEFAULT_ENABLE_X_REQUEST_ID   = false

	DEFAULT_X_REQUEST_ID_LIMIT    = 0
	DEFAULT_MAX_MPART_MEM         = 1 << 20
	DEFAULT_MAX_REQUEST_BODY_SIZE = 10 * 1024 * 1024 // 10MB
	DEFAULT_GID                   = 0
	DEFAULT_UID                   = 0

	DEFAULT_LANG     = "en-US"
	DEFAULT_I18N_DIR = "./locales"

	DEFAULT_HOOK_TIMEOUT_SECONDS   = 30
	DEFAULT_MAX_CONCURRENT_HOOKS   = 10
	DEFAULT_HOOK_EXECUTION_TIMEOUT = 5

	DEFAULT_ALLOW_AUTO_CHMOD = false

	// Security defaults
	DEFAULT_ALLOWED_COMMAND_PATHS = ""
	DEFAULT_MAX_ARG_LENGTH        = 1024 * 1024      // 1MB
	DEFAULT_MAX_TOTAL_ARGS_LENGTH = 10 * 1024 * 1024 // 10MB
	DEFAULT_MAX_ARGS_COUNT        = 1000
	DEFAULT_STRICT_MODE           = false

	// Rate limiting defaults
	DEFAULT_RATE_LIMIT_ENABLED = false
	DEFAULT_RATE_LIMIT_RPS     = 100 // requests per second
	DEFAULT_RATE_LIMIT_BURST   = 10  // burst size

	// Logging defaults
	DEFAULT_LOG_REQUEST_BODY = false // 默认不记录请求体，避免敏感信息泄露
)

const (
	ENV_KEY_HOST = "HOST"
	ENV_KEY_PORT = "PORT"

	ENV_KEY_VERBOSE    = "VERBOSE"
	ENV_KEY_DEBUG      = "DEBUG"
	ENV_KEY_NO_PANIC   = "NO_PANIC"
	ENV_KEY_LOG_PATH   = "LOG_PATH"
	ENV_KEY_HOT_RELOAD = "HOT_RELOAD"

	ENV_KEY_HOOKS_URLPREFIX       = "URL_PREFIX"
	ENV_KEY_HOOKS                 = "HOOKS"
	ENV_KEY_TEMPLATE              = "TEMPLATE"
	ENV_KEY_HTTP_METHODS          = "HTTP_METHODS"
	ENV_KEY_PID_FILE              = "PID_FILE"
	ENV_KEY_X_REQUEST_ID          = "X_REQUEST_ID"
	ENV_KEY_MAX_MPART_MEM         = "MAX_MPART_MEM"
	ENV_KEY_MAX_REQUEST_BODY_SIZE = "MAX_REQUEST_BODY_SIZE"
	ENV_KEY_GID                   = "GID"
	ENV_KEY_UID                   = "UID"
	ENV_KEY_HEADER                = "HEADER"

	ENV_KEY_LANG = "LANGUAGE"
	ENV_KEY_I18N = "LANG_DIR"

	ENV_KEY_HOOK_TIMEOUT_SECONDS   = "HOOK_TIMEOUT_SECONDS"
	ENV_KEY_MAX_CONCURRENT_HOOKS   = "MAX_CONCURRENT_HOOKS"
	ENV_KEY_HOOK_EXECUTION_TIMEOUT = "HOOK_EXECUTION_TIMEOUT"
	ENV_KEY_ALLOW_AUTO_CHMOD       = "ALLOW_AUTO_CHMOD"

	// Security environment keys
	ENV_KEY_ALLOWED_COMMAND_PATHS = "ALLOWED_COMMAND_PATHS"
	ENV_KEY_MAX_ARG_LENGTH        = "MAX_ARG_LENGTH"
	ENV_KEY_MAX_TOTAL_ARGS_LENGTH = "MAX_TOTAL_ARGS_LENGTH"
	ENV_KEY_MAX_ARGS_COUNT        = "MAX_ARGS_COUNT"
	ENV_KEY_STRICT_MODE           = "STRICT_MODE"

	// Rate limiting environment keys
	ENV_KEY_RATE_LIMIT_ENABLED = "RATE_LIMIT_ENABLED"
	ENV_KEY_RATE_LIMIT_RPS     = "RATE_LIMIT_RPS"
	ENV_KEY_RATE_LIMIT_BURST   = "RATE_LIMIT_BURST"

	// Logging environment keys
	ENV_KEY_LOG_REQUEST_BODY = "LOG_REQUEST_BODY"
)

type AppFlags struct {
	Host               string
	Port               int
	Verbose            bool
	LogPath            string
	Debug              bool
	NoPanic            bool
	HotReload          bool
	HooksURLPrefix     string
	AsTemplate         bool
	UseXRequestID      bool
	XRequestIDLimit    int
	MaxMultipartMem    int64
	MaxRequestBodySize int64
	SetGID             int
	SetUID             int
	HttpMethods        string
	PidPath            string

	ShowVersion     bool
	HooksFiles      hook.HooksFiles
	ResponseHeaders hook.ResponseHeaders

	Lang    string
	I18nDir string

	HookTimeoutSeconds   int
	MaxConcurrentHooks   int
	HookExecutionTimeout int
	AllowAutoChmod       bool

	// Security settings
	AllowedCommandPaths string // 逗号分隔的允许的命令路径列表
	MaxArgLength        int    // 单个参数最大长度
	MaxTotalArgsLength  int    // 所有参数总长度限制
	MaxArgsCount        int    // 最大参数数量
	StrictMode          bool   // 严格模式：禁止危险字符

	// Rate limiting settings
	RateLimitEnabled bool // 是否启用限流
	RateLimitRPS     int  // 每秒请求数限制
	RateLimitBurst   int  // 突发请求数限制

	// Logging settings
	LogRequestBody bool // 是否在调试模式下记录请求体（默认false，避免敏感信息泄露）
}
