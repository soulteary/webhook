package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildHookecho 编译 hookecho 程序并返回可执行文件路径
func buildHookecho(t *testing.T) (binPath string, cleanupFn func()) {
	tmp, err := os.MkdirTemp("", "hookecho-test-")
	if err != nil {
		t.Fatal(err)
	}

	binPath = filepath.Join(tmp, "hookecho")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		_ = os.RemoveAll(tmp)
		t.Fatal(err)
	}

	// 编译 hookecho.go
	cmd := exec.Command("go", "build", "-o", binPath, filepath.Join(wd, "hookecho.go"))
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmp)
		t.Fatalf("Building hookecho: %v", err)
	}

	return binPath, func() { _ = os.RemoveAll(tmp) }
}

func TestHookecho_NoArgs(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	cmd := exec.Command(binPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hookecho with no args should not fail: %v", err)
	}

	outputStr := string(output)
	if outputStr != "" {
		t.Errorf("hookecho with no args should produce no output, got: %q", outputStr)
	}
}

func TestHookecho_WithArgs(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "single arg",
			args: []string{"test"},
			want: "arg: test\n",
		},
		{
			name: "multiple args",
			args: []string{"arg1", "arg2", "arg3"},
			want: "arg: arg1 arg2 arg3\n",
		},
		{
			name: "args with spaces",
			args: []string{"arg with", "spaces"},
			want: "arg: arg with spaces\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("hookecho failed: %v", err)
			}

			outputStr := string(output)
			if outputStr != tt.want {
				t.Errorf("hookecho output = %q, want %q", outputStr, tt.want)
			}
		})
	}
}

func TestHookecho_WithEnvVars(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	tests := []struct {
		name   string
		env    []string
		want   string
		hasEnv bool
	}{
		{
			name:   "HOOK_ prefix env var",
			env:    []string{"HOOK_TEST=value1"},
			want:   "env: HOOK_TEST=value1\n",
			hasEnv: true,
		},
		{
			name:   "multiple HOOK_ env vars",
			env:    []string{"HOOK_TEST1=value1", "HOOK_TEST2=value2"},
			want:   "env:",
			hasEnv: true,
		},
		{
			name:   "no HOOK_ prefix",
			env:    []string{"TEST=value"},
			want:   "",
			hasEnv: false,
		},
		{
			name:   "mixed env vars",
			env:    []string{"HOOK_TEST=value", "OTHER=value2"},
			want:   "env: HOOK_TEST=value\n",
			hasEnv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath)
			cmd.Env = append(os.Environ(), tt.env...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("hookecho failed: %v", err)
			}

			outputStr := string(output)
			if tt.hasEnv {
				if !strings.Contains(outputStr, "env:") {
					t.Errorf("hookecho output should contain 'env:', got: %q", outputStr)
				}
				// 检查是否包含 HOOK_ 环境变量
				for _, env := range tt.env {
					if strings.HasPrefix(env, "HOOK_") {
						if !strings.Contains(outputStr, env) {
							t.Errorf("hookecho output should contain %q, got: %q", env, outputStr)
						}
					}
				}
			} else {
				if strings.Contains(outputStr, "env:") {
					t.Errorf("hookecho output should not contain 'env:', got: %q", outputStr)
				}
			}
		})
	}
}

func TestHookecho_WithArgsAndEnv(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	cmd := exec.Command(binPath, "test", "args")
	cmd.Env = append(os.Environ(), "HOOK_TEST=value")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hookecho failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "arg: test args\n") {
		t.Errorf("hookecho should output args, got: %q", outputStr)
	}
	if !strings.Contains(outputStr, "env:") {
		t.Errorf("hookecho should output env vars, got: %q", outputStr)
	}
	if !strings.Contains(outputStr, "HOOK_TEST=value") {
		t.Errorf("hookecho should output HOOK_TEST env var, got: %q", outputStr)
	}
}

