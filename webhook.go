package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/i18n"
	"github.com/soulteary/webhook/internal/monitor"
	"github.com/soulteary/webhook/internal/pidfile"
	"github.com/soulteary/webhook/internal/platform"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/soulteary/webhook/internal/server"
	"github.com/soulteary/webhook/internal/version"
)

var (
	signals chan os.Signal
	pidFile *pidfile.PIDFile
)

//go:embed locales/*.toml
var WebhookLocales embed.FS

func NeedEchoVersionInfo(appFlags flags.AppFlags) {
	if appFlags.ShowVersion {
		i18n.Println(i18n.MSG_WEBHOOK_VERSION, version.Version)
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

func SetupLogger(appFlags flags.AppFlags, logQueue *[]string) (logFile *os.File, err error) {
	if appFlags.LogPath != "" {
		logFile, err = os.OpenFile(appFlags.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			*logQueue = append(*logQueue, i18n.Sprintf(i18n.ERR_SERVER_OPENING_LOG_FILE, appFlags.LogPath, err))
		}
	}
	return logFile, err
}

func main() {
	appFlags := flags.Parse()

	i18n.GLOBAL_LOCALES = i18n.InitLocaleByFiles(i18n.LoadLocaleFiles(appFlags.I18nDir, WebhookLocales))
	i18n.GLOBAL_LANG = appFlags.Lang

	// check if we need to echo version info and quit app
	NeedEchoVersionInfo(appFlags)
	// check if the privileges params are correct, or exit(1)
	CheckPrivilegesParamsCorrect(appFlags)

	if appFlags.Debug || appFlags.LogPath != "" {
		appFlags.Verbose = true
	}

	if len(rules.HooksFiles) == 0 {
		rules.HooksFiles = append(rules.HooksFiles, "hooks.json")
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
	logFile, err := SetupLogger(appFlags, &logQueue)
	if err == nil && logFile != nil {
		log.SetOutput(logFile)
	}
	log.SetPrefix("[webhook] ")
	log.SetFlags(log.Ldate | log.Ltime)

	if len(logQueue) != 0 {
		for i := range logQueue {
			log.Println(logQueue[i])
		}
		os.Exit(1)
	}

	if !appFlags.Verbose {
		log.SetOutput(io.Discard)
	}

	// Create pidfile
	if appFlags.PidPath != "" {
		var err error

		pidFile, err = pidfile.New(appFlags.PidPath)
		if err != nil {
			log.Fatal(i18n.ERR_CREATING_PID_FILE, err)
		}

		defer func() {
			// NOTE(moorereason): my testing shows that this doesn't work with
			// ^C, so we also do a Remove in the signal handler elsewhere.
			if nerr := pidFile.Remove(); nerr != nil {
				log.Print(nerr)
			}
		}()
	}

	log.Println(i18n.Sprintf(i18n.MSG_SERVER_IS_STARTING, version.Version))

	// set os signal watcher
	if appFlags.AsTemplate {
		signals = platform.SetupSignals(signals, rules.ReloadAllHooksAsTemplate, pidFile)
	} else {
		signals = platform.SetupSignals(signals, rules.ReloadAllHooksNotAsTemplate, pidFile)
	}

	// load and parse hooks
	rules.ParseAndLoadHooks(appFlags.AsTemplate)

	if !appFlags.Verbose && !appFlags.NoPanic && rules.LenLoadedHooks() == 0 {
		log.SetOutput(os.Stdout)
		log.Fatalln(i18n.Sprintf(i18n.ERR_COULD_NOT_LOAD_ANY_HOOKS))
	}

	if appFlags.HotReload {
		monitor.ApplyWatcher(appFlags)
	}

	server.Launch(appFlags, addr, *ln)
}
