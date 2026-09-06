//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplementaryGroupsContain(t *testing.T) {
	require.True(t, supplementaryGroupsContain([]int{10, 20, 30}, 20))
	require.False(t, supplementaryGroupsContain([]int{10, 20, 30}, 40))
}

func TestCheckFileChmodAccessRequiresOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "command")
	require.NoError(t, os.WriteFile(path, []byte("command"), 0o600))
	info, err := os.Stat(path)
	require.NoError(t, err)

	require.NoError(t, CheckFileChmodAccess(info, -1))
	require.Error(t, CheckFileChmodAccess(info, os.Geteuid()+1))
}
