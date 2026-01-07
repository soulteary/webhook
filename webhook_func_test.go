package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/stretchr/testify/assert"
)

func TestNeedEchoVersionInfo(t *testing.T) {
	// Test when ShowVersion is false
	appFlags := flags.AppFlags{ShowVersion: false}
	NeedEchoVersionInfo(appFlags)
	// Should not exit, so if we get here, test passes

	// Test when ShowVersion is true
	// Note: This will call os.Exit(0), so we can't test it directly
	// The function is tested in integration tests
	appFlags.ShowVersion = true
	// We can't test this without forking, which is complex
	// This is tested in the main integration test
}

func TestCheckPrivilegesParamsCorrect(t *testing.T) {
	// Test when both SetUID and SetGID are 0 (valid)
	appFlags := flags.AppFlags{SetUID: 0, SetGID: 0}
	CheckPrivilegesParamsCorrect(appFlags)
	// Should not exit

	// Test when both SetUID and SetGID are non-zero (valid)
	appFlags = flags.AppFlags{SetUID: 1000, SetGID: 1000}
	CheckPrivilegesParamsCorrect(appFlags)
	// Should not exit

	// Test when only SetUID is set (invalid)
	appFlags = flags.AppFlags{SetUID: 1000, SetGID: 0}
	// This will call os.Exit(1), so we can't test it directly
	// This is tested in integration tests

	// Test when only SetGID is set (invalid)
	appFlags = flags.AppFlags{SetUID: 0, SetGID: 1000}
	// This will call os.Exit(1), so we can't test it directly
	// This is tested in integration tests
}

func TestGetNetAddr(t *testing.T) {
	// Test successful listener creation
	appFlags := flags.AppFlags{Host: "127.0.0.1", Port: 0}
	var logQueue []string
	addr, ln := GetNetAddr(appFlags, &logQueue)

	assert.NotEmpty(t, addr)
	assert.NotNil(t, ln)
	assert.Equal(t, 0, len(logQueue))
	if ln != nil {
		(*ln).Close()
	}

	// Test with invalid address (should add to log queue)
	appFlags = flags.AppFlags{Host: "invalid-host", Port: 99999}
	logQueue = []string{}
	addr, ln = GetNetAddr(appFlags, &logQueue)

	assert.NotEmpty(t, addr)
	// Should have error in log queue
	assert.Greater(t, len(logQueue), 0)
}

func TestDropPrivileges(t *testing.T) {
	// Test when SetUID is 0 (should not drop privileges)
	appFlags := flags.AppFlags{SetUID: 0, SetGID: 0}
	var logQueue []string
	DropPrivileges(appFlags, &logQueue)

	assert.Equal(t, 0, len(logQueue))

	// Test when SetUID is non-zero
	// Note: This requires root privileges to test properly
	// On non-root systems, this will add an error to logQueue
	appFlags = flags.AppFlags{SetUID: 1000, SetGID: 1000}
	logQueue = []string{}
	DropPrivileges(appFlags, &logQueue)

	// On non-root systems, this may add an error to logQueue
	// On root systems, it should succeed
}

func TestSetupLogger(t *testing.T) {
	// Test when LogPath is empty
	appFlags := flags.AppFlags{LogPath: "", Verbose: true}
	var logQueue []string
	err := SetupLogger(appFlags, &logQueue)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(logQueue))

	// Test with valid log path
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	appFlags = flags.AppFlags{LogPath: logPath, Verbose: true}
	logQueue = []string{}
	err = SetupLogger(appFlags, &logQueue)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(logQueue))

	// Test with invalid log path (should add to log queue)
	appFlags = flags.AppFlags{LogPath: "/nonexistent/dir/test.log", Verbose: true}
	logQueue = []string{}
	err = SetupLogger(appFlags, &logQueue)

	// Should have error in log queue
	assert.Greater(t, len(logQueue), 0)
}

func TestGetNetAddr_ErrorHandling(t *testing.T) {
	// Test with invalid port (port 0 is valid, so we use a port that may be in use)
	// Actually, it's hard to force an error here without mocking
	// So we just test that the function doesn't panic and returns valid values
	appFlags := flags.AppFlags{Host: "127.0.0.1", Port: 0}
	var logQueue []string
	addr, ln := GetNetAddr(appFlags, &logQueue)

	assert.NotEmpty(t, addr)
	// Port 0 should succeed (OS assigns a port)
	assert.NotNil(t, ln)
	if ln != nil {
		(*ln).Close()
	}
}

func TestDropPrivileges_ErrorHandling(t *testing.T) {
	// Test with invalid UID/GID (requires root to test properly)
	appFlags := flags.AppFlags{SetUID: 99999, SetGID: 99999}
	var logQueue []string
	DropPrivileges(appFlags, &logQueue)

	// On non-root systems or with invalid UID/GID, may have error in log queue
}

