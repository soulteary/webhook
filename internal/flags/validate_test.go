package flags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/cli-kit/validator"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "port",
		Message: "invalid port number",
	}
	assert.Equal(t, "port: invalid port number", err.Error())
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{}
	assert.False(t, result.HasErrors())

	result.AddError("field1", "error1")
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 1)

	result.AddError("field2", "error2")
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 2)
}

// createValidFlags 创建一个具有所有默认有效值的 AppFlags，用于测试
func createValidFlags() AppFlags {
	return AppFlags{
		Port:               DEFAULT_PORT,
		MaxConcurrentHooks: DEFAULT_MAX_CONCURRENT_HOOKS,
		MaxArgLength:       DEFAULT_MAX_ARG_LENGTH,
		MaxTotalArgsLength: DEFAULT_MAX_TOTAL_ARGS_LENGTH,
		MaxArgsCount:       DEFAULT_MAX_ARGS_COUNT,
		MaxMultipartMem:    int64(DEFAULT_MAX_MPART_MEM),
		MaxRequestBodySize: int64(DEFAULT_MAX_REQUEST_BODY_SIZE),
		MaxHeaderBytes:     DEFAULT_MAX_HEADER_BYTES,
		HooksFiles:         []string{}, // 设置为空，避免验证默认的 hooks.json
	}
}

func TestValidate_Port(t *testing.T) {
	// Create a temporary hooks file to avoid validation errors
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"valid port", 8080, true},
		{"port too small", 0, false},
		{"port too large", 65536, false},
		{"negative port", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.Port = tt.port
			flags.HooksFiles = []string{hookFile}
			result := Validate(flags)
			if tt.valid {
				assert.False(t, result.HasErrors(), "port %d should be valid", tt.port)
			} else {
				// Check that port error is present
				hasPortError := false
				for _, err := range result.Errors {
					if validationErr, ok := err.(*ValidationError); ok && validationErr.Field == "port" {
						hasPortError = true
						break
					}
				}
				assert.True(t, hasPortError, "port %d should have validation error", tt.port)
			}
		})
	}
}

func TestValidate_LogPath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		logPath  string
		hasError bool
		setup    func() string
		cleanup  func(string)
	}{
		{
			name:     "empty log path",
			logPath:  "",
			hasError: false,
		},
		{
			name:     "valid log path",
			logPath:  filepath.Join(tempDir, "test.log"),
			hasError: false,
			setup: func() string {
				return filepath.Join(tempDir, "test.log")
			},
		},
		{
			name:     "non-existent directory",
			logPath:  filepath.Join(tempDir, "nonexistent", "test.log"),
			hasError: true,
		},
		{
			name:     "non-writable directory",
			logPath:  filepath.Join(tempDir, "nowrite", "test.log"),
			hasError: true,
			setup: func() string {
				dir := filepath.Join(tempDir, "nowrite")
				os.Mkdir(dir, 0400) // read-only
				return dir
			},
			cleanup: func(dir string) {
				os.Chmod(dir, 0755)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanupDir string
			if tt.setup != nil {
				cleanupDir = tt.setup()
			}
			if tt.cleanup != nil && cleanupDir != "" {
				defer tt.cleanup(cleanupDir)
			}

			flags := createValidFlags()
			flags.LogPath = tt.logPath
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			} else if tt.logPath != "" {
				// Only check if log path is provided
				// Empty log path is valid and won't trigger validation
			}
		})
	}
}

func TestValidate_PidPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create a temporary hooks file to avoid validation errors
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	flags := createValidFlags()
	flags.PidPath = filepath.Join(tempDir, "test.pid")
	flags.HooksFiles = []string{hookFile}
	result := Validate(flags)
	assert.False(t, result.HasErrors())
}

func TestValidate_I18nDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		i18nDir  string
		hasError bool
	}{
		{"empty i18n dir", "", false},
		{"valid i18n dir", tempDir, false},
		{"non-existent dir", filepath.Join(tempDir, "nonexistent"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.I18nDir = tt.i18nDir
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			}
		})
	}
}

