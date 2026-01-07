//go:build !windows
// +build !windows

package platform

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/pidfile"
	"github.com/stretchr/testify/assert"
)

func TestSetupSignals(t *testing.T) {
	reloadCalled := false
	var reloadMutex sync.Mutex
	reloadFn := func() {
		reloadMutex.Lock()
		defer reloadMutex.Unlock()
		reloadCalled = true
	}

	var testPidFile *pidfile.PIDFile
	var signals chan os.Signal

	// Setup signals - SetupSignals creates a new channel internally
	signals = SetupSignals(signals, reloadFn, testPidFile)

	// Ensure signals are stopped and channel is closed when test completes
	t.Cleanup(func() {
		signal.Stop(signals)
		close(signals)
	})

	// Wait a bit for the goroutine to start
	time.Sleep(200 * time.Millisecond)

	// Test SIGUSR1 (should trigger reload)
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}
	err = proc.Signal(syscall.SIGUSR1)
	if err != nil {
		t.Fatalf("Failed to send SIGUSR1: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	reloadMutex.Lock()
	assert.True(t, reloadCalled, "SIGUSR1 should trigger reload")
	reloadMutex.Unlock()

	// Test SIGHUP (should trigger reload)
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	err = proc.Signal(syscall.SIGHUP)
	if err != nil {
		t.Fatalf("Failed to send SIGHUP: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	reloadMutex.Lock()
	assert.True(t, reloadCalled, "SIGHUP should trigger reload")
	reloadMutex.Unlock()
}

func TestSetupSignalsWithHandler(t *testing.T) {
	reloadCalled := false
	var reloadMutex sync.Mutex
	reloadFn := func() {
		reloadMutex.Lock()
		defer reloadMutex.Unlock()
		reloadCalled = true
	}

	exitCalled := false
	exitCode := -1
	var exitMutex sync.Mutex
	mockExit := func(code int) {
		exitMutex.Lock()
		defer exitMutex.Unlock()
		exitCalled = true
		exitCode = code
	}

	var testPidFile *pidfile.PIDFile
	signals := make(chan os.Signal, 1)

	// Setup signals with mock exit function
	signals = SetupSignalsWithHandler(signals, reloadFn, testPidFile, mockExit)

	// Ensure signals are stopped and channel is closed when test completes
	t.Cleanup(func() {
		signal.Stop(signals)
		close(signals)
	})

	// Wait a bit for the goroutine to start
	time.Sleep(200 * time.Millisecond)

	// Test SIGUSR1 (should trigger reload, not exit)
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	signals <- syscall.SIGUSR1
	time.Sleep(100 * time.Millisecond)

	reloadMutex.Lock()
	assert.True(t, reloadCalled, "SIGUSR1 should trigger reload")
	reloadMutex.Unlock()

	exitMutex.Lock()
	assert.False(t, exitCalled, "SIGUSR1 should not trigger exit")
	exitMutex.Unlock()

	// Test SIGTERM (should trigger exit)
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	exitMutex.Lock()
	exitCalled = false
	exitCode = -1
	exitMutex.Unlock()

	signals <- syscall.SIGTERM
	time.Sleep(100 * time.Millisecond)

	exitMutex.Lock()
	assert.True(t, exitCalled, "SIGTERM should trigger exit")
	assert.Equal(t, 0, exitCode, "SIGTERM should exit with code 0")
	exitMutex.Unlock()

	// Test Interrupt (should trigger exit)
	exitMutex.Lock()
	exitCalled = false
	exitCode = -1
	exitMutex.Unlock()

	signals <- os.Interrupt
	time.Sleep(100 * time.Millisecond)

	exitMutex.Lock()
	assert.True(t, exitCalled, "Interrupt should trigger exit")
	assert.Equal(t, 0, exitCode, "Interrupt should exit with code 0")
	exitMutex.Unlock()
}

func TestSignalHandler(t *testing.T) {
	reloadCalled := false
	var reloadMutex sync.Mutex
	reloadFn := func() {
		reloadMutex.Lock()
		defer reloadMutex.Unlock()
		reloadCalled = true
	}

	exitCalled := false
	var exitMutex sync.Mutex
	mockExit := func(code int) {
		exitMutex.Lock()
		defer exitMutex.Unlock()
		exitCalled = true
		_ = code // exitCode is set but not used in this test
	}

	handler := NewSignalHandler(mockExit)
	signals := make(chan os.Signal, 1)
	var testPidFile *pidfile.PIDFile

	// Ensure channel is closed when test completes to stop the goroutine
	t.Cleanup(func() {
		close(signals)
	})

	// Start the signal handler in a goroutine
	go handler.watchForSignals(signals, reloadFn, nil, testPidFile)
	time.Sleep(50 * time.Millisecond)

	// Test SIGHUP
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	signals <- syscall.SIGHUP
	time.Sleep(100 * time.Millisecond)

	reloadMutex.Lock()
	assert.True(t, reloadCalled, "SIGHUP should trigger reload")
	reloadMutex.Unlock()

	// Test default case (unhandled signal)
	// Use SIGQUIT as an unhandled signal for testing (it's not in our switch statement)
	// Note: SIGQUIT is not registered in SetupSignals, so it will hit the default case
	signals <- syscall.SIGQUIT
	time.Sleep(100 * time.Millisecond)

	// Default case should just log, not exit or reload
	exitMutex.Lock()
	assert.False(t, exitCalled, "Unhandled signal should not trigger exit")
	exitMutex.Unlock()
}

func TestDropPrivileges(t *testing.T) {
	// Test with valid UID/GID (will likely fail in test environment unless running as root)
	// This is expected behavior - we're just testing that the function doesn't panic
	err := DropPrivileges(1000, 1000)
	// In a non-root test environment, this will fail, which is expected
	_ = err // We're just checking it doesn't panic
}

func TestDropPrivileges_ErrorPaths(t *testing.T) {
	// Test with invalid UID/GID to trigger error paths
	// In non-root environment, Setgid will fail first, covering that error path
	err := DropPrivileges(-1, -1)
	if err == nil {
		t.Skip("Running as root, cannot test error paths")
	}
	// Verify that error is returned (Setgid or Setuid failed)
	assert.Error(t, err)

	// Test with zero values (might also fail in non-root)
	err2 := DropPrivileges(0, 0)
	if err2 == nil {
		t.Skip("Running as root, cannot test error paths")
	}
	// Verify that error is returned
	assert.Error(t, err2)

	// Test with very large values (will likely fail)
	err3 := DropPrivileges(999999, 999999)
	if err3 == nil {
		t.Skip("Running as root, cannot test error paths")
	}
	// Verify that error is returned
	assert.Error(t, err3)

	// Test with current GID but invalid UID to try to cover Setuid error path
	// This might succeed Setgid but fail Setuid, covering more code paths
	currentGid := os.Getgid()
	err4 := DropPrivileges(-1, currentGid)
	if err4 == nil {
		t.Skip("Running as root, cannot test error paths")
	}
	// This should cover Setuid error path if Setgid succeeds
	assert.Error(t, err4)
}

func TestNewSignalHandler_WithNil(t *testing.T) {
	// Test that NewSignalHandler handles nil exitFunc correctly
	handler := NewSignalHandler(nil)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.exitFunc)
	// The exitFunc should be set to os.Exit when nil is passed
}

func TestSetupSignalsWithHandler_WithNilSignals(t *testing.T) {
	reloadCalled := false
	var reloadMutex sync.Mutex
	reloadFn := func() {
		reloadMutex.Lock()
		defer reloadMutex.Unlock()
		reloadCalled = true
	}

	mockExit := func(code int) {
		_ = code
	}

	var testPidFile *pidfile.PIDFile
	var signals chan os.Signal // nil

	// Setup signals with nil channel - should create a new channel
	signals = SetupSignalsWithHandler(signals, reloadFn, testPidFile, mockExit)
	assert.NotNil(t, signals, "SetupSignalsWithHandler should create a new channel when signals is nil")

	// Ensure signals are stopped and channel is closed when test completes
	t.Cleanup(func() {
		signal.Stop(signals)
		close(signals)
	})

	// Wait a bit for the goroutine to start
	time.Sleep(200 * time.Millisecond)

	// Test SIGUSR1 (should trigger reload)
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	signals <- syscall.SIGUSR1
	time.Sleep(100 * time.Millisecond)

	reloadMutex.Lock()
	assert.True(t, reloadCalled, "SIGUSR1 should trigger reload")
	reloadMutex.Unlock()
}

func TestSetupSignalsWithHandler_WithNilExitFunc(t *testing.T) {
	reloadCalled := false
	var reloadMutex sync.Mutex
	reloadFn := func() {
		reloadMutex.Lock()
		defer reloadMutex.Unlock()
		reloadCalled = true
	}

	var testPidFile *pidfile.PIDFile
	signals := make(chan os.Signal, 1)

	// Setup signals with nil exitFunc - should use os.Exit
	signals = SetupSignalsWithHandler(signals, reloadFn, testPidFile, nil)
	assert.NotNil(t, signals, "SetupSignalsWithHandler should return a valid channel")

	// Ensure signals are stopped and channel is closed when test completes
	t.Cleanup(func() {
		signal.Stop(signals)
		close(signals)
	})

	// Wait a bit for the goroutine to start
	time.Sleep(200 * time.Millisecond)

	// Test SIGUSR1 (should trigger reload)
	reloadMutex.Lock()
	reloadCalled = false
	reloadMutex.Unlock()

	signals <- syscall.SIGUSR1
	time.Sleep(100 * time.Millisecond)

	reloadMutex.Lock()
	assert.True(t, reloadCalled, "SIGUSR1 should trigger reload")
	reloadMutex.Unlock()
}

func TestWatchForSignals_WithPidFile_RemoveError(t *testing.T) {
	reloadFn := func() {
		// Not used in this test, but required by watchForSignals
	}

	exitCalled := false
	var exitMutex sync.Mutex
	mockExit := func(code int) {
		exitMutex.Lock()
		defer exitMutex.Unlock()
		exitCalled = true
		_ = code
	}

	// Create a PIDFile and then remove the file, so Remove() will return an error
	tmpDir := t.TempDir()
	pidFilePath := tmpDir + "/test.pid"
	testPidFile, err := pidfile.New(pidFilePath)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}
	// Remove the file so Remove() will return an error
	os.Remove(pidFilePath)

	handler := NewSignalHandler(mockExit)
	signals := make(chan os.Signal, 1)

	// Ensure channel is closed when test completes to stop the goroutine
	t.Cleanup(func() {
		close(signals)
	})

	// Start the signal handler in a goroutine
	go handler.watchForSignals(signals, reloadFn, nil, testPidFile)
	time.Sleep(50 * time.Millisecond)

	// Test SIGTERM (should trigger exit even if pidFile.Remove() fails)
	exitMutex.Lock()
	exitCalled = false
	exitMutex.Unlock()

	signals <- syscall.SIGTERM
	time.Sleep(100 * time.Millisecond)

	// Exit should still be called even if pidFile.Remove() returns an error
	exitMutex.Lock()
	assert.True(t, exitCalled, "SIGTERM should trigger exit even if pidFile.Remove() fails")
	exitMutex.Unlock()
}

func TestWatchForSignals_WithPidFile_Success(t *testing.T) {
	reloadFn := func() {
		// Not used in this test, but required by watchForSignals
	}

	exitCalled := false
	var exitMutex sync.Mutex
	mockExit := func(code int) {
		exitMutex.Lock()
		defer exitMutex.Unlock()
		exitCalled = true
		_ = code
	}

	// Create a valid PIDFile
	tmpDir := t.TempDir()
	pidFilePath := tmpDir + "/test.pid"
	testPidFile, err := pidfile.New(pidFilePath)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	handler := NewSignalHandler(mockExit)
	signals := make(chan os.Signal, 1)

	// Ensure channel is closed when test completes to stop the goroutine
	t.Cleanup(func() {
		close(signals)
	})

	// Start the signal handler in a goroutine
	go handler.watchForSignals(signals, reloadFn, nil, testPidFile)
	time.Sleep(50 * time.Millisecond)

	// Test SIGTERM (should trigger exit and successfully remove pidFile)
	exitMutex.Lock()
	exitCalled = false
	exitMutex.Unlock()

	signals <- syscall.SIGTERM
	time.Sleep(100 * time.Millisecond)

	// Exit should be called and pidFile should be removed successfully
	exitMutex.Lock()
	assert.True(t, exitCalled, "SIGTERM should trigger exit")
	exitMutex.Unlock()
}

func TestWatchForSignals_WithNilPidFile(t *testing.T) {
	reloadFn := func() {
		// Not used in this test, but required by watchForSignals
	}

	exitCalled := false
	var exitMutex sync.Mutex
	mockExit := func(code int) {
		exitMutex.Lock()
		defer exitMutex.Unlock()
		exitCalled = true
		_ = code
	}

	handler := NewSignalHandler(mockExit)
	signals := make(chan os.Signal, 1)
	var testPidFile *pidfile.PIDFile // nil

	// Ensure channel is closed when test completes to stop the goroutine
	t.Cleanup(func() {
		close(signals)
	})

	// Start the signal handler in a goroutine
	go handler.watchForSignals(signals, reloadFn, nil, testPidFile)
	time.Sleep(50 * time.Millisecond)

	// Test SIGTERM (should trigger exit even if pidFile is nil)
	exitMutex.Lock()
	exitCalled = false
	exitMutex.Unlock()

	signals <- syscall.SIGTERM
	time.Sleep(100 * time.Millisecond)

	// Exit should be called even if pidFile is nil
	exitMutex.Lock()
	assert.True(t, exitCalled, "SIGTERM should trigger exit even if pidFile is nil")
	exitMutex.Unlock()
}
