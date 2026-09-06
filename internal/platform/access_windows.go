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

// CheckFileChmodAccess is not identity-based on Windows. A later os.Chmod call
// remains subject to the current process token and filesystem ACLs.
func CheckFileChmodAccess(info os.FileInfo, uid int) error {
	return nil
}

// EffectiveIdentity reports that POSIX UID/GID checks do not apply on Windows.
func EffectiveIdentity() (uid, gid int, ok bool) {
	return 0, 0, false
}