func TestValidate_Timeouts(t *testing.T) {
	tests := []struct {
		name     string
		flags    AppFlags
		hasError bool
	}{
		{
			name: "all valid timeouts",
			flags: AppFlags{
				ReadHeaderTimeoutSeconds: 5,
				ReadTimeoutSeconds:       10,
				WriteTimeoutSeconds:      30,
				IdleTimeoutSeconds:       90,
			},
			hasError: false,
		},
		{
			name: "negative read header timeout",
			flags: AppFlags{
				ReadHeaderTimeoutSeconds: -1,
			},
			hasError: true,
		},
		{
			name: "negative read timeout",
			flags: AppFlags{
				ReadTimeoutSeconds: -1,
			},
			hasError: true,
		},
		{
			name: "negative write timeout",
			flags: AppFlags{
				WriteTimeoutSeconds: -1,
			},
			hasError: true,
		},
		{
			name: "negative idle timeout",
			flags: AppFlags{
				IdleTimeoutSeconds: -1,
			},
			hasError: true,
		},
		{
			name: "read header timeout greater than read timeout",
			flags: AppFlags{
				ReadHeaderTimeoutSeconds: 20,
				ReadTimeoutSeconds:       10,
			},
			hasError: true,
		},
	}

	// Create a temporary hooks file to avoid validation errors
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.HooksFiles = []string{hookFile}
			// Merge test-specific flags
			if tt.flags.ReadHeaderTimeoutSeconds != 0 {
				flags.ReadHeaderTimeoutSeconds = tt.flags.ReadHeaderTimeoutSeconds
			}
			if tt.flags.ReadTimeoutSeconds != 0 {
				flags.ReadTimeoutSeconds = tt.flags.ReadTimeoutSeconds
			}
			if tt.flags.WriteTimeoutSeconds != 0 {
				flags.WriteTimeoutSeconds = tt.flags.WriteTimeoutSeconds
			}
			if tt.flags.IdleTimeoutSeconds != 0 {
				flags.IdleTimeoutSeconds = tt.flags.IdleTimeoutSeconds
			}
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			} else {
				assert.False(t, result.HasErrors())
			}
		})
	}
}

func TestValidate_RateLimit(t *testing.T) {
	tests := []struct {
		name     string
		flags    AppFlags
		hasError bool
	}{
		{
			name: "rate limit disabled",
			flags: AppFlags{
				RateLimitEnabled: false,
			},
			hasError: false,
		},
		{
			name: "valid rate limit",
			flags: AppFlags{
				RateLimitEnabled: true,
				RateLimitRPS:     100,
				RateLimitBurst:   10,
			},
			hasError: false,
		},
		{
			name: "invalid RPS",
			flags: AppFlags{
				RateLimitEnabled: true,
				RateLimitRPS:     0,
				RateLimitBurst:   10,
			},
			hasError: true,
		},
		{
			name: "negative RPS",
			flags: AppFlags{
				RateLimitEnabled: true,
				RateLimitRPS:     -1,
				RateLimitBurst:   10,
			},
			hasError: true,
		},
		{
			name: "invalid burst",
			flags: AppFlags{
				RateLimitEnabled: true,
				RateLimitRPS:     100,
				RateLimitBurst:   0,
			},
			hasError: true,
		},
		{
			name: "negative burst",
			flags: AppFlags{
				RateLimitEnabled: true,
				RateLimitRPS:     100,
				RateLimitBurst:   -1,
			},
			hasError: true,
		},
	}

	// Create a temporary hooks file to avoid validation errors
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.HooksFiles = []string{hookFile}
			// Merge test-specific flags
			flags.RateLimitEnabled = tt.flags.RateLimitEnabled
			if tt.flags.RateLimitRPS != 0 {
				flags.RateLimitRPS = tt.flags.RateLimitRPS
			}
			if tt.flags.RateLimitBurst != 0 {
				flags.RateLimitBurst = tt.flags.RateLimitBurst
			}
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			} else {
				assert.False(t, result.HasErrors())
			}
		})
	}
}

func TestValidate_HookExecution(t *testing.T) {
	tests := []struct {
		name     string
		flags    AppFlags
		hasError bool
	}{
		{
			name: "valid hook timeout",
			flags: AppFlags{
				HookTimeoutSeconds: 30,
			},
			hasError: false,
		},
		{
			name: "negative hook timeout",
			flags: AppFlags{
				HookTimeoutSeconds: -1,
			},
			hasError: true,
		},
		{
			name: "invalid max concurrent hooks",
			flags: AppFlags{
				MaxConcurrentHooks: 0,
			},
			hasError: true,
		},
		{
			name: "negative max concurrent hooks",
			flags: AppFlags{
				MaxConcurrentHooks: -1,
			},
			hasError: true,
		},
		{
			name: "negative hook execution timeout",
			flags: AppFlags{
				HookExecutionTimeout: -1,
			},
			hasError: true,
		},
	}

	// Create a temporary hooks file to avoid validation errors
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.HooksFiles = []string{hookFile}
			// Merge test-specific flags
			if tt.flags.HookTimeoutSeconds != 0 {
				flags.HookTimeoutSeconds = tt.flags.HookTimeoutSeconds
			}
			// Always set MaxConcurrentHooks if it's specified in test flags
			// Use a pointer or check if it's explicitly set in the test case
			if tt.name == "invalid max concurrent hooks" || tt.name == "negative max concurrent hooks" {
				flags.MaxConcurrentHooks = tt.flags.MaxConcurrentHooks
			} else if tt.flags.MaxConcurrentHooks != 0 {
				flags.MaxConcurrentHooks = tt.flags.MaxConcurrentHooks
			}
			if tt.flags.HookExecutionTimeout != 0 {
				flags.HookExecutionTimeout = tt.flags.HookExecutionTimeout
			}
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			} else {
				assert.False(t, result.HasErrors())
			}
		})
	}
}

