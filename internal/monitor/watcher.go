package monitor

import (
	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/rules"
)

var watcher *fsnotify.Watcher

func ApplyWatcher(appFlags flags.AppFlags) {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		logger.Fatalf("error creating file watcher instance: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// 加锁读取 HooksFiles
	rules.RLockHooksFiles()
	hooksFilesCopy := make([]string, len(rules.HooksFiles))
	copy(hooksFilesCopy, rules.HooksFiles)
	rules.RUnlockHooksFiles()

	for _, hooksFilePath := range hooksFilesCopy {
		// set up file watcher
		logger.Infof("setting up file watcher for %s", hooksFilePath)

		err = watcher.Add(hooksFilePath)
		if err != nil {
			logger.Errorf("error adding hooks file %s to the watcher: %v", hooksFilePath, err)
			return
		}
	}

	go WatchForFileChange(watcher, appFlags.AsTemplate, appFlags.Verbose, appFlags.NoPanic, rules.ReloadHooks, rules.RemoveHooks)
}
