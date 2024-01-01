package flags

import (
	"flag"

	"github.com/adnanh/webhook/internal/hook"
	"github.com/adnanh/webhook/internal/rules"
)

var (
	Host            = flag.String("ip", DEFAULT_HOST, "ip the webhook should serve hooks on")
	Port            = flag.Int("port", DEFAULT_PORT, "port the webhook should serve hooks on")
	Verbose         = flag.Bool("verbose", DEFAULT_ENABLE_VERBOSE, "show verbose output")
	LogPath         = flag.String("logfile", DEFAULT_LOG_PATH, "send log output to a file; implicitly enables verbose logging")
	Debug           = flag.Bool("debug", DEFAULT_ENABLE_DEBUG, "show debug output")
	NoPanic         = flag.Bool("nopanic", DEFAULT_ENABLE_NO_PANIC, "do not panic if hooks cannot be loaded when webhook is not running in verbose mode")
	HotReload       = flag.Bool("hotreload", DEFAULT_ENABLE_HOT_RELOAD, "watch hooks file for changes and reload them automatically")
	HooksURLPrefix  = flag.String("urlprefix", DEFAULT_URL_PREFIX, "url prefix to use for served hooks (protocol://yourserver:port/PREFIX/:hook-id)")
	AsTemplate      = flag.Bool("template", DEFAULT_ENABLE_PARSE_TEMPLATE, "parse hooks file as a Go template")
	UseXRequestID   = flag.Bool("x-request-id", DEFAULT_ENABLE_X_REQUEST_ID, "use X-Request-Id header, if present, as request ID")
	XRequestIDLimit = flag.Int("x-request-id-limit", DEFAULT_X_REQUEST_ID_LIMIT, "truncate X-Request-Id header to limit; default no limit")
	MaxMultipartMem = flag.Int64("max-multipart-mem", DEFAULT_MAX_MPART_MEM, "maximum memory in bytes for parsing multipart form data before disk caching")
	SetGID          = flag.Int("setgid", DEFAULT_GID, "set group ID after opening listening port; must be used with setuid")
	SetUID          = flag.Int("setuid", DEFAULT_UID, "set user ID after opening listening port; must be used with setgid")
	HttpMethods     = flag.String("http-methods", DEFAULT_HTTP_METHODS, `set default allowed HTTP methods (ie. "POST"); separate methods with comma`)
	PidPath         = flag.String("pidfile", DEFAULT_PID_FILE, "create PID file at the given path")

	JustDisplayVersion = flag.Bool("version", false, "display webhook version and quit")

	ResponseHeaders hook.ResponseHeaders
)

func Parse() {
	flag.Var(&rules.HooksFiles, "hooks", "path to the json file containing defined hooks the webhook should serve, use multiple times to load from different files")
	flag.Var(&ResponseHeaders, "header", "response header to return, specified in format name=value, use multiple times to set multiple headers")

	flag.Parse()
}