func TestValidate_Security(t *testing.T) {
	// Create a temporary hooks file to avoid validation errors
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	tests := []struct {
		name     string
		flags    AppFlags
		hasError bool
	}{
		{
			name: "valid security settings",
			flags: AppFlags{
				MaxArgLength:       1024 * 1024,
				MaxTotalArgsLength: 10 * 1024 * 1024,
				MaxArgsCount:       1000,
			},
			hasError: false,
		},
		{
			name: "invalid max arg length",
			flags: AppFlags{
				MaxArgLength: 0,
			},
			hasError: true,
		},
		{
			name: "invalid max total args length",
			flags: AppFlags{
				MaxTotalArgsLength: 0,
			},
			hasError: true,
		},
		{
			name: "invalid max args count",
			flags: AppFlags{
				MaxArgsCount: 0,
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.HooksFiles = []string{hookFile}
			// Merge test-specific flags
			// For invalid test cases, we need to set 0 values explicitly
			if tt.name == "invalid max arg length" || tt.flags.MaxArgLength != 0 {
				flags.MaxArgLength = tt.flags.MaxArgLength
			}
			if tt.name == "invalid max total args length" || tt.flags.MaxTotalArgsLength != 0 {
				flags.MaxTotalArgsLength = tt.flags.MaxTotalArgsLength
			}
			if tt.name == "invalid max args count" || tt.flags.MaxArgsCount != 0 {
				flags.MaxArgsCount = tt.flags.MaxArgsCount
			}
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			} else {
				assert.False(t, result.HasErrors())
			}
		})
	}
}

