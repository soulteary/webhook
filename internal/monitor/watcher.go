package monitor

import (
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/rules"
)

var watcher *fsnotify.Watcher

func ApplyWatcher(appFlags flags.AppFlags) {
	// -hooks-dir: watch directory for new/changed/removed hook config files (including when dir is empty)
	if appFlags.HooksDir != "" {
		if err := os.MkdirAll(appFlags.HooksDir, 0750); err != nil {
			logger.Fatalf("error creating hooks-dir %s: %v", appFlags.HooksDir, err)
		}
		go WatchDir(appFlags.HooksDir, appFlags.AsTemplate, appFlags.Verbose, appFlags.NoPanic)
		return
	}

	// -hotreload with explicit -hooks: watch each file (watcher kept for process lifetime; do not Close in ApplyWatcher)
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		logger.Fatalf("error creating file watcher instance: %v", err)
	}

	rules.RLockHooksFiles()
	hooksFilesCopy := make([]string, len(rules.HooksFiles))
	copy(hooksFilesCopy, rules.HooksFiles)
	rules.RUnlockHooksFiles()

	for _, hooksFilePath := range hooksFilesCopy {
		logger.Infof("setting up file watcher for %s", hooksFilePath)
		err = watcher.Add(hooksFilePath)
		if err != nil {
			logger.Errorf("error adding hooks file %s to the watcher: %v", hooksFilePath, err)
			return
		}
	}

	removeHooksFn := func(path string, verbose bool, noPanic bool) {
		rules.RemoveHooks(path, verbose, noPanic, false)
	}
	go WatchForFileChange(watcher, appFlags.AsTemplate, appFlags.Verbose, appFlags.NoPanic, rules.ReloadHooks, removeHooksFn)
}

// closeWatcherForTest 关闭全局 watcher，仅用于测试以停止 goroutine、避免与后续测试产生竞态或泄漏。
// 生产代码不应调用（watcher 进程生命周期内不关闭）。
func closeWatcherForTest() {
	if watcher != nil {
		_ = watcher.Close()
		watcher = nil
	}
}
