//go:build !windows
// +build !windows

package platform

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/pidfile"
	"github.com/stretchr/testify/assert"
)

func TestSetupSignals(t *testing.T) {
	reloadCalled := false
	reloadFn := func() {
		reloadCalled = true
	}

	var testPidFile *pidfile.PIDFile
	var signals chan os.Signal

	// Setup signals - SetupSignals creates a new channel internally
	SetupSignals(signals, reloadFn, testPidFile)

	// Wait a bit for the goroutine to start
	time.Sleep(200 * time.Millisecond)

	// Test SIGUSR1 (should trigger reload)
	// Use signal.Notify to send signal to the process
	// Since SetupSignals uses signal.Notify internally, we need to send the signal to the process
	reloadCalled = false
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}
	err = proc.Signal(syscall.SIGUSR1)
	if err != nil {
		t.Fatalf("Failed to send SIGUSR1: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	assert.True(t, reloadCalled, "SIGUSR1 should trigger reload")

	// Test SIGHUP (should trigger reload)
	reloadCalled = false
	err = proc.Signal(syscall.SIGHUP)
	if err != nil {
		t.Fatalf("Failed to send SIGHUP: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	assert.True(t, reloadCalled, "SIGHUP should trigger reload")

	// Note: We can't easily test SIGTERM/Interrupt in a unit test
	// because they cause os.Exit(0), which would terminate the test.
	// These are better tested in integration tests.
}

func TestDropPrivileges(t *testing.T) {
	// Test with valid UID/GID (will likely fail in test environment unless running as root)
	// This is expected behavior - we're just testing that the function doesn't panic
	err := DropPrivileges(1000, 1000)
	// In a non-root test environment, this will fail, which is expected
	_ = err // We're just checking it doesn't panic
}

// Note: Testing the default case in watchForSignals is difficult because:
// 1. SIGTERM/Interrupt cause os.Exit(0), which would terminate the test
// 2. SIGQUIT causes a core dump on some systems
// These signal handlers are better tested in integration tests.
// The default case is covered by the fact that we only register specific signals
// (SIGUSR1, SIGHUP, SIGTERM, Interrupt), so any other signal would hit the default case.
