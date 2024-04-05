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

	DEFAULT_X_REQUEST_ID_LIMIT = 0
	DEFAULT_MAX_MPART_MEM      = 1 << 20
	DEFAULT_GID                = 0
	DEFAULT_UID                = 0

	DEFAULT_LANG     = "en-US"
	DEFAULT_I18N_DIR = "./locales"
)

const (
	ENV_KEY_HOST = "HOST"
	ENV_KEY_PORT = "PORT"

	ENV_KEY_VERBOSE    = "VERBOSE"
	ENV_KEY_DEBUG      = "DEBUG"
	ENV_KEY_NO_PANIC   = "NO_PANIC"
	ENV_KEY_LOG_PATH   = "LOG_PATH"
	ENV_KEY_HOT_RELOAD = "HOT_RELOAD"

	ENV_KEY_HOOKS_URLPREFIX = "URL_PREFIX"
	ENV_KEY_HOOKS           = "HOOKS"
	ENV_KEY_TEMPLATE        = "TEMPLATE"
	ENV_KEY_HTTP_METHODS    = "HTTP_METHODS"
	ENV_KEY_PID_FILE        = "PID_FILE"
	ENV_KEY_X_REQUEST_ID    = "X_REQUEST_ID"
	ENV_KEY_MAX_MPART_MEM   = "MAX_MPART_MEM"
	ENV_KEY_GID             = "GID"
	ENV_KEY_UID             = "UID"
	ENV_KEY_HEADER          = "HEADER"

	ENV_KEY_LANG = "LANGUAGE"
	ENV_KEY_I18N = "LANG_DIR"
)

type AppFlags struct {
	Host            string
	Port            int
	Verbose         bool
	LogPath         string
	Debug           bool
	NoPanic         bool
	HotReload       bool
	HooksURLPrefix  string
	AsTemplate      bool
	UseXRequestID   bool
	XRequestIDLimit int
	MaxMultipartMem int64
	SetGID          int
	SetUID          int
	HttpMethods     string
	PidPath         string

	ShowVersion     bool
	HooksFiles      hook.HooksFiles
	ResponseHeaders hook.ResponseHeaders

	Lang    string
	I18nDir string
}
