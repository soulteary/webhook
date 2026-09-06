package rules

import (
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/logger"
)

func ParseAndLoadHooks(isAsTemplate bool) {
	// 加读锁读取 HooksFiles
	hooksMutex.RLock()
	hooksFilesCopy := make([]string, len(HooksFiles))
	copy(hooksFilesCopy, HooksFiles)
	// 检查索引是否需要重建（如果索引为空但数据存在，说明索引不同步）
	needRebuildIndex := len(hooksIndex) == 0 && len(LoadedHooksFromFiles) > 0
	hooksMutex.RUnlock()

	// 如果需要，重建索引
	if needRebuildIndex {
		hooksMutex.Lock()
		buildIndexLocked()
		hooksMutex.Unlock()
	}

	// load and parse hooks
	for _, hooksFilePath := range hooksFilesCopy {
		logger.Infof("attempting to load hooks from %s", hooksFilePath)

		newHooks := hook.Hooks{}

		err := newHooks.LoadFromFile(hooksFilePath, isAsTemplate)
		if err != nil {
			logger.Errorf("couldn't load hooks from file! %+v", err)
		} else {
			logger.Infof("found %d hook(s) in file", len(newHooks))

			for _, hook := range newHooks {
				if MatchLoadedHook(hook.ID) != nil {
					logger.Fatalf("error: hook with the id %s has already been loaded! please check your hooks file for duplicate hooks ids!", hook.ID)
				}
				logger.Debugf("\tloaded: %s", hook.ID)
			}

			// 加写锁更新 LoadedHooksFromFiles
			hooksMutex.Lock()
			// 更新索引
			updateIndexForFileLocked(hooksFilePath, newHooks)
			hooksMutex.Unlock()
		}
	}

	// 加写锁更新 HooksFiles
	hooksMutex.Lock()
	newHooksFiles := HooksFiles[:0]
	for _, filePath := range HooksFiles {
		if _, ok := LoadedHooksFromFiles[filePath]; ok {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}
	HooksFiles = newHooksFiles
	hooksMutex.Unlock()
}

// AddAndLoadHooksFile adds a hook config file path to HooksFiles and loads it.
// If the path is already in HooksFiles, ReloadHooks is not called by this function (caller may call ReloadHooks separately).
// Used when watching -hooks-dir and a new file appears.
func AddAndLoadHooksFile(hooksFilePath string, isAsTemplate bool) {
	AddAndLoadHooksFileWithOptions(hooksFilePath, isAsTemplate, LoadOptions{})
}

// AddAndLoadHooksFileWithOptions validates a new file before adding it to the
// watched ruleset. Existing files use the same protected reload path.
func AddAndLoadHooksFileWithOptions(hooksFilePath string, isAsTemplate bool, options LoadOptions) {
	hooksMutex.Lock()
	for _, p := range HooksFiles {
		if p == hooksFilePath {
			hooksMutex.Unlock()
			ReloadHooksWithOptions(hooksFilePath, isAsTemplate, options)
			return
		}
	}
	hooksMutex.Unlock()

	logger.Infof("attempting to load hooks from %s", hooksFilePath)
	newHooks, err := loadHooksFile(hooksFilePath, isAsTemplate, options)
	if err != nil {
		logger.Errorf("couldn't load or validate hooks from file; skipping file: %+v", err)
		return
	}
	logger.Infof("found %d hook(s) in file", len(newHooks))
	hooksMutex.Lock()
	for _, p := range HooksFiles {
		if p == hooksFilePath {
			hooksMutex.Unlock()
			ReloadHooksWithOptions(hooksFilePath, isAsTemplate, options)
			return
		}
	}
	if options.RequireNonEmpty && prospectiveHookCountLocked(hooksFilePath, newHooks) == 0 {
		hooksMutex.Unlock()
		logger.Errorf("couldn't add hooks file %s: explicit hook configuration must contain at least one hook", hooksFilePath)
		return
	}
	seenIDs := make(map[string]bool, len(newHooks))
	for _, h := range newHooks {
		if seenIDs[h.ID] || hooksIndex[h.ID] != nil {
			hooksMutex.Unlock()
			logger.Errorf("error: hook with the id %s has already been loaded! skipping file %s", h.ID, hooksFilePath)
			return
		}
		seenIDs[h.ID] = true
		logger.Debugf("\tloaded: %s", h.ID)
	}
	HooksFiles = append(HooksFiles, hooksFilePath)
	updateIndexForFileLocked(hooksFilePath, newHooks)
	hooksMutex.Unlock()
}
