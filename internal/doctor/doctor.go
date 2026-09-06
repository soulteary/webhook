package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/pidfile"
	"github.com/soulteary/webhook/internal/platform"
	"github.com/soulteary/webhook/internal/security"
)

// Check is one diagnostic produced by Run.
type Check struct {
	OK      bool
	Subject string
	Detail  string
}

// Run validates the effective configuration and verifies every configured
// command and working directory without starting the HTTP server.
func Run(appFlags flags.AppFlags) []Check {
	checks := make([]Check, 0)
	validation := flags.Validate(appFlags)
	if validation.HasErrors() {
		for _, err := range validation.Errors {
			checks = append(checks, Check{Subject: "configuration", Detail: err.Error()})
		}
		return checks
	}
	checks = append(checks, Check{OK: true, Subject: "configuration", Detail: "valid"})
	if appFlags.SetUID != 0 && appFlags.SetGID != 0 && platform.SupportsPrivilegeDrop() {
		uid, gid, ok := platform.EffectiveIdentity()
		if !ok {
			checks = append(checks, Check{Subject: "privilege drop", Detail: "cannot determine current process identity"})
		} else if err := checkPrivilegeDropIdentity(uid, gid, appFlags.SetUID, appFlags.SetGID); err != nil {
			checks = append(checks, Check{Subject: "privilege drop", Detail: err.Error()})
		} else {
			checks = append(checks, Check{OK: true, Subject: "privilege drop", Detail: fmt.Sprintf("UID %d/GID %d", appFlags.SetUID, appFlags.SetGID)})
		}
	}
	accessUID, accessGID := appFlags.SetUID, appFlags.SetGID
	if accessUID == 0 && accessGID == 0 {
		if uid, gid, ok := platform.EffectiveIdentity(); ok {
			accessUID, accessGID = uid, gid
		}
	}
	destinations := []struct {
		subject string
		path    string
		pid     bool
	}{
		{subject: "log path", path: appFlags.LogPath},
		{subject: "PID path", path: appFlags.PidPath, pid: true},
	}
	if usesFileAudit(appFlags) {
		if strings.TrimSpace(appFlags.AuditFilePath) == "" {
			checks = append(checks, Check{Subject: "audit file path", Detail: "is required for file-backed audit storage"})
		} else {
			destinations = append(destinations, struct {
				subject string
				path    string
				pid     bool
			}{subject: "audit file path", path: appFlags.AuditFilePath})
		}
	}
	for _, destination := range destinations {
		subject, path := destination.subject, destination.path
		if path == "" {
			continue
		}
		if err := checkWritableFilePath(path, accessUID, accessGID); err != nil {
			checks = append(checks, Check{Subject: subject, Detail: err.Error()})
		} else if destination.pid {
			if err := pidfile.CheckExisting(path); err != nil {
				checks = append(checks, Check{Subject: subject, Detail: err.Error()})
			} else {
				checks = append(checks, Check{OK: true, Subject: subject, Detail: filepath.Clean(path)})
			}
		} else {
			checks = append(checks, Check{OK: true, Subject: subject, Detail: filepath.Clean(path)})
		}
	}
	if appFlags.HooksDir != "" {
		useCurrentIdentity := appFlags.SetUID == 0 && appFlags.SetGID == 0
		if err := checkHooksDirectory(appFlags.HooksDir, accessUID, accessGID, useCurrentIdentity); err != nil {
			checks = append(checks, Check{Subject: "hooks directory", Detail: err.Error()})
			return checks
		} else {
			checks = append(checks, Check{OK: true, Subject: "hooks directory", Detail: filepath.Clean(appFlags.HooksDir)})
		}
	}
	commandValidator := security.NewCommandValidator()
	for _, allowedPath := range strings.Split(appFlags.AllowedCommandPaths, ",") {
		if allowedPath = strings.TrimSpace(allowedPath); allowedPath != "" {
			commandValidator.AllowedPaths = append(commandValidator.AllowedPaths, allowedPath)
		}
	}

	if len(appFlags.HooksFiles) == 0 {
		return append(checks, Check{OK: true, Subject: "hooks", Detail: "no hook files discovered"})
	}

	for _, path := range appFlags.HooksFiles {
		if err := checkTargetPathAccess(path, accessUID, accessGID, 4); err != nil {
			checks = append(checks, Check{Subject: path, Detail: err.Error()})
			continue
		}
		var hooks hook.Hooks
		var err error
		if appFlags.ValidateStrict {
			err = hooks.LoadFromFileStrict(path, appFlags.AsTemplate)
		} else {
			err = hooks.LoadFromFile(path, appFlags.AsTemplate)
		}
		if err != nil {
			checks = append(checks, Check{Subject: path, Detail: err.Error()})
			continue
		}
		checks = append(checks, Check{OK: true, Subject: path, Detail: fmt.Sprintf("%d hook(s) loaded", len(hooks))})

		for _, configuredHook := range hooks {
			subject := fmt.Sprintf("hook %q", configuredHook.ID)
			resolvedCommand, err := checkCommand(configuredHook.ExecuteCommand, configuredHook.CommandWorkingDirectory)
			if err == nil {
				err = commandValidator.ValidateCommandPath(resolvedCommand)
			}
			if err == nil {
				err = checkTargetPathAccess(resolvedCommand, accessUID, accessGID, 1)
			}
			if err != nil {
				checks = append(checks, Check{Subject: subject + " command", Detail: err.Error()})
			} else {
				checks = append(checks, Check{OK: true, Subject: subject + " command", Detail: resolvedCommand})
			}
			directoryToCheck := configuredHook.CommandWorkingDirectory
			if directoryToCheck == "" && len(configuredHook.PassFileToCommand) != 0 {
				directoryToCheck = os.TempDir()
			}
			if directoryToCheck != "" {
				err := checkWorkingDirectory(directoryToCheck)
				if err == nil {
					required := uint32(1)
					if len(configuredHook.PassFileToCommand) != 0 {
						required |= 2
					}
					err = checkTargetPathAccess(directoryToCheck, accessUID, accessGID, required)
				}
				if err != nil {
					checks = append(checks, Check{Subject: subject + " working directory", Detail: err.Error()})
				} else {
					checks = append(checks, Check{OK: true, Subject: subject + " working directory", Detail: directoryToCheck})
				}
			}
		}
	}
	return checks
}

