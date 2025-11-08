//go:build linux || windows
// +build linux windows

package platform

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDropPrivileges_NotSupported(t *testing.T) {
	// Test that DropPrivileges returns an error on unsupported platforms
	err := DropPrivileges(1000, 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), runtime.GOOS)
}
