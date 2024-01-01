package rules_test

import (
	"testing"

	"github.com/adnanh/webhook/internal/hook"
	"github.com/adnanh/webhook/internal/rules"
	"github.com/stretchr/testify/assert" // You might want to get this assertion library for convenience.
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
