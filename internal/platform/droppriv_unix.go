//go:build !windows && !linux
// +build !windows,!linux

package platform

import (
	"syscall"
)

// dropPrivileges is a platform-specific implementation for dropping privileges.
func dropPrivileges(uid, gid int) error {
	err := syscall.Setgid(gid)
	if err != nil {
		return err
	}

	err = syscall.Setuid(uid)
	if err != nil {
		return err
	}

	return nil
}

// DropPrivileges drops the process privileges to the specified UID and GID.
// This is a backward-compatible wrapper function.
func DropPrivileges(uid, gid int) error {
	return dropPrivileges(uid, gid)
}
