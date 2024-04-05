package fn_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/stretchr/testify/assert"
)

func TestScanDirByExt(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	files := []string{
		"file1.txt",
		"file2.txt",
		"file3.go",
		"file4.go",
		"file5.py",
		"file6.TXT",
	}
	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		_, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("create test file failed: %v", err)
		}
	}

	testCases := []struct {
		name     string
		filePath string
		fileExt  string
		expected []string
	}{
		{
			name:     "scan .txt files",
			filePath: tempDir,
			fileExt:  "txt",
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "file2.txt"),
				filepath.Join(tempDir, "file6.TXT"),
			},
		},
		{
			name:     "scan .go files",
			filePath: tempDir,
			fileExt:  "go",
			expected: []string{
				filepath.Join(tempDir, "file3.go"),
				filepath.Join(tempDir, "file4.go"),
			},
		},
		{
			name:     "scan .py files",
			filePath: tempDir,
			fileExt:  "py",
			expected: []string{
				filepath.Join(tempDir, "file5.py"),
			},
		},
		{
			name:     "scan unknown files",
			filePath: tempDir,
			fileExt:  "doc",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fn.ScanDirByExt(tc.filePath, tc.fileExt)
			assert.Equal(t, tc.expected, result)
		})
	}
}
