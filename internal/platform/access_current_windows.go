//go:build windows

package platform

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// CheckCurrentPathAccess verifies access using the current Windows process
// token. FILE_FLAG_BACKUP_SEMANTICS permits the same check for directories.
func CheckCurrentPathAccess(path string, required uint32) error {
	desiredAccess := uint32(0)
	if required&4 != 0 {
		desiredAccess |= windows.GENERIC_READ
	}
	if required&2 != 0 {
		desiredAccess |= windows.GENERIC_WRITE
	}
	if required&1 != 0 {
		desiredAccess |= windows.GENERIC_EXECUTE
	}

	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		pathPointer,
		desiredAccess,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return fmt.Errorf("current process lacks required access: %w", err)
	}
	return windows.CloseHandle(handle)
}
