package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCommandValidator(t *testing.T) {
	cv := NewCommandValidator()
	if cv == nil {
		t.Fatal("NewCommandValidator() should not return nil")
	}

	if cv.MaxArgLength != DefaultMaxArgLength {
		t.Errorf("MaxArgLength = %d, want %d", cv.MaxArgLength, DefaultMaxArgLength)
	}

	if cv.MaxTotalArgsLength != DefaultMaxTotalArgsLength {
		t.Errorf("MaxTotalArgsLength = %d, want %d", cv.MaxTotalArgsLength, DefaultMaxTotalArgsLength)
	}

	if cv.MaxArgsCount != DefaultMaxArgsCount {
		t.Errorf("MaxArgsCount = %d, want %d", cv.MaxArgsCount, DefaultMaxArgsCount)
	}

	if cv.StrictMode {
		t.Error("StrictMode should be false by default")
	}

	if len(cv.SensitivePatterns) == 0 {
		t.Error("SensitivePatterns should not be empty")
	}
}

func TestValidateCommandPath(t *testing.T) {
	cv := NewCommandValidator()

	t.Run("no whitelist", func(t *testing.T) {
		err := cv.ValidateCommandPath("/usr/bin/ls")
		if err != nil {
			t.Errorf("ValidateCommandPath() with no whitelist should allow all paths, got error: %v", err)
		}
	})

	t.Run("with whitelist - directory", func(t *testing.T) {
		cv.AllowedPaths = []string{"/usr/bin"}
		err := cv.ValidateCommandPath("/usr/bin/ls")
		if err != nil {
			t.Errorf("ValidateCommandPath() should allow path in whitelist directory, got error: %v", err)
		}
	})

	t.Run("with whitelist - exact file", func(t *testing.T) {
		cv.AllowedPaths = []string{"/usr/bin/ls"}
		err := cv.ValidateCommandPath("/usr/bin/ls")
		if err != nil {
			t.Errorf("ValidateCommandPath() should allow exact file path, got error: %v", err)
		}
	})

	t.Run("with whitelist - not allowed", func(t *testing.T) {
		cv.AllowedPaths = []string{"/usr/bin"}
		err := cv.ValidateCommandPath("/bin/sh")
		if err == nil {
			t.Error("ValidateCommandPath() should reject path not in whitelist")
		}
	})

	t.Run("relative path", func(t *testing.T) {
		cv.AllowedPaths = []string{"/usr/bin"}
		err := cv.ValidateCommandPath("ls")
		if err == nil {
			t.Error("ValidateCommandPath() should reject relative path when whitelist is set")
		}
	})
}

func TestValidateArgs(t *testing.T) {
	cv := NewCommandValidator()

	t.Run("valid args", func(t *testing.T) {
		args := []string{"arg1", "arg2", "arg3"}
		err := cv.ValidateArgs(args)
		if err != nil {
			t.Errorf("ValidateArgs() with valid args should not return error, got: %v", err)
		}
	})

	t.Run("too many args", func(t *testing.T) {
		cv.MaxArgsCount = 2
		args := make([]string, 3)
		for i := range args {
			args[i] = "arg"
		}
		err := cv.ValidateArgs(args)
		if err == nil {
			t.Error("ValidateArgs() should return error when args count exceeds limit")
		}
	})

	t.Run("arg too long", func(t *testing.T) {
		cv.MaxArgLength = 10
		args := []string{"this argument is too long"}
		err := cv.ValidateArgs(args)
		if err == nil {
			t.Error("ValidateArgs() should return error when arg length exceeds limit")
		}
	})

	t.Run("total args too long", func(t *testing.T) {
		cv.MaxTotalArgsLength = 10
		args := []string{"arg1", "arg2", "arg3"}
		err := cv.ValidateArgs(args)
		if err == nil {
			t.Error("ValidateArgs() should return error when total args length exceeds limit")
		}
	})

	t.Run("strict mode - dangerous characters", func(t *testing.T) {
		cv.StrictMode = true
		args := []string{"arg1; rm -rf /"}
		err := cv.ValidateArgs(args)
		if err == nil {
			t.Error("ValidateArgs() in strict mode should reject args with dangerous characters")
		}
	})

	t.Run("strict mode - safe args", func(t *testing.T) {
		cv.StrictMode = true
		args := []string{"arg1", "arg2"}
		err := cv.ValidateArgs(args)
		if err != nil {
			t.Errorf("ValidateArgs() in strict mode should allow safe args, got error: %v", err)
		}
	})
}

