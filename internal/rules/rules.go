package rules

import (
	"log"
	"os"
	"sync"

	"github.com/soulteary/webhook/internal/hook"
)

var (
	// hooksMutex 保护 LoadedHooksFromFiles 和 HooksFiles 的并发访问
	hooksMutex           sync.RWMutex
	LoadedHooksFromFiles = make(map[string]hook.Hooks)
	HooksFiles           hook.HooksFiles
)

func RemoveHooks(hooksFilePath string, verbose bool, noPanic bool) {
	hooksMutex.Lock()
	defer hooksMutex.Unlock()

	for _, hook := range LoadedHooksFromFiles[hooksFilePath] {
		log.Printf("\tremoving: %s\n", hook.ID)
	}

	newHooksFiles := HooksFiles[:0]
	for _, filePath := range HooksFiles {
		if filePath != hooksFilePath {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}

	HooksFiles = newHooksFiles

	removedHooksCount := len(LoadedHooksFromFiles[hooksFilePath])

	delete(LoadedHooksFromFiles, hooksFilePath)

	log.Printf("removed %d hook(s) that were loaded from file %s\n", removedHooksCount, hooksFilePath)

	if !verbose && !noPanic && lenLoadedHooksLocked() == 0 {
		log.SetOutput(os.Stdout)
		log.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to run without the hooks, either use -verbose flag, or -nopanic")
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

func MatchLoadedHook(id string) *hook.Hook {
	hooksMutex.RLock()
	defer hooksMutex.RUnlock()

	for _, hooks := range LoadedHooksFromFiles {
		if hook := hooks.Match(id); hook != nil {
			return hook
		}
	}

	return nil
}

func ReloadHooks(hooksFilePath string, asTemplate bool) {
	hooksInFile := hook.Hooks{}

	// parse and swap
	log.Printf("attempting to reload hooks from %s\n", hooksFilePath)

	err := hooksInFile.LoadFromFile(hooksFilePath, asTemplate)

	if err != nil {
		log.Printf("couldn't load hooks from file! %+v\n", err)
	} else {
		seenHooksIds := make(map[string]bool)

		log.Printf("found %d hook(s) in file\n", len(hooksInFile))

		// 在加锁前检查重复的 hook ID（需要读取当前加载的 hooks）
		hooksMutex.RLock()
		for _, hook := range hooksInFile {
			wasHookIDAlreadyLoaded := false

			for _, loadedHook := range LoadedHooksFromFiles[hooksFilePath] {
				if loadedHook.ID == hook.ID {
					wasHookIDAlreadyLoaded = true
					break
				}
			}

			// 检查是否在其他文件中已加载
			hookExistsInOtherFile := false
			if !wasHookIDAlreadyLoaded {
				for filePath, hooks := range LoadedHooksFromFiles {
					if filePath != hooksFilePath {
						if matchedHook := hooks.Match(hook.ID); matchedHook != nil {
							hookExistsInOtherFile = true
							break
						}
					}
				}
			}

			if hookExistsInOtherFile || seenHooksIds[hook.ID] {
				hooksMutex.RUnlock()
				log.Printf("error: hook with the id %s has already been loaded from file %s!\nplease check your hooks file for duplicate hooks ids!", hook.ID, hooksFilePath)
				log.Printf("reverting hooks back to the previous configuration (file: %s)", hooksFilePath)
				return
			}

			seenHooksIds[hook.ID] = true
		}
		hooksMutex.RUnlock()

		// 加写锁进行更新
		hooksMutex.Lock()
		for _, hook := range hooksInFile {
			log.Printf("\tloaded: %s\n", hook.ID)
		}
		LoadedHooksFromFiles[hooksFilePath] = hooksInFile
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
