//go:build !windows
// +build !windows

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunHookecho_NoArgs(t *testing.T) {
	var buf bytes.Buffer

	shouldExit, exitCode := RunHookecho([]string{"hookecho"}, os.Environ(), &buf)

	if shouldExit {
		t.Errorf("RunHookecho should not exit with no args, got exitCode=%d", exitCode)
	}

	output := buf.String()
	if output != "" {
		t.Errorf("RunHookecho with no args should produce no output, got: %q", output)
	}
}

func TestRunHookecho_WithArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "single arg",
			args: []string{"hookecho", "test"},
			want: "arg: test\n",
		},
		{
			name: "multiple args",
			args: []string{"hookecho", "arg1", "arg2", "arg3"},
			want: "arg: arg1 arg2 arg3\n",
		},
		{
			name: "args with spaces",
			args: []string{"hookecho", "arg with", "spaces"},
			want: "arg: arg with spaces\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			shouldExit, exitCode := RunHookecho(tt.args, os.Environ(), &buf)

			if shouldExit {
				t.Errorf("RunHookecho should not exit, got exitCode=%d", exitCode)
			}

			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("RunHookecho output = %q, want contains %q", output, tt.want)
			}
		})
	}
}

func TestRunHookecho_WithEnvVars(t *testing.T) {
	tests := []struct {
		name   string
		env    []string
		hasEnv bool
	}{
		{
			name:   "HOOK_ prefix env var",
			env:    []string{"HOOK_TEST=value1"},
			hasEnv: true,
		},
		{
			name:   "multiple HOOK_ env vars",
			env:    []string{"HOOK_TEST1=value1", "HOOK_TEST2=value2"},
			hasEnv: true,
		},
		{
			name:   "no HOOK_ prefix",
			env:    []string{"TEST=value"},
			hasEnv: false,
		},
		{
			name:   "mixed env vars",
			env:    []string{"HOOK_TEST=value", "OTHER=value2"},
			hasEnv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			shouldExit, exitCode := RunHookecho([]string{"hookecho"}, tt.env, &buf)

			if shouldExit {
				t.Errorf("RunHookecho should not exit, got exitCode=%d", exitCode)
			}

			output := buf.String()
			if tt.hasEnv {
				if !strings.Contains(output, "env:") {
					t.Errorf("RunHookecho output should contain 'env:', got: %q", output)
				}
				for _, env := range tt.env {
					if strings.HasPrefix(env, "HOOK_") {
						if !strings.Contains(output, env) {
							t.Errorf("RunHookecho output should contain %q, got: %q", env, output)
						}
					}
				}
			} else {
				if strings.Contains(output, "env:") {
					t.Errorf("RunHookecho output should not contain 'env:', got: %q", output)
				}
			}
		})
	}
}

func TestRunHookecho_ExitCode(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   bool
		wantCode   int
		wantOutput string
	}{
		{
			name:       "exit code 0",
			args:       []string{"hookecho", "exit=0"},
			wantExit:   true,
			wantCode:   0,
			wantOutput: "arg: exit=0\n",
		},
		{
			name:       "exit code 1",
			args:       []string{"hookecho", "exit=1"},
			wantExit:   true,
			wantCode:   1,
			wantOutput: "arg: exit=1\n",
		},
		{
			name:       "exit code 42",
			args:       []string{"hookecho", "exit=42"},
			wantExit:   true,
			wantCode:   42,
			wantOutput: "arg: exit=42\n",
		},
		{
			name:       "exit code 255",
			args:       []string{"hookecho", "exit=255"},
			wantExit:   true,
			wantCode:   255,
			wantOutput: "arg: exit=255\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			shouldExit, exitCode := RunHookecho(tt.args, os.Environ(), &buf)

			if shouldExit != tt.wantExit {
				t.Errorf("RunHookecho shouldExit = %v, want %v", shouldExit, tt.wantExit)
			}
			if shouldExit && exitCode != tt.wantCode {
				t.Errorf("RunHookecho exitCode = %d, want %d", exitCode, tt.wantCode)
			}

			output := buf.String()
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("RunHookecho output = %q, want contains %q", output, tt.wantOutput)
			}
		})
	}
}

func TestRunHookecho_ExitCodeInvalid(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   bool
		wantCode   int
		wantOutput string
	}{
		{
			name:       "invalid exit code - not a number",
			args:       []string{"hookecho", "exit=abc"},
			wantExit:   true,
			wantCode:   -1,
			wantOutput: "Exit code abc not an int!",
		},
		{
			name:       "invalid exit code - empty",
			args:       []string{"hookecho", "exit="},
			wantExit:   true,
			wantCode:   -1,
			wantOutput: "Exit code  not an int!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			shouldExit, exitCode := RunHookecho(tt.args, os.Environ(), &buf)

			if shouldExit != tt.wantExit {
				t.Errorf("RunHookecho shouldExit = %v, want %v", shouldExit, tt.wantExit)
			}
			if shouldExit && exitCode != tt.wantCode {
				t.Errorf("RunHookecho exitCode = %d, want %d", exitCode, tt.wantCode)
			}

			output := buf.String()
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("RunHookecho output = %q, want contains %q", output, tt.wantOutput)
			}
		})
	}
}

func TestRunHookecho_WithArgsAndEnv(t *testing.T) {
	var buf bytes.Buffer

	env := append(os.Environ(), "HOOK_TEST=value")
	shouldExit, exitCode := RunHookecho([]string{"hookecho", "test", "args"}, env, &buf)

	if shouldExit {
		t.Errorf("RunHookecho should not exit, got exitCode=%d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "arg: test args\n") {
		t.Errorf("RunHookecho should output args, got: %q", output)
	}
	if !strings.Contains(output, "env:") {
		t.Errorf("RunHookecho should output env vars, got: %q", output)
	}
	if !strings.Contains(output, "HOOK_TEST=value") {
		t.Errorf("RunHookecho should output HOOK_TEST env var, got: %q", output)
	}
}