func TestSanitizeForLog(t *testing.T) {
	cv := NewCommandValidator()

	t.Run("normal command and args", func(t *testing.T) {
		cmd, args := cv.SanitizeForLog("/usr/bin/ls", []string{"-l", "/tmp"})
		if cmd != "/usr/bin/ls" {
			t.Errorf("SanitizeForLog() cmd = %s, want /usr/bin/ls", cmd)
		}
		if len(args) != 2 {
			t.Errorf("SanitizeForLog() args length = %d, want 2", len(args))
		}
	})

	t.Run("long command path", func(t *testing.T) {
		longPath := string(make([]byte, 600))
		cmd, _ := cv.SanitizeForLog(longPath, []string{})
		if len(cmd) <= 500 {
			t.Error("SanitizeForLog() should truncate long command paths")
		}
	})

	t.Run("sensitive information in args", func(t *testing.T) {
		args := []string{"password=secret123", "token=abc123"}
		_, sanitizedArgs := cv.SanitizeForLog("/usr/bin/command", args)
		for _, arg := range sanitizedArgs {
			if arg != "password=***" && arg != "token=***" {
				// 检查是否包含脱敏后的值
				if !contains(arg, "***") {
					t.Errorf("SanitizeForLog() should sanitize sensitive args, got: %s", arg)
				}
			}
		}
	})

	t.Run("long arg", func(t *testing.T) {
		longArg := string(make([]byte, 300))
		_, args := cv.SanitizeForLog("/usr/bin/command", []string{longArg})
		if len(args[0]) <= 200 {
			t.Error("SanitizeForLog() should truncate long args")
		}
	})
}

func TestLogCommandExecution(t *testing.T) {
	cv := NewCommandValidator()

	// 这个测试主要确保函数不会 panic
	cv.LogCommandExecution("req-123", "hook-456", "/usr/bin/ls", []string{"-l"}, []string{"PATH=/usr/bin"})
}

func TestValidateCommand(t *testing.T) {
	cv := NewCommandValidator()

	t.Run("valid command", func(t *testing.T) {
		err := cv.ValidateCommand("/usr/bin/ls", []string{"-l"})
		if err != nil {
			t.Errorf("ValidateCommand() with valid command should not return error, got: %v", err)
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		cv.AllowedPaths = []string{"/usr/bin"}
		err := cv.ValidateCommand("/bin/sh", []string{"-c", "echo test"})
		if err == nil {
			t.Error("ValidateCommand() should return error when path is not in whitelist")
		}
	})

	t.Run("invalid args", func(t *testing.T) {
		cv.MaxArgsCount = 1
		err := cv.ValidateCommand("/usr/bin/ls", []string{"-l", "-a"})
		if err == nil {
			t.Error("ValidateCommand() should return error when args are invalid")
		}
	})
}

func TestCommandValidationError(t *testing.T) {
	err := NewCommandValidationError("path", "test error", "/usr/bin/ls", []string{"-l"})
	if err == nil {
		t.Fatal("NewCommandValidationError() should not return nil")
	}

	if err.Error() == "" {
		t.Error("CommandValidationError.Error() should return non-empty string")
	}

	if !IsCommandValidationError(err) {
		t.Error("IsCommandValidationError() should return true for CommandValidationError")
	}

	// 测试非 CommandValidationError
	regularErr := &testError{msg: "regular error"}
	if IsCommandValidationError(regularErr) {
		t.Error("IsCommandValidationError() should return false for non-CommandValidationError")
	}
}

func TestIsDirectory(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "webhook_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建临时文件
	tmpFile := filepath.Join(tmpDir, "test_file")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing directory", tmpDir, true},
		{"existing file", tmpFile, false},
		{"non-existent path with separator", tmpDir + string(filepath.Separator) + "nonexistent" + string(filepath.Separator), true},
		{"non-existent path without separator", filepath.Join(tmpDir, "nonexistent"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDirectory(tt.path)
			if result != tt.expected {
				t.Errorf("isDirectory(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestSanitizeArg(t *testing.T) {
	cv := NewCommandValidator()

	t.Run("normal arg", func(t *testing.T) {
		result := cv.sanitizeArg("normal-arg")
		if result != "normal-arg" {
			t.Errorf("sanitizeArg() = %s, want normal-arg", result)
		}
	})

	t.Run("long arg", func(t *testing.T) {
		longArg := string(make([]byte, 300))
		result := cv.sanitizeArg(longArg)
		if len(result) <= 200 {
			t.Error("sanitizeArg() should truncate long args")
		}
		if !contains(result, "[truncated]") {
			t.Error("sanitizeArg() should append [truncated] for long args")
		}
	})

	t.Run("sensitive arg", func(t *testing.T) {
		args := []string{"password=secret123"}
		result := cv.sanitizeArg(args[0])
		if !contains(result, "***") {
			t.Errorf("sanitizeArg() should sanitize sensitive args, got: %s", result)
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
