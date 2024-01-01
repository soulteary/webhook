package rules

import (
	"log"
	"os"

	"github.com/soulteary/webhook/internal/hook"
)

var (
	LoadedHooksFromFiles = make(map[string]hook.Hooks)
	HooksFiles           hook.HooksFiles
)

func RemoveHooks(hooksFilePath string, verbose bool, noPanic bool) {
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

	if !verbose && !noPanic && LenLoadedHooks() == 0 {
		log.SetOutput(os.Stdout)
		log.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to run without the hooks, either use -verbose flag, or -nopanic")
	}
}

func LenLoadedHooks() int {
	sum := 0
	for _, hooks := range LoadedHooksFromFiles {
		sum += len(hooks)
	}

	return sum
}

func MatchLoadedHook(id string) *hook.Hook {
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

		for _, hook := range hooksInFile {
			wasHookIDAlreadyLoaded := false

			for _, loadedHook := range LoadedHooksFromFiles[hooksFilePath] {
				if loadedHook.ID == hook.ID {
					wasHookIDAlreadyLoaded = true
					break
				}
			}

			if (MatchLoadedHook(hook.ID) != nil && !wasHookIDAlreadyLoaded) || seenHooksIds[hook.ID] {
				log.Printf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!", hook.ID)
				log.Println("reverting hooks back to the previous configuration")
				return
			}

			seenHooksIds[hook.ID] = true
			log.Printf("\tloaded: %s\n", hook.ID)
		}

		LoadedHooksFromFiles[hooksFilePath] = hooksInFile
	}
}

func reloadAllHooks(asTemplate bool) {
	for _, hooksFilePath := range HooksFiles {
		ReloadHooks(hooksFilePath, asTemplate)
	}
}

func ReloadAllHooksAsTemplate() {
	reloadAllHooks(true)
}

func ReloadAllHooksNotAsTemplate() {
	reloadAllHooks(false)
}
