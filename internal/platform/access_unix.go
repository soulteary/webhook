//go:build !windows

package platform

import (
	"fmt"
	"os"
	"syscall"
)

// CheckFileModeAccess verifies Unix mode bits for a prospective UID/GID.
// required uses the conventional read=4, write=2, execute=1 bit mask.
func CheckFileModeAccess(info os.FileInfo, uid, gid int, required uint32) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot inspect ownership for %s", info.Name())
	}
	permissions := uint32(info.Mode().Perm())
	var allowed uint32
	switch {
	case uint32(uid) == stat.Uid:
		allowed = permissions >> 6
	case uint32(gid) == stat.Gid:
		allowed = permissions >> 3
	default:
		allowed = permissions
	}
	if allowed&required != required {
		return fmt.Errorf("UID %d/GID %d lacks required access to %s", uid, gid, info.Name())
	}
	return nil
}
