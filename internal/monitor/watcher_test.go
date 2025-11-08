package monitor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
)

func TestApplyWatcher(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-hooks.json")

	// Create the file
	err := os.WriteFile(testFile, []byte(`[]`), 0644)
	assert.NoError(t, err)

	// Setup rules
	rules.HooksFiles = []string{testFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Create app flags
	appFlags := flags.AppFlags{
		AsTemplate: false,
		Verbose:    false,
		NoPanic:    false,
	}

	// Apply watcher (this will start a goroutine)
	ApplyWatcher(appFlags)

	// Wait a bit for the watcher to be set up
	// Note: This test mainly verifies that ApplyWatcher doesn't panic
	// The actual file watching is tested in monitor_test.go
}

func TestApplyWatcher_ErrorAddingFile(t *testing.T) {
	// Setup rules with a non-existent file
	rules.HooksFiles = []string{"/nonexistent/file.json"}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Create app flags
	appFlags := flags.AppFlags{
		AsTemplate: false,
		Verbose:    false,
		NoPanic:    false,
	}

	// Apply watcher (should handle error gracefully)
	// This should not panic, but will log an error
	ApplyWatcher(appFlags)
}

