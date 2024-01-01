package flags

import (
	"flag"

	"github.com/adnanh/webhook/internal/hook"
	"github.com/adnanh/webhook/internal/rules"
)

var (
	Host               = flag.String("ip", "0.0.0.0", "ip the webhook should serve hooks on")
	Port               = flag.Int("port", 9000, "port the webhook should serve hooks on")
	Verbose            = flag.Bool("verbose", false, "show verbose output")
	LogPath            = flag.String("logfile", "", "send log output to a file; implicitly enables verbose logging")
	Debug              = flag.Bool("debug", false, "show debug output")
	NoPanic            = flag.Bool("nopanic", false, "do not panic if hooks cannot be loaded when webhook is not running in verbose mode")
	HotReload          = flag.Bool("hotreload", false, "watch hooks file for changes and reload them automatically")
	HooksURLPrefix     = flag.String("urlprefix", "hooks", "url prefix to use for served hooks (protocol://yourserver:port/PREFIX/:hook-id)")
	AsTemplate         = flag.Bool("template", false, "parse hooks file as a Go template")
	JustDisplayVersion = flag.Bool("version", false, "display webhook version and quit")
	UseXRequestID      = flag.Bool("x-request-id", false, "use X-Request-Id header, if present, as request ID")
	XRequestIDLimit    = flag.Int("x-request-id-limit", 0, "truncate X-Request-Id header to limit; default no limit")
	MaxMultipartMem    = flag.Int64("max-multipart-mem", 1<<20, "maximum memory in bytes for parsing multipart form data before disk caching")
	SetGID             = flag.Int("setgid", 0, "set group ID after opening listening port; must be used with setuid")
	SetUID             = flag.Int("setuid", 0, "set user ID after opening listening port; must be used with setgid")
	HttpMethods        = flag.String("http-methods", "", `set default allowed HTTP methods (ie. "POST"); separate methods with comma`)
	PidPath            = flag.String("pidfile", "", "create PID file at the given path")

	ResponseHeaders hook.ResponseHeaders
)

func Parse() {
	flag.Var(&rules.HooksFiles, "hooks", "path to the json file containing defined hooks the webhook should serve, use multiple times to load from different files")
	flag.Var(&ResponseHeaders, "header", "response header to return, specified in format name=value, use multiple times to set multiple headers")

	flag.Parse()
}
