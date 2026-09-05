package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
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
			if err != nil {
				checks = append(checks, Check{Subject: subject + " command", Detail: err.Error()})
			} else {
				checks = append(checks, Check{OK: true, Subject: subject + " command", Detail: resolvedCommand})
			}
			if configuredHook.CommandWorkingDirectory != "" {
				if err := checkWorkingDirectory(configuredHook.CommandWorkingDirectory); err != nil {
					checks = append(checks, Check{Subject: subject + " working directory", Detail: err.Error()})
				} else {
					checks = append(checks, Check{OK: true, Subject: subject + " working directory", Detail: configuredHook.CommandWorkingDirectory})
				}
			}
		}
	}
	return checks
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
