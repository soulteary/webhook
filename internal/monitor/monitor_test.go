package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestWatchForFileChange(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-hooks.json")

	// Create the file
	err := os.WriteFile(testFile, []byte(`[]`), 0644)
	assert.NoError(t, err)

	// Create a watcher
	watcher, err := fsnotify.NewWatcher()
	assert.NoError(t, err)
	defer func() {
		// Close watcher to stop the goroutine
		watcher.Close()
		// Give goroutine time to exit
		time.Sleep(100 * time.Millisecond)
	}()

	// Add the file to the watcher
	err = watcher.Add(testFile)
	assert.NoError(t, err)

	// Track function calls
	reloadCalled := false

	reloadHooks := func(hooksFilePath string, asTemplate bool) {
		reloadCalled = true
		assert.Equal(t, testFile, hooksFilePath)
	}

	removeHooks := func(hooksFilePath string, verbose bool, noPanic bool) {
		// Not used in this test
	}

	// Start watching in a goroutine
	go WatchForFileChange(watcher, false, false, false, reloadHooks, removeHooks)

	// Wait a bit for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Test Write event
	err = os.WriteFile(testFile, []byte(`[{"id":"test"}]`), 0644)
	assert.NoError(t, err)

	// Wait for the event to be processed
	time.Sleep(200 * time.Millisecond)
	assert.True(t, reloadCalled, "reloadHooks should have been called")

	// Reset for next test
	reloadCalled = false

	// Test Rename event (file overwritten)
	err = os.WriteFile(testFile, []byte(`[{"id":"test2"}]`), 0644)
	assert.NoError(t, err)

	// Wait for the event to be processed
	time.Sleep(300 * time.Millisecond)
	assert.True(t, reloadCalled, "reloadHooks should have been called for overwrite")
}

func TestWatchForFileChange_Remove(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-hooks.json")

	// Create the file
	err := os.WriteFile(testFile, []byte(`[]`), 0644)
	assert.NoError(t, err)

	// Create a watcher
	watcher, err := fsnotify.NewWatcher()
	assert.NoError(t, err)
	defer func() {
		// Close watcher to stop the goroutine
		watcher.Close()
		// Give goroutine time to exit
		time.Sleep(100 * time.Millisecond)
	}()

	// Add the file to the watcher
	err = watcher.Add(testFile)
	assert.NoError(t, err)

	// Track function calls
	removeCalled := false

	reloadHooks := func(hooksFilePath string, asTemplate bool) {
		// Not used in this test
	}

	removeHooks := func(hooksFilePath string, verbose bool, noPanic bool) {
		removeCalled = true
		assert.Equal(t, testFile, hooksFilePath)
	}

	// Start watching in a goroutine
	go WatchForFileChange(watcher, false, false, false, reloadHooks, removeHooks)

	// Wait a bit for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Remove the file
	err = os.Remove(testFile)
	assert.NoError(t, err)

	// Wait for the event to be processed
	time.Sleep(200 * time.Millisecond)
	assert.True(t, removeCalled, "removeHooks should have been called")
}

func TestWatchForFileChange_Error(t *testing.T) {
	// Create a watcher
	watcher, err := fsnotify.NewWatcher()
	assert.NoError(t, err)

	// Track function calls
	reloadHooks := func(hooksFilePath string, asTemplate bool) {}
	removeHooks := func(hooksFilePath string, verbose bool, noPanic bool) {}

	// Start watching in a goroutine
	go WatchForFileChange(watcher, false, false, false, reloadHooks, removeHooks)

	// Wait a bit for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Close the watcher to trigger an error
	watcher.Close()

	// Wait for the error to be processed and goroutine to exit
	time.Sleep(200 * time.Millisecond)
}
