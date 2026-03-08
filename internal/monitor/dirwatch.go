package monitor

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/hooksdir"
	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/rules"
)

const (
	dirWatchDebounce = 400 * time.Millisecond
)

// isHookFile returns true if the base name has a hook config extension.
func isHookFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return hooksdir.HookExts[ext]
}

// WatchDir watches hooksDir for new/changed/removed hook config files and calls add/reload/remove.
// When a new file appears, AddAndLoadHooksFile is called; when a file is modified, ReloadHooks; when removed, RemoveHooks.
func WatchDir(hooksDir string, asTemplate bool, verbose bool, noPanic bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatalf("error creating file watcher for hooks-dir: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	dirAbs, err := filepath.Abs(hooksDir)
	if err != nil {
		dirAbs = filepath.Clean(hooksDir)
	}

	err = watcher.Add(hooksDir)
	if err != nil {
		logger.Fatalf("error adding hooks-dir %s to watcher: %v", hooksDir, err)
	}
	logger.Infof("watching hooks-dir %s for hook config files", hooksDir)

	removeHooksFn := func(path string, v bool, np bool) {
		rules.RemoveHooks(path, v, np, true)
	}

	processors := make(map[string]*fileProcessor)
	var processorsMu sync.RWMutex

	eventQueue := make(chan fsnotify.Event, eventBufferSize)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				select {
				case eventQueue <- event:
				default:
					logger.Warnf("dir watch event queue full, dropping event for %s", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if err != nil {
					logger.Errorf("hooks-dir watcher error: %v", err)
				}
			}
		}
	}()

	for event := range eventQueue {
		path := event.Name
		// Normalize to absolute so comparison with rules.HooksFiles (from ScanHookFiles) is consistent.
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			pathAbs = filepath.Clean(path)
		}
		rel, err := filepath.Rel(dirAbs, pathAbs)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		base := filepath.Base(pathAbs)
		if base == "" || base == "." {
			continue
		}

		if event.Op&fsnotify.Create == fsnotify.Create {
			if !isHookFile(base) {
				continue
			}
			processorsMu.Lock()
			p, exists := processors[pathAbs]
			if !exists {
				p = &fileProcessor{}
				processors[pathAbs] = p
			}
			processorsMu.Unlock()
			p.mu.Lock()
			if p.debounceTimer != nil {
				p.debounceTimer.Stop()
			}
			p.debounceTimer = time.AfterFunc(dirWatchDebounce, func() {
				p.mu.Lock()
				defer p.mu.Unlock()
				if p.processing {
					return
				}
				p.processing = true
				logger.Infof("new hook config file %s", pathAbs)
				rules.AddAndLoadHooksFile(pathAbs, asTemplate)
				p.processing = false
				p.debounceTimer = nil
			})
			p.mu.Unlock()
		} else if event.Op&fsnotify.Write == fsnotify.Write {
			if !isHookFile(base) {
				continue
			}
			rules.RLockHooksFiles()
			fileInList := false
			for _, f := range rules.HooksFiles {
				if f == pathAbs {
					fileInList = true
					break
				}
			}
			rules.RUnlockHooksFiles()
			if !fileInList {
				continue
			}
			processorsMu.Lock()
			p, exists := processors[pathAbs]
			if !exists {
				p = &fileProcessor{}
				processors[pathAbs] = p
			}
			processorsMu.Unlock()
			p.mu.Lock()
			if p.debounceTimer != nil {
				p.debounceTimer.Stop()
			}
			p.debounceTimer = time.AfterFunc(debounceDelay, func() {
				p.mu.Lock()
				defer p.mu.Unlock()
				if p.processing {
					return
				}
				p.processing = true
				logger.Infof("hooks file %s modified", pathAbs)
				retryReloadHooks(pathAbs, asTemplate, rules.ReloadHooks)
				p.processing = false
				p.debounceTimer = nil
			})
			p.mu.Unlock()
		} else if event.Op&fsnotify.Remove == fsnotify.Remove {
			if !isHookFile(base) {
				continue
			}
			rules.RLockHooksFiles()
			fileInList := false
			for _, f := range rules.HooksFiles {
				if f == pathAbs {
					fileInList = true
					break
				}
			}
			rules.RUnlockHooksFiles()
			if fileInList {
				logger.Infof("hooks file %s removed", pathAbs)
				removeHooksFn(pathAbs, verbose, noPanic)
			}
			processorsMu.Lock()
			delete(processors, pathAbs)
			processorsMu.Unlock()
		}
	}
}
