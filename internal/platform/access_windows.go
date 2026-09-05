//go:build windows

package platform

import (
	"fmt"
	"os"
)

// CheckFileModeAccess is unavailable on Windows, where setuid/setgid is also
// unsupported by this application.
func CheckFileModeAccess(info os.FileInfo, uid, gid int, required uint32) error {
	return fmt.Errorf("cannot evaluate UID/GID access on Windows")
}

// EffectiveIdentity reports that POSIX UID/GID checks do not apply on Windows.
func EffectiveIdentity() (uid, gid int, ok bool) {
	return 0, 0, false
}