func checkPrivilegeDropIdentity(currentUID, currentGID, targetUID, targetGID int) error {
	if currentUID == 0 {
		return nil
	}
	if targetUID != currentUID || targetGID != currentGID {
		return fmt.Errorf("current UID %d/GID %d cannot switch to UID %d/GID %d without root privileges", currentUID, currentGID, targetUID, targetGID)
	}
	return nil
}

func usesFileAudit(appFlags flags.AppFlags) bool {
	if !appFlags.AuditEnabled {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(appFlags.AuditStorageType)) {
	case "file":
		return true
	case "redis":
		return appFlags.RedisEnabled
	default:
		return false
	}
}

// HasFailures reports whether any diagnostic failed.
func HasFailures(checks []Check) bool {
	for _, check := range checks {
		if !check.OK {
			return true
		}
	}
	return false
}

func checkCommand(command, workingDirectory string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("execute-command is empty")
	}
	candidate, err := security.ResolveCommandCandidate(command, workingDirectory)
	if err != nil {
		return "", err
	}
	resolved, err := exec.LookPath(candidate)
	if err != nil && filepath.IsAbs(command) {
		base := filepath.Base(command)
		if base == "true" || base == "false" {
			resolved, err = exec.LookPath(base)
		}
	}
	if err != nil {
		return "", fmt.Errorf("not found: %s", candidate)
	}
	command = resolved
	info, err := os.Stat(command)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", command)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("not executable: %s", command)
	}
	return command, nil
}

func checkWorkingDirectory(path string) error {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}

