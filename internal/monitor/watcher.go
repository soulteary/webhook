package monitor

import (
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/rules"
)

var watcher *fsnotify.Watcher

func ApplyWatcher(appFlags flags.AppFlags) {
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

	go WatchForFileChange(watcher, appFlags.AsTemplate, appFlags.Verbose, appFlags.NoPanic, rules.ReloadHooks, rules.RemoveHooks)
}
