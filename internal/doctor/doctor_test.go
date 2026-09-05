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

func TestTargetIdentityAccessUsesFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private")
	require.NoError(t, os.WriteFile(path, []byte("private"), 0o000))
	require.Error(t, checkTargetPathAccess(path, 12345, 12345, 4))
}

func TestTargetIdentityRequiresWritablePassFileDirectory(t *testing.T) {
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o755))
	directory := filepath.Join(parent, "read-only")
	require.NoError(t, os.Mkdir(directory, 0o555))
	require.Error(t, checkTargetPathAccess(directory, 12345, 12345, 3))
}

func TestTargetIdentityChecksResolvedSymlinkParents(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o755))
	publicDir := filepath.Join(root, "public")
	privateDir := filepath.Join(root, "private")
	require.NoError(t, os.Mkdir(publicDir, 0o755))
	require.NoError(t, os.Mkdir(privateDir, 0o700))
	target := filepath.Join(privateDir, "hook")
	require.NoError(t, os.WriteFile(target, []byte("hook"), 0o644))
	link := filepath.Join(publicDir, "hook")
	require.NoError(t, os.Symlink(target, link))

	require.Error(t, checkTargetPathAccess(link, 12345, 12345, 4))
}

func TestTargetIdentityChecksWritableFileParent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o555))
	require.Error(t, checkWritableFilePath(filepath.Join(root, "webhook.log"), 12345, 12345))
}

func TestUsesFileAudit(t *testing.T) {
	require.True(t, usesFileAudit(flags.AppFlags{AuditEnabled: true, AuditStorageType: "file"}))
	require.True(t, usesFileAudit(flags.AppFlags{AuditEnabled: true, AuditStorageType: "redis", RedisEnabled: true}))
	require.False(t, usesFileAudit(flags.AppFlags{AuditEnabled: true, AuditStorageType: "redis"}))
	require.False(t, usesFileAudit(flags.AppFlags{AuditStorageType: "file"}))
}

func TestRunRequiresFileBackedAuditPath(t *testing.T) {
	appFlags := validFlags("")
	appFlags.HooksFiles = nil
	appFlags.AuditEnabled = true
	appFlags.AuditStorageType = "file"
	appFlags.AuditFilePath = ""

	checks := Run(appFlags)
	require.True(t, HasFailures(checks))
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

func TestRunCreatesAndChecksHooksDirectory(t *testing.T) {
	hooksDir := filepath.Join(t.TempDir(), "new", "hooks")
	appFlags := validFlags("")
	appFlags.HooksFiles = nil
	appFlags.HooksDir = hooksDir

	checks := Run(appFlags)
	require.False(t, HasFailures(checks), "%+v", checks)
	info, err := os.Stat(hooksDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestRunRejectsUnusableHooksDirectory(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(parentFile, []byte("file"), 0o600))
	appFlags := validFlags("")
	appFlags.HooksFiles = nil
	appFlags.HooksDir = filepath.Join(parentFile, "hooks")

	checks := Run(appFlags)
	require.True(t, HasFailures(checks))
	require.Contains(t, checks[len(checks)-1].Subject, "hooks directory")
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
