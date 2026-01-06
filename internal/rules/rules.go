package rules

import (
	"sync"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/logger"
)

var (
	// hooksMutex 保护 LoadedHooksFromFiles、HooksFiles 和 hooksIndex 的并发访问
	hooksMutex           sync.RWMutex
	LoadedHooksFromFiles = make(map[string]hook.Hooks)
	HooksFiles           hook.HooksFiles
	// hooksIndex 是 Hook ID 到 Hook 指针的索引，用于快速查找
	hooksIndex = make(map[string]*hook.Hook)
)

func RemoveHooks(hooksFilePath string, verbose bool, noPanic bool) {
	hooksMutex.Lock()
	defer hooksMutex.Unlock()

	for _, hook := range LoadedHooksFromFiles[hooksFilePath] {
		logger.Debugf("\tremoving: %s", hook.ID)
	}

	newHooksFiles := HooksFiles[:0]
	for _, filePath := range HooksFiles {
		if filePath != hooksFilePath {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}

	HooksFiles = newHooksFiles

	removedHooksCount := len(LoadedHooksFromFiles[hooksFilePath])

	// 删除索引
	removeIndexForFileLocked(hooksFilePath)

	delete(LoadedHooksFromFiles, hooksFilePath)

	logger.Infof("removed %d hook(s) that were loaded from file %s", removedHooksCount, hooksFilePath)

	if !verbose && !noPanic && lenLoadedHooksLocked() == 0 {
		logger.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to run without the hooks, either use -verbose flag, or -nopanic")
	}
}

func LenLoadedHooks() int {
	hooksMutex.RLock()
	defer hooksMutex.RUnlock()
	return lenLoadedHooksLocked()
}

// lenLoadedHooksLocked 在已持有锁的情况下计算 hook 数量（内部使用）
func lenLoadedHooksLocked() int {
	sum := 0
	for _, hooks := range LoadedHooksFromFiles {
		sum += len(hooks)
	}
	return sum
}

// buildIndexLocked 在已持有写锁的情况下建立索引（内部使用）
func buildIndexLocked() {
	hooksIndex = make(map[string]*hook.Hook)
	for _, hooks := range LoadedHooksFromFiles {
		for i := range hooks {
			hooksIndex[hooks[i].ID] = &hooks[i]
		}
	}
}

// BuildIndex 重建索引（用于测试或手动同步）
func BuildIndex() {
	hooksMutex.Lock()
	defer hooksMutex.Unlock()
	buildIndexLocked()
}

// updateIndexForFileLocked 在已持有写锁的情况下更新指定文件的索引（内部使用）
func updateIndexForFileLocked(hooksFilePath string, hooks hook.Hooks) {
	// 先删除该文件原有的 hooks 索引
	if oldHooks, exists := LoadedHooksFromFiles[hooksFilePath]; exists {
		for i := range oldHooks {
			delete(hooksIndex, oldHooks[i].ID)
		}
	}
	// 添加新的 hooks 索引
	for i := range hooks {
		hooksIndex[hooks[i].ID] = &hooks[i]
	}
}

// removeIndexForFileLocked 在已持有写锁的情况下删除指定文件的索引（内部使用）
func removeIndexForFileLocked(hooksFilePath string) {
	if hooks, exists := LoadedHooksFromFiles[hooksFilePath]; exists {
		for i := range hooks {
			delete(hooksIndex, hooks[i].ID)
		}
	}
}

func MatchLoadedHook(id string) *hook.Hook {
	hooksMutex.RLock()

	// 如果索引中有，直接返回
	if hook := hooksIndex[id]; hook != nil {
		hooksMutex.RUnlock()
		return hook
	}

	// 如果索引为空但 LoadedHooksFromFiles 不为空，说明索引不同步
	// 这种情况不应该发生，但作为后备方案，我们重建索引
	needRebuild := len(hooksIndex) == 0 && len(LoadedHooksFromFiles) > 0
	hooksMutex.RUnlock()

	// 如果需要重建索引，先获取写锁重建
	if needRebuild {
		hooksMutex.Lock()
		// 再次检查，可能其他 goroutine 已经重建了
		if len(hooksIndex) == 0 && len(LoadedHooksFromFiles) > 0 {
			buildIndexLocked()
		}
		hooksMutex.Unlock()

		// 重建后再次尝试查找
		hooksMutex.RLock()
		hook := hooksIndex[id]
		hooksMutex.RUnlock()
		return hook
	}

	return nil
}

func ReloadHooks(hooksFilePath string, asTemplate bool) {
	hooksInFile := hook.Hooks{}

	// parse and swap
	logger.Infof("attempting to reload hooks from %s", hooksFilePath)

	err := hooksInFile.LoadFromFile(hooksFilePath, asTemplate)

	if err != nil {
		logger.Errorf("couldn't load hooks from file! %+v", err)
	} else {
		seenHooksIds := make(map[string]bool)

		logger.Infof("found %d hook(s) in file", len(hooksInFile))

		// 在加锁前检查重复的 hook ID（需要读取当前加载的 hooks）
		hooksMutex.RLock()
		// 构建当前文件中的旧 hook ID 集合（用于重载场景，允许在当前文件中重复）
		oldHookIDsInFile := make(map[string]bool)
		if oldHooks, exists := LoadedHooksFromFiles[hooksFilePath]; exists {
			for i := range oldHooks {
				oldHookIDsInFile[oldHooks[i].ID] = true
			}
		}

		for _, hook := range hooksInFile {
			// 检查是否在当前文件中已存在（允许，因为是重载）
			wasHookIDAlreadyLoaded := oldHookIDsInFile[hook.ID]

			// 使用索引检查是否在其他文件中已加载（更高效）
			hookExistsInOtherFile := false
			if !wasHookIDAlreadyLoaded {
				// 如果索引中存在该 ID，说明它来自其他文件（因为当前文件的旧 hooks 已经在索引中，但我们已经排除了）
				if _, exists := hooksIndex[hook.ID]; exists {
					hookExistsInOtherFile = true
				}
			}

			// 检查是否在当前文件中有重复的 ID
			if seenHooksIds[hook.ID] {
				hooksMutex.RUnlock()
				logger.Errorf("error: hook with the id %s has already been loaded from file %s! please check your hooks file for duplicate hooks ids!", hook.ID, hooksFilePath)
				logger.Warnf("reverting hooks back to the previous configuration (file: %s)", hooksFilePath)
				return
			}

			// 检查是否在其他文件中已存在
			if hookExistsInOtherFile {
				hooksMutex.RUnlock()
				logger.Errorf("error: hook with the id %s has already been loaded from file %s! please check your hooks file for duplicate hooks ids!", hook.ID, hooksFilePath)
				logger.Warnf("reverting hooks back to the previous configuration (file: %s)", hooksFilePath)
				return
			}

			seenHooksIds[hook.ID] = true
		}
		hooksMutex.RUnlock()

		// 加写锁进行更新
		hooksMutex.Lock()
		for _, hook := range hooksInFile {
			logger.Debugf("\tloaded: %s", hook.ID)
		}
		LoadedHooksFromFiles[hooksFilePath] = hooksInFile
		// 更新索引
		updateIndexForFileLocked(hooksFilePath, hooksInFile)
		hooksMutex.Unlock()
	}
}

func reloadAllHooks(asTemplate bool) {
	hooksMutex.RLock()
	hooksFilesCopy := make([]string, len(HooksFiles))
	copy(hooksFilesCopy, HooksFiles)
	hooksMutex.RUnlock()

	for _, hooksFilePath := range hooksFilesCopy {
		ReloadHooks(hooksFilePath, asTemplate)
	}
}

func ReloadAllHooksAsTemplate() {
	reloadAllHooks(true)
}

func ReloadAllHooksNotAsTemplate() {
	reloadAllHooks(false)
}

// RLockHooksFiles 获取 HooksFiles 的读锁（用于外部包访问）
func RLockHooksFiles() {
	hooksMutex.RLock()
}

// RUnlockHooksFiles 释放 HooksFiles 的读锁（用于外部包访问）
func RUnlockHooksFiles() {
	hooksMutex.RUnlock()
}

// LockHooksFiles 获取 HooksFiles 的写锁（用于外部包访问）
func LockHooksFiles() {
	hooksMutex.Lock()
}

// UnlockHooksFiles 释放 HooksFiles 的写锁（用于外部包访问）
func UnlockHooksFiles() {
	hooksMutex.Unlock()
}
