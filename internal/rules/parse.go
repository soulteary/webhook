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
			LoadedHooksFromFiles[hooksFilePath] = newHooks
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
	hooksMutex.Lock()
	for _, p := range HooksFiles {
		if p == hooksFilePath {
			hooksMutex.Unlock()
			ReloadHooks(hooksFilePath, isAsTemplate)
			return
		}
	}
	HooksFiles = append(HooksFiles, hooksFilePath)
	hooksMutex.Unlock()

	logger.Infof("attempting to load hooks from %s", hooksFilePath)
	newHooks := hook.Hooks{}
	err := newHooks.LoadFromFile(hooksFilePath, isAsTemplate)
	if err != nil {
		logger.Errorf("couldn't load hooks from file! %+v", err)
		return
	}
	logger.Infof("found %d hook(s) in file", len(newHooks))
	for _, h := range newHooks {
		if MatchLoadedHook(h.ID) != nil {
			logger.Errorf("error: hook with the id %s has already been loaded! skipping file %s", h.ID, hooksFilePath)
			hooksMutex.Lock()
			newList := HooksFiles[:0]
			for _, p := range HooksFiles {
				if p != hooksFilePath {
					newList = append(newList, p)
				}
			}
			HooksFiles = newList
			hooksMutex.Unlock()
			return
		}
		logger.Debugf("\tloaded: %s", h.ID)
	}
	hooksMutex.Lock()
	LoadedHooksFromFiles[hooksFilePath] = newHooks
	updateIndexForFileLocked(hooksFilePath, newHooks)
	hooksMutex.Unlock()
}
