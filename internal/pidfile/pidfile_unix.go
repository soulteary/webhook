//go:build !windows && !darwin
// +build !windows,!darwin

package pidfile

import (
	"errors"

	"golang.org/x/sys/unix"
)

func processExists(pid int) bool {
	err := unix.Kill(pid, 0)
	return err == nil || errors.Is(err, unix.EPERM)
}
