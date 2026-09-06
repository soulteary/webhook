//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckCurrentPathAccessRejectsReadOnlyFileForWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "read-only.log")
	require.NoError(t, os.WriteFile(path, []byte("log"), 0o600))
	require.NoError(t, os.Chmod(path, 0o400))
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	require.NoError(t, CheckCurrentPathAccess(path, 4))
	require.Error(t, CheckCurrentPathAccess(path, 2))
}
