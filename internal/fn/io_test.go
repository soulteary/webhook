package fn_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/stretchr/testify/assert"
)

func TestScanDirByExt(t *testing.T) {
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "test2.jpg")
	testFile3 := filepath.Join(tempDir, "test3.txt")

	os.Create(testFile1)
	os.Create(testFile2)
	os.Create(testFile3)

	defer os.Remove(testFile1)
	defer os.Remove(testFile2)
	defer os.Remove(testFile3)

	txtFiles := fn.ScanDirByExt(tempDir, ".txt")
	assert.Equal(t, []string{testFile1, testFile3}, txtFiles)

	jpgFiles := fn.ScanDirByExt(tempDir, ".jpg")
	assert.Equal(t, []string{testFile2}, jpgFiles)

	nonExistentPath := filepath.Join(tempDir, "non-existent")
	nonExistentFiles := fn.ScanDirByExt(nonExistentPath, ".txt")
	assert.Nil(t, nonExistentFiles)
}
