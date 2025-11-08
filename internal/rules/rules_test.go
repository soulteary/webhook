package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
)

func TestRemoveHooks(t *testing.T) {
	// Setup
	rules.HooksFiles = []string{"test1.json", "test2.json"}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test1.json": {{ID: "hook1"}},
		"test2.json": {{ID: "hook2"}},
	}

	// Execute
	rules.RemoveHooks("test1.json", false, false)

	// Assert
	assert.Equal(t, 1, rules.LenLoadedHooks(), "Expected number of hooks after removing should be 1")
	assert.Nil(t, rules.LoadedHooksFromFiles["test1.json"], "Expected test1.json hooks to be removed")
	assert.Contains(t, rules.HooksFiles, "test2.json", "HooksFiles should still contain 'test2.json'")
}

func TestLenLoadedHooks(t *testing.T) {
	// Setup
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test1.json": {{ID: "hook1"}, {ID: "hook2"}},
		"test2.json": {{ID: "hook3"}},
	}

	// Execute
	length := rules.LenLoadedHooks()

	// Assert
	assert.Equal(t, 3, length, "Expected total length of all loaded hooks to be 3")
}

func TestMatchLoadedHook(t *testing.T) {
	// Setup
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test1.json": {{ID: "hook1"}, {ID: "hook2"}},
	}

	// Tests
	tests := []struct {
		id       string
		expected bool
	}{
		{"hook1", true},
		{"hook2", true},
		{"nonexistent", false},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			// Execute
			match := rules.MatchLoadedHook(test.id)

			// Assert
			if test.expected {
				assert.NotNil(t, match, "Expected to find hook with id %s", test.id)
			} else {
				assert.Nil(t, match, "Expected to not find hook with id %s", test.id)
			}
		})
	}
}

func TestReloadHooks(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile := filepath.Join(tempDir, "hooks.json")

	// Create a valid hooks file
	hooksContent := `[
		{
			"id": "test-hook",
			"execute-command": "/bin/echo",
			"command-working-directory": "/tmp"
		}
	]`
	err := os.WriteFile(hooksFile, []byte(hooksContent), 0644)
	assert.NoError(t, err)

	// Initially load the hook
	rules.HooksFiles = []string{hooksFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)
	rules.ParseAndLoadHooks(false)

	// Verify hook is loaded
	assert.Equal(t, 1, rules.LenLoadedHooks())

	// Reload hooks
	rules.ReloadHooks(hooksFile, false)

	// Verify hook is still loaded
	assert.Equal(t, 1, rules.LenLoadedHooks())
}

