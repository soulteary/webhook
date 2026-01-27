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

	_, _ = os.Create(testFile1)
	_, _ = os.Create(testFile2)
	_, _ = os.Create(testFile3)

	defer func() { _ = os.Remove(testFile1) }()
	defer func() { _ = os.Remove(testFile2) }()
	defer func() { _ = os.Remove(testFile3) }()

	txtFiles := fn.ScanDirByExt(tempDir, ".txt")
	assert.Equal(t, []string{testFile1, testFile3}, txtFiles)

	jpgFiles := fn.ScanDirByExt(tempDir, ".jpg")
	assert.Equal(t, []string{testFile2}, jpgFiles)

	nonExistentPath := filepath.Join(tempDir, "non-existent")
	nonExistentFiles := fn.ScanDirByExt(nonExistentPath, ".txt")
	assert.Nil(t, nonExistentFiles)

	// Test with extension without dot
	txtFiles2 := fn.ScanDirByExt(tempDir, "txt")
	assert.Equal(t, []string{testFile1, testFile3}, txtFiles2)

	// Test with extension with multiple dots
	txtFiles3 := fn.ScanDirByExt(tempDir, "..txt")
	assert.Equal(t, []string{testFile1, testFile3}, txtFiles3)

	// Test with nested directories
	subDir := filepath.Join(tempDir, "subdir")
	_ = os.Mkdir(subDir, 0755)
	testFile4 := filepath.Join(subDir, "test4.txt")
	_, _ = os.Create(testFile4)
	defer func() { _ = os.Remove(testFile4) }()

	txtFiles4 := fn.ScanDirByExt(tempDir, ".txt")
	assert.Contains(t, txtFiles4, testFile4)
}

func TestScanDirByExt_ErrorHandling(t *testing.T) {
	// Test with non-existent directory
	result := fn.ScanDirByExt("/non/existent/path", ".txt")
	assert.Nil(t, result)

	// Test with empty extension
	tempDir := t.TempDir()
	result = fn.ScanDirByExt(tempDir, "")
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestScanDirByExt_CaseInsensitive(t *testing.T) {
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test1.TXT")
	testFile2 := filepath.Join(tempDir, "test2.txt")
	testFile3 := filepath.Join(tempDir, "test3.Txt")

	_, _ = os.Create(testFile1)
	_, _ = os.Create(testFile2)
	_, _ = os.Create(testFile3)

	defer func() { _ = os.Remove(testFile1) }()
	defer func() { _ = os.Remove(testFile2) }()
	defer func() { _ = os.Remove(testFile3) }()

	txtFiles := fn.ScanDirByExt(tempDir, ".txt")
	assert.Len(t, txtFiles, 3)
}
