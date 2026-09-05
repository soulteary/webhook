package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/platform"
	"github.com/stretchr/testify/require"
)

func TestRunChecksCommandsAndWorkingDirectories(t *testing.T) {
	tempDir := t.TempDir()
	command := filepath.Join(tempDir, "command")
	require.NoError(t, os.WriteFile(command, []byte("#!/bin/sh\n"), 0o700))
	hooksPath := filepath.Join(tempDir, "hooks.yaml")
	require.NoError(t, os.WriteFile(hooksPath, []byte("- id: ok\n  execute-command: command\n  command-working-directory: "+tempDir+"\n"), 0o600))

	appFlags := validFlags(hooksPath)
	checks := Run(appFlags)
	require.False(t, HasFailures(checks), "%+v", checks)
}

func TestRunReportsMissingCommand(t *testing.T) {
	tempDir := t.TempDir()
	hooksPath := filepath.Join(tempDir, "hooks.yaml")
	require.NoError(t, os.WriteFile(hooksPath, []byte("- id: missing\n  execute-command: /definitely/missing/webhook-command\n"), 0o600))

	checks := Run(validFlags(hooksPath))
	require.True(t, HasFailures(checks))
}

func TestRunRejectsCommandOutsideAllowlist(t *testing.T) {
	tempDir := t.TempDir()
	command := filepath.Join(tempDir, "command")
	require.NoError(t, os.WriteFile(command, []byte("#!/bin/sh\n"), 0o700))
	hooksPath := filepath.Join(tempDir, "hooks.yaml")
	require.NoError(t, os.WriteFile(hooksPath, []byte("- id: denied\n  execute-command: "+command+"\n"), 0o600))

	appFlags := validFlags(hooksPath)
	appFlags.AllowedCommandPaths = filepath.Join(tempDir, "allowed")
	checks := Run(appFlags)
	require.True(t, HasFailures(checks))
}

func TestRunRejectsPartialPrivilegeConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	hooksPath := filepath.Join(tempDir, "hooks.yaml")
	require.NoError(t, os.WriteFile(hooksPath, []byte("- id: ok\n  execute-command: /bin/echo\n"), 0o600))

	appFlags := validFlags(hooksPath)
	appFlags.SetUID = 1000
	checks := Run(appFlags)
	require.True(t, HasFailures(checks))
	require.Contains(t, checks[0].Detail, "setuid/setgid")
}

func TestRunRejectsUnsupportedPrivilegeDrop(t *testing.T) {
	if platform.SupportsPrivilegeDrop() {
		t.Skip("privilege dropping is supported on this platform")
	}
	tempDir := t.TempDir()
	hooksPath := filepath.Join(tempDir, "hooks.yaml")
	require.NoError(t, os.WriteFile(hooksPath, []byte("- id: ok\n  execute-command: /bin/echo\n"), 0o600))

	appFlags := validFlags(hooksPath)
	appFlags.SetUID = 1000
	appFlags.SetGID = 1000
	checks := Run(appFlags)
	require.True(t, HasFailures(checks))
	require.Contains(t, checks[0].Detail, "not supported")
}

func TestRunRejectsDuplicateHookFiles(t *testing.T) {
	tempDir := t.TempDir()
	hooksPath := filepath.Join(tempDir, "hooks.yaml")
	require.NoError(t, os.WriteFile(hooksPath, []byte("- id: ok\n  execute-command: /bin/echo\n"), 0o600))

	appFlags := validFlags(hooksPath)
	appFlags.HooksFiles = append(appFlags.HooksFiles, hooksPath)
	checks := Run(appFlags)
	require.True(t, HasFailures(checks))
	require.Contains(t, checks[0].Detail, "duplicate hook file")
}

func TestCheckCommandResolvesRelativeWorkingDirectoryAbsolutely(t *testing.T) {
	tempDir := t.TempDir()
	scriptsDir := filepath.Join(tempDir, "scripts")
	require.NoError(t, os.Mkdir(scriptsDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "run.sh"), []byte("#!/bin/sh\n"), 0o700))

	oldWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(oldWorkingDirectory)) })

	resolved, err := checkCommand("run.sh", "scripts")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(resolved), resolved)
}

func validFlags(hooksPath string) flags.AppFlags {
	return flags.AppFlags{
		Profile:                  "compat",
		Port:                     9000,
		HooksFiles:               hook.HooksFiles{hooksPath},
		HookTimeoutSeconds:       30,
		MaxConcurrentHooks:       10,
		HookExecutionTimeout:     5,
		MaxArgLength:             1024,
		MaxTotalArgsLength:       4096,
		MaxArgsCount:             10,
		MaxMultipartMem:          1024,
		MaxRequestBodySize:       1024,
		MaxHeaderBytes:           1024,
		ReadHeaderTimeoutSeconds: 5,
		ReadTimeoutSeconds:       10,
		WriteTimeoutSeconds:      30,
		IdleTimeoutSeconds:       90,
	}
}