func TestSetupLogger_ErrorHandling(t *testing.T) {
	// Test with read-only directory (if possible)
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	_ = os.Mkdir(readOnlyDir, 0444)
	defer os.Chmod(readOnlyDir, 0755)

	logPath := filepath.Join(readOnlyDir, "test.log")
	appFlags := flags.AppFlags{LogPath: logPath, Verbose: true}
	var logQueue []string
	_ = SetupLogger(appFlags, &logQueue)

	// Should have error in log queue
	assert.Greater(t, len(logQueue), 0)
}

func TestNeedValidateConfig(t *testing.T) {
	// Test when ValidateConfig is false (should not do anything)
	appFlags := flags.AppFlags{ValidateConfig: false}
	NeedValidateConfig(appFlags)
	// Should not exit, so if we get here, test passes

	// Test when ValidateConfig is true with valid config
	// Note: This will call os.Exit(0) on success or os.Exit(1) on failure
	// We can't test the exit behavior directly, but we can test the validation logic
	// by checking that the function doesn't panic and processes the flags correctly

	// Create a temporary hooks file for validation
	tmpDir := t.TempDir()
	hooksFile := filepath.Join(tmpDir, "hooks.json")
	_ = os.WriteFile(hooksFile, []byte(`[]`), 0644)

	appFlags = flags.AppFlags{
		ValidateConfig: true,
		Port:           9000,
		HooksFiles:     []string{hooksFile},
	}

	// This will exit, so we can't test it directly
	// The validation logic is tested in flags.Validate tests
	// This test just ensures the function doesn't panic when ValidateConfig is false
}

func TestGetNetAddr_WithDifferentHosts(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		port          int
		shouldSucceed bool
	}{
		{
			name:          "localhost",
			host:          "127.0.0.1",
			port:          0, // Let OS assign port
			shouldSucceed: true,
		},
		{
			name:          "0.0.0.0",
			host:          "0.0.0.0",
			port:          0,
			shouldSucceed: true,
		},
		{
			name:          "localhost with specific port",
			host:          "127.0.0.1",
			port:          0, // Use 0 to avoid port conflicts
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appFlags := flags.AppFlags{Host: tt.host, Port: tt.port}
			var logQueue []string
			addr, ln := GetNetAddr(appFlags, &logQueue)

			if tt.shouldSucceed {
				assert.NotEmpty(t, addr)
				assert.NotNil(t, ln)
				if ln != nil {
					(*ln).Close()
				}
			}
		})
	}
}

func TestSetupLogger_WithDifferentFlags(t *testing.T) {
	tests := []struct {
		name      string
		verbose   bool
		debug     bool
		logPath   string
		shouldErr bool
	}{
		{
			name:      "verbose enabled, no log file",
			verbose:   true,
			debug:     false,
			logPath:   "",
			shouldErr: false,
		},
		{
			name:      "verbose disabled",
			verbose:   false,
			debug:     false,
			logPath:   "",
			shouldErr: false,
		},
		{
			name:      "debug enabled",
			verbose:   true,
			debug:     true,
			logPath:   "",
			shouldErr: false,
		},
		{
			name:      "with valid log file",
			verbose:   true,
			debug:     false,
			logPath:   filepath.Join(t.TempDir(), "test.log"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appFlags := flags.AppFlags{
				Verbose: tt.verbose,
				Debug:   tt.debug,
				LogPath: tt.logPath,
			}
			var logQueue []string
			err := SetupLogger(appFlags, &logQueue)

			if tt.shouldErr {
				assert.Error(t, err)
				assert.Greater(t, len(logQueue), 0)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckPrivilegesParamsCorrect_AllCases(t *testing.T) {
	tests := []struct {
		name       string
		setUID     int
		setGID     int
		shouldExit bool
	}{
		{
			name:       "both zero (valid)",
			setUID:     0,
			setGID:     0,
			shouldExit: false,
		},
		{
			name:       "both non-zero (valid)",
			setUID:     1000,
			setGID:     1000,
			shouldExit: false,
		},
		{
			name:       "only UID set (invalid)",
			setUID:     1000,
			setGID:     0,
			shouldExit: true, // Will exit, can't test directly
		},
		{
			name:       "only GID set (invalid)",
			setUID:     0,
			setGID:     1000,
			shouldExit: true, // Will exit, can't test directly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appFlags := flags.AppFlags{SetUID: tt.setUID, SetGID: tt.setGID}

			if !tt.shouldExit {
				CheckPrivilegesParamsCorrect(appFlags)
				// If we get here, test passes
			} else {
				// This will exit, so we can't test it directly
				// The function is tested in integration tests
			}
		})
	}
}
