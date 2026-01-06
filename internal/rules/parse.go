package rules

import (
	"log"

	"github.com/soulteary/webhook/internal/hook"
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
		log.Printf("attempting to load hooks from %s\n", hooksFilePath)

		newHooks := hook.Hooks{}

		err := newHooks.LoadFromFile(hooksFilePath, isAsTemplate)
		if err != nil {
			log.Printf("couldn't load hooks from file! %+v\n", err)
		} else {
			log.Printf("found %d hook(s) in file\n", len(newHooks))

			for _, hook := range newHooks {
				if MatchLoadedHook(hook.ID) != nil {
					log.Fatalf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!\n", hook.ID)
				}
				log.Printf("\tloaded: %s\n", hook.ID)
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
