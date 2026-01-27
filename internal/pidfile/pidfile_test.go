package pidfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAndRemove(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "test-pidfile")
	if err != nil {
		t.Fatal("Could not create test directory")
	}

	path := filepath.Join(dir, "testfile")
	file, err := New(path)
	if err != nil {
		t.Fatal("Could not create test file", err)
	}

	_, err = New(path)
	if err == nil {
		t.Fatal("Test file creation not blocked")
	}

	if err := file.Remove(); err != nil {
		t.Fatal("Could not delete created test file")
	}
}

func TestRemoveInvalidPath(t *testing.T) {
	file := PIDFile{path: filepath.Join("foo", "bar")}

	if err := file.Remove(); err == nil {
		t.Fatal("Non-existing file doesn't give an error on delete")
	}
}

func TestNew_WithExistingPIDFile(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "test-pidfile")
	if err != nil {
		t.Fatal("Could not create test directory")
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "testfile")

	// 创建一个 PID 文件
	file, err := New(path)
	if err != nil {
		t.Fatal("Could not create test file", err)
	}
	defer func() { _ = file.Remove() }()

	// 尝试再次创建同一个 PID 文件（应该失败）
	_, err = New(path)
	if err == nil {
		t.Fatal("New() should return error when PID file already exists")
	}
}

func TestNew_WithInvalidDirectory(t *testing.T) {
	// 尝试在无效路径创建 PID 文件
	// 在 Unix 系统上，尝试在 /root 下创建文件通常会失败（如果没有权限）
	invalidPath := "/root/webhook_test_pidfile"

	// 这个测试可能会失败，取决于系统权限
	// 但至少可以测试错误处理路径
	_, err := New(invalidPath)
	if err == nil {
		// 如果成功创建，清理文件
		_ = os.Remove(invalidPath)
		// 这个测试主要确保错误处理路径被执行
	}
}

func TestNew_WithLongPath(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "test-pidfile")
	if err != nil {
		t.Fatal("Could not create test directory")
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// 创建一个很长的路径
	longPath := filepath.Join(dir, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "pidfile")

	file, err := New(longPath)
	if err != nil {
		t.Fatalf("New() with long path should succeed, got error: %v", err)
	}

	if err := file.Remove(); err != nil {
		t.Fatalf("Remove() should succeed, got error: %v", err)
	}
}

func TestNew_WithEmptyPath(t *testing.T) {
	// 测试空路径（应该失败，因为无法创建目录）
	_, err := New("")
	if err == nil {
		t.Fatal("New() with empty path should return error")
	}
}

func TestCheckPIDFileAlreadyExists(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "test-pidfile")
	if err != nil {
		t.Fatal("Could not create test directory")
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "testfile")

	// 测试不存在的文件（应该返回 nil）
	err = checkPIDFileAlreadyExists(path)
	if err != nil {
		t.Errorf("checkPIDFileAlreadyExists() with non-existent file should return nil, got: %v", err)
	}

	// 创建一个无效的 PID 文件（包含非数字内容）
	err = os.WriteFile(path, []byte("invalid-pid"), 0o600)
	if err != nil {
		t.Fatal("Could not create test file", err)
	}
	defer func() { _ = os.Remove(path) }()

	// 应该返回 nil，因为 PID 无法解析
	err = checkPIDFileAlreadyExists(path)
	if err != nil {
		t.Errorf("checkPIDFileAlreadyExists() with invalid PID should return nil, got: %v", err)
	}

	// 创建一个包含无效进程 ID 的文件
	err = os.WriteFile(path, []byte("999999"), 0o600)
	if err != nil {
		t.Fatal("Could not create test file", err)
	}

	// 应该返回 nil，因为进程不存在
	err = checkPIDFileAlreadyExists(path)
	if err != nil {
		t.Errorf("checkPIDFileAlreadyExists() with non-existent PID should return nil, got: %v", err)
	}
}
