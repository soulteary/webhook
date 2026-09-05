package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
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
