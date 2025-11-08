//go:build !windows
// +build !windows

package platform

import (
	"os"
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
	_ = SetupSignals(signals, reloadFn, testPidFile)

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

	// Start the signal handler in a goroutine
	go handler.watchForSignals(signals, reloadFn, testPidFile)
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
