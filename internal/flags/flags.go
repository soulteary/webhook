package flags

import (
	"flag"
	"fmt"
	"strings"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
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
	flags.SetGID = fn.GetEnvInt(ENV_KEY_GID, DEFAULT_GID)
	flags.SetUID = fn.GetEnvInt(ENV_KEY_UID, DEFAULT_UID)
	flags.HttpMethods = fn.GetEnvStr(ENV_KEY_HTTP_METHODS, DEFAULT_HTTP_METHODS)
	flags.PidPath = fn.GetEnvStr(ENV_KEY_PID_FILE, DEFAULT_PID_FILE)

	// init i18n, set lang and i18n dir
	flags.Lang = fn.GetEnvStr(ENV_KEY_LANG, DEFAULT_LANG)
	flags.I18nDir = fn.GetEnvStr(ENV_KEY_I18N, DEFAULT_I18N_DIR)

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

func ParseCLI(flags AppFlags) AppFlags {
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

		Lang    = flag.String("lang", DEFAULT_LANG, "set the language code for the webhook")
		I18nDir = flag.String("lang-dir", DEFAULT_I18N_DIR, "set the directory for the i18n files")

		ShowVersion     = flag.Bool("version", false, "display webhook version and quit")
		ResponseHeaders hook.ResponseHeaders
	)

	hooksFiles := rules.HooksFiles
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
	rules.HooksFiles = flags.HooksFiles

	if len(ResponseHeaders) > 0 {
		flags.ResponseHeaders = ResponseHeaders
	}

	if *Lang != DEFAULT_LANG {
		flags.Lang = *Lang
	}

	if *I18nDir != DEFAULT_I18N_DIR {
		flags.I18nDir = *I18nDir
	}
	return flags
}

func Parse() AppFlags {
	envs := ParseEnvs()
	cli := ParseCLI(envs)
	return cli
}
