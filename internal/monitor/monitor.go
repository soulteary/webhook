package monitor

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// debounceDelay 是文件修改事件去重的延迟时间
	debounceDelay = 300 * time.Millisecond
	// maxRetries 是重试的最大次数
	maxRetries = 3
	// retryDelay 是重试之间的延迟时间
	retryDelay = 100 * time.Millisecond
	// eventBufferSize 是事件缓冲队列的大小
	eventBufferSize = 100
)

// fileProcessor 用于管理每个文件的处理状态
type fileProcessor struct {
	mu            sync.Mutex
	debounceTimer *time.Timer
	processing    bool
}

// WatchForFileChange 监控文件变化，支持事件去重、并发保护和重试机制
func WatchForFileChange(watcher *fsnotify.Watcher, asTemplate bool, verbose bool, noPanic bool, reloadHooks func(hooksFilePath string, asTemplate bool), removeHooks func(hooksFilePath string, verbose bool, noPanic bool)) {
	// 使用带缓冲的 channel 来缓冲事件，避免事件丢失
	eventQueue := make(chan fsnotify.Event, eventBufferSize)

	// 文件处理器映射，用于管理每个文件的去重定时器和处理状态
	processors := make(map[string]*fileProcessor)
	var processorsMu sync.RWMutex

	// 启动事件接收 goroutine，将事件放入缓冲队列
	go func() {
		for {
			select {
			case event, ok := <-(*watcher).Events:
				if !ok {
					return
				}
				// 非阻塞地尝试将事件放入队列
				select {
				case eventQueue <- event:
				default:
					log.Printf("warning: event queue full, dropping event for file %s (operation: %v)", event.Name, event.Op)
				}
			case err, ok := <-(*watcher).Errors:
				if !ok {
					return
				}
				// 只在有实际错误时才记录，nil 表示 channel 已关闭或没有错误
				if err != nil {
					log.Printf("watcher error (asTemplate: %v): %v", asTemplate, err)
				}
			}
		}
	}()

	// 处理事件的主循环
	for {
		select {
		case event, ok := <-eventQueue:
			if !ok {
				return
			}
			handleEvent(event, watcher, asTemplate, verbose, noPanic, reloadHooks, removeHooks, &processors, &processorsMu)
		}
	}
}

// handleEvent 处理单个文件系统事件
func handleEvent(event fsnotify.Event, watcher *fsnotify.Watcher, asTemplate bool, verbose bool, noPanic bool, reloadHooks func(hooksFilePath string, asTemplate bool), removeHooks func(hooksFilePath string, verbose bool, noPanic bool), processors *map[string]*fileProcessor, processorsMu *sync.RWMutex) {
	fileName := event.Name

	if event.Op&fsnotify.Write == fsnotify.Write {
		// 文件写入事件：使用 debounce 机制
		processorsMu.Lock()
		processor, exists := (*processors)[fileName]
		if !exists {
			processor = &fileProcessor{}
			(*processors)[fileName] = processor
		}
		processorsMu.Unlock()

		processor.mu.Lock()
		// 如果已有定时器，先停止它
		if processor.debounceTimer != nil {
			processor.debounceTimer.Stop()
		}
		// 创建新的定时器，延迟处理
		processor.debounceTimer = time.AfterFunc(debounceDelay, func() {
			processor.mu.Lock()
			defer processor.mu.Unlock()

			if processor.processing {
				return
			}
			processor.processing = true

			log.Printf("hooks file %s modified\n", fileName)
			// 使用重试机制执行 reloadHooks
			retryReloadHooks(fileName, asTemplate, reloadHooks)

			processor.processing = false
			processor.debounceTimer = nil
		})
		processor.mu.Unlock()

	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		// 文件删除事件：立即处理，不需要 debounce
		processorsMu.Lock()
		processor, exists := (*processors)[fileName]
		if exists {
			processor.mu.Lock()
			if processor.debounceTimer != nil {
				processor.debounceTimer.Stop()
				processor.debounceTimer = nil
			}
			processor.mu.Unlock()
			delete(*processors, fileName)
		}
		processorsMu.Unlock()

		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			log.Printf("hooks file %s removed, no longer watching this file for changes, removing hooks that were loaded from it\n", fileName)
			err = (*watcher).Remove(fileName)
			if err != nil {
				log.Printf("error removing file %s from watcher (operation: Remove, event: %v): %v", fileName, event.Op, err)
			}
			removeHooks(fileName, verbose, noPanic)
		}

	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		// 文件重命名事件：需要等待一段时间后检查文件状态
		processorsMu.Lock()
		processor, exists := (*processors)[fileName]
		if exists {
			processor.mu.Lock()
			if processor.debounceTimer != nil {
				processor.debounceTimer.Stop()
				processor.debounceTimer = nil
			}
			processor.mu.Unlock()
		}
		processorsMu.Unlock()

		// 使用 goroutine 异步处理，避免阻塞事件循环
		go func() {
			time.Sleep(100 * time.Millisecond)
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				// file was removed
				log.Printf("hooks file %s removed, no longer watching this file for changes, and removing hooks that were loaded from it\n", fileName)
				err = (*watcher).Remove(fileName)
				if err != nil {
					log.Printf("error removing file %s from watcher (operation: Remove, event: %v): %v", fileName, event.Op, err)
				}
				removeHooks(fileName, verbose, noPanic)

				// 清理处理器
				processorsMu.Lock()
				delete(*processors, fileName)
				processorsMu.Unlock()
			} else {
				// file was overwritten
				log.Printf("hooks file %s overwritten\n", fileName)
				retryReloadHooks(fileName, asTemplate, reloadHooks)

				err = (*watcher).Remove(fileName)
				if err != nil {
					log.Printf("error removing file %s from watcher (operation: Remove after overwrite, event: %v): %v", fileName, event.Op, err)
				}
				err = (*watcher).Add(fileName)
				if err != nil {
					log.Printf("error adding file %s to watcher (operation: Add after overwrite, event: %v): %v", fileName, event.Op, err)
				}
			}
		}()
	}
}

// retryReloadHooks 使用重试机制执行 reloadHooks
func retryReloadHooks(hooksFilePath string, asTemplate bool, reloadHooks func(hooksFilePath string, asTemplate bool)) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 执行 reloadHooks
		// 注意：由于 reloadHooks 没有返回值，我们无法直接判断是否成功
		// 这里我们假设它总是成功，如果需要更精确的错误处理，需要修改 reloadHooks 的签名
		reloadHooks(hooksFilePath, asTemplate)

		// 如果这是最后一次尝试，直接返回
		if attempt == maxRetries-1 {
			return
		}

		// 等待一段时间后重试（仅在非最后一次尝试时）
		time.Sleep(retryDelay)
	}
}
