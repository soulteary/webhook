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

// RemoveHooks removes hooks loaded from the given file. When allowZeroHooks is true (e.g. -hooks-dir mode),
// having zero hooks after removal does not cause exit.
func RemoveHooks(hooksFilePath string, verbose bool, noPanic bool, allowZeroHooks bool) {
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

	if !allowZeroHooks && !verbose && !noPanic && lenLoadedHooksLocked() == 0 {
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

// updateIndexForFileLocked atomically replaces a file's hooks and index entries.
func updateIndexForFileLocked(hooksFilePath string, hooks hook.Hooks) {
	// 先删除该文件原有的 hooks 索引
	if oldHooks, exists := LoadedHooksFromFiles[hooksFilePath]; exists {
		for i := range oldHooks {
			delete(hooksIndex, oldHooks[i].ID)
		}
	}
	LoadedHooksFromFiles[hooksFilePath] = hooks
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
	ReloadHooksWithOptions(hooksFilePath, asTemplate, LoadOptions{})
}

// ReloadHooksWithOptions parses and validates a candidate before replacing
// the currently active hooks. A rejected candidate leaves the old ruleset in
// place.
func ReloadHooksWithOptions(hooksFilePath string, asTemplate bool, options LoadOptions) {
	logger.Infof("attempting to reload hooks from %s", hooksFilePath)

	hooksInFile, err := loadHooksFile(hooksFilePath, asTemplate, options)
	if err != nil {
		logger.Errorf("couldn't load or validate hooks from file; keeping previous configuration: %+v", err)
		return
	}
	logger.Infof("found %d hook(s) in file", len(hooksInFile))

	// Check the prospective aggregate and duplicate namespace under the same
	// write lock used for the swap, so concurrent reloads cannot invalidate the
	// decision before it is committed.
	hooksMutex.Lock()
	defer hooksMutex.Unlock()
	if options.RequireNonEmpty && prospectiveHookCountLocked(hooksFilePath, hooksInFile) == 0 {
		logger.Errorf("couldn't reload hooks from file %s: explicit hook configuration must contain at least one hook; keeping previous configuration", hooksFilePath)
		return
	}

	oldHookIDsInFile := make(map[string]bool)
	if oldHooks, exists := LoadedHooksFromFiles[hooksFilePath]; exists {
		for i := range oldHooks {
			oldHookIDsInFile[oldHooks[i].ID] = true
		}
	}
	seenHookIDs := make(map[string]bool)
	for _, configuredHook := range hooksInFile {
		if seenHookIDs[configuredHook.ID] || (!oldHookIDsInFile[configuredHook.ID] && hooksIndex[configuredHook.ID] != nil) {
			logger.Errorf("error: hook with the id %s has already been loaded from file %s! please check your hooks file for duplicate hooks ids!", configuredHook.ID, hooksFilePath)
			logger.Warnf("reverting hooks back to the previous configuration (file: %s)", hooksFilePath)
			return
		}
		seenHookIDs[configuredHook.ID] = true
		logger.Debugf("\tloaded: %s", configuredHook.ID)
	}
	updateIndexForFileLocked(hooksFilePath, hooksInFile)
}

func reloadAllHooks(asTemplate bool, options LoadOptions) {
	hooksMutex.RLock()
	hooksFilesCopy := make([]string, len(HooksFiles))
	copy(hooksFilesCopy, HooksFiles)
	hooksMutex.RUnlock()

	for _, hooksFilePath := range hooksFilesCopy {
		ReloadHooksWithOptions(hooksFilePath, asTemplate, options)
	}
}

func ReloadAllHooksAsTemplate() {
	reloadAllHooks(true, LoadOptions{})
}

func ReloadAllHooksNotAsTemplate() {
	reloadAllHooks(false, LoadOptions{})
}

// ReloadAllHooksWithOptions applies the active startup policy to every file.
func ReloadAllHooksWithOptions(asTemplate bool, options LoadOptions) {
	reloadAllHooks(asTemplate, options)
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
