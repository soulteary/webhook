package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/i18n"
	"github.com/soulteary/webhook/internal/monitor"
	"github.com/soulteary/webhook/internal/pidfile"
	"github.com/soulteary/webhook/internal/platform"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/soulteary/webhook/internal/server"
	"github.com/soulteary/webhook/internal/version"
)

var (
	watcher *fsnotify.Watcher
	signals chan os.Signal
	pidFile *pidfile.PIDFile
)

func main() {
	appFlags := flags.Parse()

	i18n.GLOBAL_LOCALES = i18n.InitLocaleByFiles(i18n.LoadLocaleFiles(appFlags.I18nDir))
	i18n.GLOBAL_LANG = appFlags.Lang

	sayHi := i18n.GetMessage("HelloWorld")
	fmt.Println(sayHi)

	if appFlags.ShowVersion {
		fmt.Println("webhook version " + version.Version)
		os.Exit(0)
	}

	if (appFlags.SetUID != 0 || appFlags.SetGID != 0) && (appFlags.SetUID == 0 || appFlags.SetGID == 0) {
		fmt.Println("error: setuid and setgid options must be used together")
		os.Exit(1)
	}

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

	addr := fmt.Sprintf("%s:%d", appFlags.Host, appFlags.Port)

	// Open listener early so we can drop privileges.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logQueue = append(logQueue, fmt.Sprintf("error listening on port: %s", err))
		// we'll bail out below
	}

	if appFlags.SetUID != 0 {
		err := platform.DropPrivileges(appFlags.SetUID, appFlags.SetGID)
		if err != nil {
			logQueue = append(logQueue, fmt.Sprintf("error dropping privileges: %s", err))
			// we'll bail out below
		}
	}

	if appFlags.LogPath != "" {
		file, err := os.OpenFile(appFlags.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			logQueue = append(logQueue, fmt.Sprintf("error opening log file %q: %v", appFlags.LogPath, err))
			// we'll bail out below
		} else {
			log.SetOutput(file)
		}
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
			log.Fatalf("Error creating pidfile: %v", err)
		}

		defer func() {
			// NOTE(moorereason): my testing shows that this doesn't work with
			// ^C, so we also do a Remove in the signal handler elsewhere.
			if nerr := pidFile.Remove(); nerr != nil {
				log.Print(nerr)
			}
		}()
	}

	log.Println("version " + version.Version + " starting")

	// set os signal watcher
	if appFlags.AsTemplate {
		platform.SetupSignals(signals, rules.ReloadAllHooksAsTemplate, pidFile)
	} else {
		platform.SetupSignals(signals, rules.ReloadAllHooksNotAsTemplate, pidFile)
	}

	// load and parse hooks
	for _, hooksFilePath := range rules.HooksFiles {
		log.Printf("attempting to load hooks from %s\n", hooksFilePath)

		newHooks := hook.Hooks{}

		err := newHooks.LoadFromFile(hooksFilePath, appFlags.AsTemplate)
		if err != nil {
			log.Printf("couldn't load hooks from file! %+v\n", err)
		} else {
			log.Printf("found %d hook(s) in file\n", len(newHooks))

			for _, hook := range newHooks {
				if rules.MatchLoadedHook(hook.ID) != nil {
					log.Fatalf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!\n", hook.ID)
				}
				log.Printf("\tloaded: %s\n", hook.ID)
			}

			rules.LoadedHooksFromFiles[hooksFilePath] = newHooks
		}
	}

	newHooksFiles := rules.HooksFiles[:0]
	for _, filePath := range rules.HooksFiles {
		if _, ok := rules.LoadedHooksFromFiles[filePath]; ok {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}

	rules.HooksFiles = newHooksFiles

	if !appFlags.Verbose && !appFlags.NoPanic && rules.LenLoadedHooks() == 0 {
		log.SetOutput(os.Stdout)
		log.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to start without the hooks, either use -verbose flag, or -nopanic")
	}

	if appFlags.HotReload {
		var err error

		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Fatal("error creating file watcher instance\n", err)
		}
		defer watcher.Close()

		for _, hooksFilePath := range rules.HooksFiles {
			// set up file watcher
			log.Printf("setting up file watcher for %s\n", hooksFilePath)

			err = watcher.Add(hooksFilePath)
			if err != nil {
				log.Print("error adding hooks file to the watcher\n", err)
				return
			}
		}

		go monitor.WatchForFileChange(watcher, appFlags.AsTemplate, appFlags.Verbose, appFlags.NoPanic, rules.ReloadHooks, rules.RemoveHooks)
	}

	server.Launch(appFlags, addr, ln)
}
