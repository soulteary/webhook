package audit

import (
	"testing"
	"time"

	auditkit "github.com/soulteary/audit-kit"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/stretchr/testify/assert"
)

func TestIsEnabledWhenNotInitialized(t *testing.T) {
	// Reset global manager for test
	globalManager = nil

	assert.False(t, IsEnabled(), "IsEnabled should return false when not initialized")
}

func TestLogWhenNotEnabled(t *testing.T) {
	// Reset global manager for test
	globalManager = nil

	// Should not panic when logging without initialization
	record := auditkit.NewRecord(EventHookExecuted, auditkit.ResultSuccess)
	Log(record)
}

func TestNewManagerWithFileStorage(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
		AuditMaskIP:      true,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.True(t, manager.enabled)
	assert.True(t, manager.maskIP)

	// Clean up
	if manager.writer != nil {
		_ = manager.writer.Stop()
	}
}

func TestNewManagerWithNoopStorage(t *testing.T) {
	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "none",
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Clean up
	if manager.writer != nil {
		_ = manager.writer.Stop()
	}
}

func TestLogHookExecuted(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
		AuditMaskIP:      false,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	// Set as global manager for this test
	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	// Log a hook execution
	LogHookExecuted("req-123", "test-hook", "192.168.1.1", "test-agent", 100)

	// Give some time for async write
	time.Sleep(100 * time.Millisecond)

	assert.True(t, IsEnabled())
}

func TestLogHookFailed(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogHookFailed("req-456", "test-hook", "192.168.1.1", "test-agent", "command_failed", 200)

	time.Sleep(100 * time.Millisecond)
}

func TestLogHookTimeout(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogHookTimeout("req-789", "test-hook", "192.168.1.1", "test-agent", 30000)

	time.Sleep(100 * time.Millisecond)
}

func TestLogHookNotFound(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogHookNotFound("req-000", "unknown-hook", "192.168.1.1", "test-agent")

	time.Sleep(100 * time.Millisecond)
}

func TestLogRateLimited(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogRateLimited("req-rate", "test-hook", "192.168.1.1", "test-agent")

	time.Sleep(100 * time.Millisecond)
}

func TestIPMasking(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
		AuditMaskIP:      true,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	// Create a record and verify IP is masked
	record := auditkit.NewRecord(EventHookExecuted, auditkit.ResultSuccess).
		WithIP("192.168.1.100")

	// Before logging, IP should be original
	assert.Equal(t, "192.168.1.100", record.IP)

	// Log will mask the IP
	Log(record)

	// The record's IP should now be masked (in-place modification)
	assert.Contains(t, record.IP, "***")

	time.Sleep(100 * time.Millisecond)
}

func TestGetStats(t *testing.T) {
	// When not initialized
	globalManager = nil
	stats := GetStats()
	assert.Nil(t, stats)

	// When initialized
	tmpFile := t.TempDir() + "/test_audit.log"
	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     2,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	stats = GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 100, stats.QueueCap)
	assert.Equal(t, 2, stats.Workers)
	assert.True(t, stats.Started)
}

func TestLogSignatureEvents(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogSignatureValid("req-sig-1", "test-hook", "192.168.1.1", "sha256")
	LogSignatureInvalid("req-sig-2", "test-hook", "192.168.1.1", "sha256", "invalid_signature")

	time.Sleep(100 * time.Millisecond)
}

func TestLogAccessEvents(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogAccessGranted("req-acc-1", "test-hook", "192.168.1.1", "test-agent")
	LogAccessDenied("req-acc-2", "test-hook", "192.168.1.1", "test-agent", "ip_blocked")

	time.Sleep(100 * time.Millisecond)
}

func TestLogMethodNotAllowed(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogMethodNotAllowed("req-method", "test-hook", "192.168.1.1", "test-agent", "DELETE")

	time.Sleep(100 * time.Millisecond)
}

func TestLogRulesNotSatisfied(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogRulesNotSatisfied("req-rules", "test-hook", "192.168.1.1", "test-agent")

	time.Sleep(100 * time.Millisecond)
}

func TestLogHookTriggered(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogHookTriggered("req-trigger", "test-hook", "192.168.1.1", "test-agent", "POST")

	time.Sleep(100 * time.Millisecond)
}

func TestLogHookCancelled(t *testing.T) {
	tmpFile := t.TempDir() + "/test_audit.log"

	appFlags := flags.AppFlags{
		AuditEnabled:     true,
		AuditStorageType: "file",
		AuditFilePath:    tmpFile,
		AuditQueueSize:   100,
		AuditWorkers:     1,
	}

	manager, err := NewManager(appFlags)
	assert.NoError(t, err)

	oldManager := globalManager
	globalManager = manager
	defer func() {
		globalManager = oldManager
		if manager.writer != nil {
			_ = manager.writer.Stop()
		}
	}()

	LogHookCancelled("req-cancel", "test-hook", "192.168.1.1", "test-agent", 5000)

	time.Sleep(100 * time.Millisecond)
}