func checkHooksDirectory(path string, uid, gid int, useCurrentIdentity bool) error {
	path = filepath.Clean(path)
	if useCurrentIdentity {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return fmt.Errorf("cannot create %s: %w", path, err)
		}
	} else {
		info, err := os.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("not a directory: %s", path)
			}
			if err := checkTargetPathAccess(path, uid, gid, 5); err != nil {
				return err
			}
		} else if os.IsNotExist(err) {
			parent, parentErr := nearestExistingParent(path)
			if parentErr != nil {
				return parentErr
			}
			if err := checkTargetPathAccess(parent, uid, gid, 3); err != nil {
				return fmt.Errorf("cannot create %s: %w", path, err)
			}
			return nil
		} else {
			return err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("cannot create file watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()
	if err := watcher.Add(path); err != nil {
		return fmt.Errorf("cannot watch %s: %w", path, err)
	}
	return nil
}

func nearestExistingParent(path string) (string, error) {
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("no existing parent directory for %s", path)
		}
		info, err := os.Stat(parent)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("not a directory: %s", parent)
			}
			return parent, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		path = parent
	}
}

func checkTargetPathAccess(path string, uid, gid int, required uint32) error {
	if uid == 0 && gid == 0 {
		// UID/GID 0 means root on Unix, but EffectiveIdentity deliberately
		// reports ok=false on platforms without POSIX identities (Windows).
		// In that case, verify access with the current process token instead of
		// treating the zero values as an unconditional success.
		if _, _, ok := platform.EffectiveIdentity(); ok {
			return nil
		}
	}
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	if err := checkPathAndParentsAccess(path, uid, gid, required); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if resolved != path {
		if err := checkPathAndParentsAccess(resolved, uid, gid, required); err != nil {
			return err
		}
	}
	return nil
}

func checkPathAndParentsAccess(path string, uid, gid int, required uint32) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if err := checkPathAccess(path, info, uid, gid, required); err != nil {
		return fmt.Errorf("target identity cannot access %s: %w", path, err)
	}
	for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
		info, err := os.Stat(parent)
		if err != nil {
			return err
		}
		if err := checkPathAccess(parent, info, uid, gid, 1); err != nil {
			return fmt.Errorf("target identity cannot traverse %s: %w", parent, err)
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
	}
	return nil
}

func checkPathAccess(path string, info os.FileInfo, uid, gid int, required uint32) error {
	if uid == 0 && gid == 0 {
		return platform.CheckCurrentPathAccess(path, required)
	}
	return platform.CheckFileModeAccess(info, uid, gid, required)
}

func checkWritableFilePath(path string, uid, gid int) error {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("not a file: %s", path)
		}
		return checkTargetPathAccess(path, uid, gid, 2)
	}
	if !os.IsNotExist(err) {
		return err
	}
	linkInfo, linkErr := os.Lstat(path)
	if linkErr == nil && linkInfo.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := resolveSymlinkChain(path, uid, gid)
		if err != nil {
			return err
		}
		targetInfo, targetErr := os.Stat(linkTarget)
		if targetErr == nil {
			if targetInfo.IsDir() {
				return fmt.Errorf("symlink target is not a file: %s", linkTarget)
			}
			return checkTargetPathAccess(linkTarget, uid, gid, 2)
		}
		if !os.IsNotExist(targetErr) {
			return targetErr
		}
		targetParent := filepath.Dir(linkTarget)
		targetParentInfo, err := os.Stat(targetParent)
		if err != nil {
			return fmt.Errorf("cannot access symlink target parent %s: %w", targetParent, err)
		}
		if !targetParentInfo.IsDir() {
			return fmt.Errorf("symlink target parent is not a directory: %s", targetParent)
		}
		return checkTargetPathAccess(targetParent, uid, gid, 3)
	}
	if linkErr != nil && !os.IsNotExist(linkErr) {
		return linkErr
	}
	parent := filepath.Dir(path)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("cannot access immediate parent directory %s: %w", parent, err)
	}
	if !parentInfo.IsDir() {
		return fmt.Errorf("immediate parent is not a directory: %s", parent)
	}
	return checkTargetPathAccess(parent, uid, gid, 3)
}

func resolveSymlinkChain(path string, uid, gid int) (string, error) {
	seen := make(map[string]bool)
	for {
		absolutePath, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			return "", err
		}
		path = absolutePath
		if seen[path] {
			return "", fmt.Errorf("symlink loop detected at %s", path)
		}
		seen[path] = true

		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return path, nil
		}
		if err := checkTargetPathAccess(filepath.Dir(path), uid, gid, 1); err != nil {
			return "", err
		}

		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
}
