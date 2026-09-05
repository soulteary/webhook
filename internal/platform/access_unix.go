//go:build !windows

package platform

import (
	"fmt"
	"os"
	"strconv"
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
	uidString := strconv.Itoa(uid)
	gidString := strconv.Itoa(gid)
	groups, groupsErr := os.Getgroups()
	var allowed uint32
	switch {
	case uidString == strconv.FormatUint(uint64(stat.Uid), 10):
		allowed = permissions >> 6
	case gidString == strconv.FormatUint(uint64(stat.Gid), 10) ||
		groupsErr == nil && supplementaryGroupsContain(groups, stat.Gid):
		allowed = permissions >> 3
	default:
		allowed = permissions
	}
	if allowed&required != required {
		return fmt.Errorf("UID %d/GID %d lacks required access to %s", uid, gid, info.Name())
	}
	return nil
}

func supplementaryGroupsContain(groups []int, fileGID uint32) bool {
	fileGIDString := strconv.FormatUint(uint64(fileGID), 10)
	for _, group := range groups {
		if strconv.Itoa(group) == fileGIDString {
			return true
		}
	}
	return false
}

// EffectiveIdentity returns the process identity used for normal filesystem
// access when no post-drop UID/GID is configured.
func EffectiveIdentity() (uid, gid int, ok bool) {
	return os.Geteuid(), os.Getegid(), true
}