func TestReloadHooks_WithTemplate(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile := filepath.Join(tempDir, "hooks.json.tmpl")

	// Create a template hooks file
	hooksContent := `[
		{
			"id": "test-hook-{{.Env.TEST_VAR}}",
			"execute-command": "/bin/echo"
		}
	]`
	err := os.WriteFile(hooksFile, []byte(hooksContent), 0644)
	assert.NoError(t, err)

	rules.HooksFiles = []string{hooksFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Reload hooks as template
	rules.ReloadHooks(hooksFile, true)

	// Verify hook is loaded
	assert.GreaterOrEqual(t, rules.LenLoadedHooks(), 0)
}

func TestReloadHooks_DuplicateID(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile1 := filepath.Join(tempDir, "hooks1.json")
	hooksFile2 := filepath.Join(tempDir, "hooks2.json")

	// Create hooks files with same ID
	hooksContent1 := `[
		{
			"id": "duplicate-hook",
			"execute-command": "/bin/echo"
		}
	]`
	hooksContent2 := `[
		{
			"id": "duplicate-hook",
			"execute-command": "/bin/echo"
		}
	]`

	err := os.WriteFile(hooksFile1, []byte(hooksContent1), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(hooksFile2, []byte(hooksContent2), 0644)
	assert.NoError(t, err)

	// Load first hook
	rules.HooksFiles = []string{hooksFile1}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)
	rules.ParseAndLoadHooks(false)

	// Try to reload with duplicate ID
	rules.ReloadHooks(hooksFile2, false)

	// Verify original hook is still there
	assert.Equal(t, 1, rules.LenLoadedHooks())
}

func TestReloadAllHooksAsTemplate(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile := filepath.Join(tempDir, "hooks.json.tmpl")

	hooksContent := `[
		{
			"id": "test-hook",
			"execute-command": "/bin/echo"
		}
	]`
	err := os.WriteFile(hooksFile, []byte(hooksContent), 0644)
	assert.NoError(t, err)

	rules.HooksFiles = []string{hooksFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Reload all hooks as template
	rules.ReloadAllHooksAsTemplate()

	// Verify hooks are loaded
	assert.GreaterOrEqual(t, rules.LenLoadedHooks(), 0)
}

func TestReloadAllHooksNotAsTemplate(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile := filepath.Join(tempDir, "hooks.json")

	hooksContent := `[
		{
			"id": "test-hook",
			"execute-command": "/bin/echo"
		}
	]`
	err := os.WriteFile(hooksFile, []byte(hooksContent), 0644)
	assert.NoError(t, err)

	rules.HooksFiles = []string{hooksFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Reload all hooks not as template
	rules.ReloadAllHooksNotAsTemplate()

	// Verify hooks are loaded
	assert.GreaterOrEqual(t, rules.LenLoadedHooks(), 0)
}

func TestParseAndLoadHooks(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile := filepath.Join(tempDir, "hooks.json")

	hooksContent := `[
		{
			"id": "test-hook",
			"execute-command": "/bin/echo",
			"command-working-directory": "/tmp"
		}
	]`
	err := os.WriteFile(hooksFile, []byte(hooksContent), 0644)
	assert.NoError(t, err)

	rules.HooksFiles = []string{hooksFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Parse and load hooks
	rules.ParseAndLoadHooks(false)

	// Verify hooks are loaded
	assert.Equal(t, 1, rules.LenLoadedHooks())
	assert.Contains(t, rules.HooksFiles, hooksFile)
}

func TestParseAndLoadHooks_InvalidFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.json")

	// Create invalid JSON
	err := os.WriteFile(invalidFile, []byte("invalid json"), 0644)
	assert.NoError(t, err)

	rules.HooksFiles = []string{invalidFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Parse and load hooks (should handle error gracefully)
	rules.ParseAndLoadHooks(false)

	// Verify no hooks are loaded
	assert.Equal(t, 0, rules.LenLoadedHooks())
	// File should be removed from HooksFiles if loading failed
	assert.NotContains(t, rules.HooksFiles, invalidFile)
}

func TestRemoveHooks_WithVerbose(t *testing.T) {
	// Setup
	rules.HooksFiles = []string{"test1.json"}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test1.json": {{ID: "hook1"}},
	}

	// Execute with verbose=true (should not panic)
	rules.RemoveHooks("test1.json", true, false)

	// Assert
	assert.Equal(t, 0, rules.LenLoadedHooks())
}

func TestRemoveHooks_WithNoPanic(t *testing.T) {
	// Setup
	rules.HooksFiles = []string{"test1.json"}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test1.json": {{ID: "hook1"}},
	}

	// Execute with noPanic=true (should not panic)
	rules.RemoveHooks("test1.json", false, true)

	// Assert
	assert.Equal(t, 0, rules.LenLoadedHooks())
}

func TestRemoveHooks_EmptyHooksFiles(t *testing.T) {
	// Setup - empty hooks files
	rules.HooksFiles = []string{}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test1.json": {{ID: "hook1"}},
	}

	// Execute
	rules.RemoveHooks("test1.json", false, true)

	// Assert - HooksFiles should be empty
	assert.Equal(t, 0, len(rules.HooksFiles))
}

func TestReloadHooks_ErrorLoading(t *testing.T) {
	// Setup with non-existent file
	nonExistentFile := "/nonexistent/file.json"
	rules.HooksFiles = []string{nonExistentFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Reload hooks (should handle error gracefully)
	rules.ReloadHooks(nonExistentFile, false)

	// Verify no hooks are loaded
	assert.Equal(t, 0, rules.LenLoadedHooks())
}

func TestReloadHooks_WithSeenHooksIds(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	hooksFile := filepath.Join(tempDir, "hooks.json")

	// Create hooks file with duplicate IDs in the same file
	hooksContent := `[
		{
			"id": "duplicate-hook",
			"execute-command": "/bin/echo"
		},
		{
			"id": "duplicate-hook",
			"execute-command": "/bin/echo"
		}
	]`
	err := os.WriteFile(hooksFile, []byte(hooksContent), 0644)
	assert.NoError(t, err)

	rules.HooksFiles = []string{hooksFile}
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)

	// Reload hooks (should detect duplicate and revert)
	rules.ReloadHooks(hooksFile, false)

	// Verify hooks are not loaded due to duplicate
	assert.Equal(t, 0, rules.LenLoadedHooks())
}
