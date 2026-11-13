package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeScript(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// A directory entry must not allow a sibling directory that merely shares its
// name as a prefix (/opt/hooks vs /opt/hooks-evil).
func TestValidateCommandPath_SiblingPrefixRejected(t *testing.T) {
	root := t.TempDir()
	allowedDir := filepath.Join(root, "hooks")
	inside := filepath.Join(allowedDir, "ok.sh")
	sibling := filepath.Join(root, "hooks-evil", "run.sh")
	writeScript(t, inside)
	writeScript(t, sibling)

	cv := NewCommandValidator()
	cv.AllowedPaths = []string{allowedDir}

	if err := cv.ValidateCommandPath(inside); err != nil {
		t.Fatalf("script inside the allowed directory was rejected: %v", err)
	}
	if err := cv.ValidateCommandPath(sibling); err == nil {
		t.Fatal("script in a sibling directory sharing the prefix was allowed")
	}
	if err := cv.ValidateCommandPath(filepath.Join(allowedDir, "..", "hooks-evil", "run.sh")); err == nil {
		t.Fatal("path escaping the allowed directory via .. was allowed")
	}
}

// A symlink inside the allowed directory must be judged by its target.
func TestValidateCommandPath_SymlinkTargetChecked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	root := t.TempDir()
	allowedDir := filepath.Join(root, "hooks")
	outside := filepath.Join(root, "outside", "evil.sh")
	writeScript(t, filepath.Join(allowedDir, "ok.sh"))
	writeScript(t, outside)
	link := filepath.Join(allowedDir, "link.sh")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	cv := NewCommandValidator()
	cv.AllowedPaths = []string{allowedDir}

	if err := cv.ValidateCommandPath(link); err == nil {
		t.Fatal("symlink pointing outside the allowed directory was allowed")
	}
}
