//go:build linux || windows
// +build linux windows

package platform

import (
	"errors"
	"runtime"
)

// dropPrivileges is a platform-specific implementation for dropping privileges.
// Privilege dropping is not supported on this platform.
func dropPrivileges(uid, gid int) error {
	return errors.New("setuid and setgid not supported on " + runtime.GOOS)
}

// DropPrivileges drops the process privileges to the specified UID and GID.
// This is a backward-compatible wrapper function.
func DropPrivileges(uid, gid int) error {
	return dropPrivileges(uid, gid)
}
