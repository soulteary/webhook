package monitor

import (
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/rules"
)

func WatchForFileChange(watcher *fsnotify.Watcher, asTemplate bool, verbose bool, noPanic bool, reloadHooks func(hooksFilePath string, asTemplate bool), removeHooks func(hooksFilePath string, verbose bool, noPanic bool)) {
	for {
		select {
		case event := <-(*watcher).Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("hooks file %s modified\n", event.Name)
				rules.ReloadHooks(event.Name, asTemplate)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				if _, err := os.Stat(event.Name); os.IsNotExist(err) {
					log.Printf("hooks file %s removed, no longer watching this file for changes, removing hooks that were loaded from it\n", event.Name)
					err = (*watcher).Remove(event.Name)
					if err != nil {
						log.Printf("error removing file %s from watcher: %s\n", event.Name, err)
					}
					rules.RemoveHooks(event.Name, verbose, noPanic)
				}
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				time.Sleep(100 * time.Millisecond)
				if _, err := os.Stat(event.Name); os.IsNotExist(err) {
					// file was removed
					log.Printf("hooks file %s removed, no longer watching this file for changes, and removing hooks that were loaded from it\n", event.Name)
					err = (*watcher).Remove(event.Name)
					if err != nil {
						log.Printf("error removing file %s from watcher: %s\n", event.Name, err)
					}
					rules.RemoveHooks(event.Name, verbose, noPanic)
				} else {
					// file was overwritten
					log.Printf("hooks file %s overwritten\n", event.Name)
					rules.ReloadHooks(event.Name, asTemplate)
					err = (*watcher).Remove(event.Name)
					if err != nil {
						log.Printf("error removing file %s from watcher: %s\n", event.Name, err)
					}
					err = (*watcher).Add(event.Name)
					if err != nil {
						log.Printf("error adding file %s to watcher: %s\n", event.Name, err)
					}
				}
			}
		case err := <-(*watcher).Errors:
			log.Println("watcher error:", err)
		}
	}
}