func TestValidate_SizeLimits(t *testing.T) {
	// Create a temporary hooks file to avoid validation errors
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")
	hookContent := `[]`
	err := os.WriteFile(hookFile, []byte(hookContent), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	oldHooksFiles := rules.HooksFiles
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()
	defer func() {
		rules.LockHooksFiles()
		rules.HooksFiles = oldHooksFiles
		rules.UnlockHooksFiles()
	}()

	tests := []struct {
		name     string
		flags    AppFlags
		hasError bool
	}{
		{
			name: "valid size limits",
			flags: AppFlags{
				MaxMultipartMem:    10 * 1024 * 1024,
				MaxRequestBodySize: 10 * 1024 * 1024,
				MaxHeaderBytes:     1024 * 1024,
			},
			hasError: false,
		},
		{
			name: "invalid max multipart mem",
			flags: AppFlags{
				MaxMultipartMem: 0,
			},
			hasError: true,
		},
		{
			name: "invalid max request body size",
			flags: AppFlags{
				MaxRequestBodySize: 0,
			},
			hasError: true,
		},
		{
			name: "invalid max header bytes",
			flags: AppFlags{
				MaxHeaderBytes: 0,
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := createValidFlags()
			flags.HooksFiles = []string{hookFile}
			// Merge test-specific flags
			// For invalid test cases, we need to set 0 values explicitly
			if tt.name == "invalid max multipart mem" || tt.flags.MaxMultipartMem != 0 {
				flags.MaxMultipartMem = tt.flags.MaxMultipartMem
			}
			if tt.name == "invalid max request body size" || tt.flags.MaxRequestBodySize != 0 {
				flags.MaxRequestBodySize = tt.flags.MaxRequestBodySize
			}
			if tt.name == "invalid max header bytes" || tt.flags.MaxHeaderBytes != 0 {
				flags.MaxHeaderBytes = tt.flags.MaxHeaderBytes
			}
			result := Validate(flags)
			if tt.hasError {
				assert.True(t, result.HasErrors())
			} else {
				assert.False(t, result.HasErrors())
			}
		})
	}
}

func TestValidate_HookFiles(t *testing.T) {
	tempDir := t.TempDir()
	validHookFile := filepath.Join(tempDir, "hooks.json")
	invalidHookFile := filepath.Join(tempDir, "invalid.json")

	// Create a valid hooks file
	validContent := `[
		{
			"id": "test-hook",
			"execute-command": "/bin/echo"
		}
	]`
	err := os.WriteFile(validHookFile, []byte(validContent), 0644)
	require.NoError(t, err)

	// Create an invalid hooks file
	err = os.WriteFile(invalidHookFile, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// Setup rules
	rules.LockHooksFiles()
	rules.HooksFiles = []string{validHookFile}
	rules.UnlockHooksFiles()

	flags := createValidFlags()
	flags.HooksFiles = []string{validHookFile}
	flags.AsTemplate = false
	result := Validate(flags)
	assert.False(t, result.HasErrors())

	// Test with invalid file
	rules.LockHooksFiles()
	rules.HooksFiles = []string{invalidHookFile}
	rules.UnlockHooksFiles()

	flags.HooksFiles = []string{invalidHookFile}
	result = Validate(flags)
	assert.True(t, result.HasErrors())
}

func TestValidate_HookContent(t *testing.T) {
	tempDir := t.TempDir()
	hookFile := filepath.Join(tempDir, "hooks.json")

	// Test with empty hook ID
	content1 := `[
		{
			"id": "",
			"execute-command": "/bin/echo"
		}
	]`
	err := os.WriteFile(hookFile, []byte(content1), 0644)
	require.NoError(t, err)

	rules.LockHooksFiles()
	rules.HooksFiles = []string{hookFile}
	rules.UnlockHooksFiles()

	flags := createValidFlags()
	flags.HooksFiles = []string{hookFile}
	flags.AsTemplate = false
	result := Validate(flags)
	assert.True(t, result.HasErrors())

	// Test with duplicate hook IDs
	content2 := `[
		{
			"id": "duplicate",
			"execute-command": "/bin/echo"
		},
		{
			"id": "duplicate",
			"execute-command": "/bin/echo"
		}
	]`
	err = os.WriteFile(hookFile, []byte(content2), 0644)
	require.NoError(t, err)

	result = Validate(flags)
	assert.True(t, result.HasErrors())
}

func TestValidateFilePath(t *testing.T) {
	tempDir := t.TempDir()
	result := &ValidationResult{}

	// Test with non-existent directory
	validateFilePath(result, "test", filepath.Join(tempDir, "nonexistent", "file.txt"), false, false)
	assert.True(t, result.HasErrors())

	// Reset
	result = &ValidationResult{}

	// Test with valid directory
	filePath := filepath.Join(tempDir, "test.txt")
	validateFilePath(result, "test", filePath, false, false)
	assert.False(t, result.HasErrors())

	// Test with mustExist=true and file doesn't exist
	result = &ValidationResult{}
	validateFilePath(result, "test", filePath, false, true)
	assert.True(t, result.HasErrors())

	// Create the file
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// Test with mustExist=true and file exists
	result = &ValidationResult{}
	validateFilePath(result, "test", filePath, false, true)
	assert.False(t, result.HasErrors())
}

func TestValidateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	result := &ValidationResult{}

	// Test with non-existent directory and mustExist=false
	validateDirectory(result, "test", filepath.Join(tempDir, "nonexistent"), false)
	assert.False(t, result.HasErrors())

	// Test with non-existent directory and mustExist=true
	result = &ValidationResult{}
	validateDirectory(result, "test", filepath.Join(tempDir, "nonexistent"), true)
	assert.True(t, result.HasErrors())

	// Test with valid directory
	result = &ValidationResult{}
	validateDirectory(result, "test", tempDir, false)
	assert.False(t, result.HasErrors())

	// Test with file instead of directory
	filePath := filepath.Join(tempDir, "file.txt")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	result = &ValidationResult{}
	validateDirectory(result, "test", filePath, false)
	assert.True(t, result.HasErrors())
}

func TestValidateDirWritable(t *testing.T) {
	tempDir := t.TempDir()

	// Test writable directory - using cli-kit/validator
	assert.NoError(t, validator.ValidateDirWritable(tempDir))

	// Test non-existent directory
	assert.Error(t, validator.ValidateDirWritable(filepath.Join(tempDir, "nonexistent")))
}

func TestValidateFileReadable(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")

	// Test non-existent file - using cli-kit/validator
	assert.Error(t, validator.ValidateFileReadable(filePath))

	// Create file
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// Test readable file
	assert.NoError(t, validator.ValidateFileReadable(filePath))
}
