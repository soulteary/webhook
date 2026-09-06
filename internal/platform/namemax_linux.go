//go:build linux

package platform

import "golang.org/x/sys/unix"

// FileNameLimit returns the maximum filename-component length for path's
// filesystem.
func FileNameLimit(path string) (int, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int(stat.Namelen), nil
}
