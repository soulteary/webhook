//go:build !windows

package platform

import "golang.org/x/sys/unix"

// CheckCurrentPathAccess verifies access with the current process identity.
func CheckCurrentPathAccess(path string, required uint32) error {
	mode := uint32(0)
	if required&4 != 0 {
		mode |= unix.R_OK
	}
	if required&2 != 0 {
		mode |= unix.W_OK
	}
	if required&1 != 0 {
		mode |= unix.X_OK
	}
	return unix.Access(path, mode)
}
