package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSubcommands(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "legacy", in: []string{"-validate-config", "-hooks", "hooks.yaml"}, want: []string{"-validate-config", "-hooks", "hooks.yaml"}},
		{name: "validate", in: []string{"validate", "--strict", "-hooks", "hooks.yaml"}, want: []string{"-validate-config", "-validate-strict", "-hooks", "hooks.yaml"}},
		{name: "doctor", in: []string{"doctor", "--strict"}, want: []string{"-doctor", "-validate-strict"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRunInitCreatesStrictlyValidConfig(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", "test-secret")
	output := filepath.Join(t.TempDir(), "hooks", "hooks.yaml")
	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, RunInit([]string{"--output", output}, &stdout, &stderr), stderr.String())

	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	var hooks hook.Hooks
	require.NoError(t, hooks.LoadFromFileStrict(output, true))
	require.Len(t, hooks, 1)
	require.Equal(t, "hello", hooks[0].ID)

	require.Equal(t, 1, RunInit([]string{"--output", output}, &stdout, &stderr))
}

func TestRunInitJSON(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", "test-secret")
	output := filepath.Join(t.TempDir(), "hooks.json")
	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, RunInit([]string{"--format", "json", "--output", output}, &stdout, &stderr), stderr.String())
	var hooks hook.Hooks
	require.NoError(t, hooks.LoadFromFileStrict(output, true))
}
