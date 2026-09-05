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