func TestHookecho_ExitCode(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name:     "exit code 0",
			args:     []string{"exit=0"},
			wantCode: 0,
		},
		{
			name:     "exit code 1",
			args:     []string{"exit=1"},
			wantCode: 1,
		},
		{
			name:     "exit code 42",
			args:     []string{"exit=42"},
			wantCode: 42,
		},
		{
			name:     "exit code 255",
			args:     []string{"exit=255"},
			wantCode: 255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			err := cmd.Run()
			if err == nil {
				if tt.wantCode != 0 {
					t.Errorf("hookecho should exit with code %d, but exited with 0", tt.wantCode)
				}
			} else {
				if exitError, ok := err.(*exec.ExitError); ok {
					actualCode := exitError.ExitCode()
					if actualCode != tt.wantCode {
						t.Errorf("hookecho exit code = %d, want %d", actualCode, tt.wantCode)
					}
				} else {
					t.Fatalf("hookecho failed with non-exit error: %v", err)
				}
			}
		})
	}
}

func TestHookecho_ExitCodeInvalid(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	tests := []struct {
		name     string
		args     []string
		wantCode int // 在 Unix 系统中，os.Exit(-1) 会转换为 255
		wantMsg  string
	}{
		{
			name:     "invalid exit code - not a number",
			args:     []string{"exit=abc"},
			wantCode: 255, // os.Exit(-1) 在 Unix 中转换为 255
			wantMsg:  "Exit code abc not an int!",
		},
		{
			name:     "invalid exit code - empty",
			args:     []string{"exit="},
			wantCode: 255, // os.Exit(-1) 在 Unix 中转换为 255
			wantMsg:  "Exit code  not an int!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("hookecho should exit with error code, but exited with 0")
			} else {
				if exitError, ok := err.(*exec.ExitError); ok {
					actualCode := exitError.ExitCode()
					if actualCode != tt.wantCode {
						t.Errorf("hookecho exit code = %d, want %d", actualCode, tt.wantCode)
					}
				} else {
					t.Fatalf("hookecho failed with non-exit error: %v", err)
				}
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.wantMsg) {
				t.Errorf("hookecho output should contain %q, got: %q", tt.wantMsg, outputStr)
			}
		})
	}
}

func TestHookecho_ExitCodeWithArgs(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	// 测试 exit= 参数与其他参数一起使用
	cmd := exec.Command(binPath, "exit=5", "other", "args")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("hookecho should exit with code 5")
	} else {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 5 {
				t.Errorf("hookecho exit code = %d, want 5", exitError.ExitCode())
			}
		}
	}

	// 检查输出包含参数
	outputStr := string(output)
	if !strings.Contains(outputStr, "arg: exit=5 other args\n") {
		t.Errorf("hookecho should output args before exiting, got: %q", outputStr)
	}
}

func TestHookecho_ExitCodeWithEnv(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	// 测试 exit= 参数与环境变量一起使用
	cmd := exec.Command(binPath, "exit=10")
	cmd.Env = append(os.Environ(), "HOOK_TEST=value")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("hookecho should exit with code 10")
	} else {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 10 {
				t.Errorf("hookecho exit code = %d, want 10", exitError.ExitCode())
			}
		}
	}

	// 检查输出包含环境变量
	outputStr := string(output)
	if !strings.Contains(outputStr, "env:") {
		t.Errorf("hookecho should output env vars before exiting, got: %q", outputStr)
	}
}

func TestHookecho_ExitCodePrefix(t *testing.T) {
	binPath, cleanup := buildHookecho(t)
	defer cleanup()

	// 测试 exit= 前缀匹配（但不是第一个参数）
	// 注意：hookecho 只检查第一个参数是否是 exit=，所以这里不会退出
	cmd := exec.Command(binPath, "notexit=5", "exit=10")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("hookecho should not exit when exit= is not the first arg, got error: %v", err)
	}

	// 检查输出包含所有参数
	outputStr := string(output)
	if !strings.Contains(outputStr, "arg: notexit=5 exit=10\n") {
		t.Errorf("hookecho should output all args, got: %q", outputStr)
	}

	// 测试第一个参数是 exit= 的情况
	cmd2 := exec.Command(binPath, "exit=10", "other", "args")
	output2, err2 := cmd2.CombinedOutput()
	if err2 == nil {
		t.Error("hookecho should exit with code 10 when exit= is first arg")
	} else {
		if exitError, ok := err2.(*exec.ExitError); ok {
			if exitError.ExitCode() != 10 {
				t.Errorf("hookecho exit code = %d, want 10", exitError.ExitCode())
			}
		}
	}

	// 检查输出包含参数（在退出前）
	outputStr2 := string(output2)
	if !strings.Contains(outputStr2, "arg: exit=10 other args\n") {
		t.Errorf("hookecho should output args before exiting, got: %q", outputStr2)
	}
}
