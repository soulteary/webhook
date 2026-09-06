//go:build !linux

package platform

import (
	"fmt"
	"os"
)

// FileNameLimit uses the common portable component limit on non-Linux builds.
// The existence check ensures validation still targets the runtime directory.
func FileNameLimit(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", path)
	}
	return 255, nil
}
