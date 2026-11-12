package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeRemove_EmptyBaseUsesTempDir(t *testing.T) {
	// 模拟未设置 command-working-directory：os.CreateTemp("", ...) 写入系统临时目录。
	f, err := os.CreateTemp("", "safe-remove-test-")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	name := f.Name()
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(name) })

	if err := safeRemove(name, ""); err != nil {
		t.Fatalf("safeRemove with empty base: %v", err)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatalf("temp file still exists after safeRemove (stat err: %v)", err)
	}
}

func TestSafeRemove_WithinBase(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "payload")
	if err := os.WriteFile(name, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(name, dir); err != nil {
		t.Fatalf("safeRemove: %v", err)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatalf("file still exists (stat err: %v)", err)
	}
}

func TestSafeRemove_RefusesOutsideBase(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(t.TempDir(), "keep")
	if err := os.WriteFile(outside, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(outside, base); err == nil {
		t.Fatal("safeRemove should refuse a path outside the base directory")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("file outside base must be left intact: %v", err)
	}
}
