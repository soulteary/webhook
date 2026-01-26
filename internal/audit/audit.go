// Package audit provides audit logging functionality for webhook service
// using the audit-kit module.
package audit

import (
	"context"
	"sync"
	"time"

	auditkit "github.com/soulteary/audit-kit"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/logger"
)

// Webhook-specific event types
const (
	// Hook execution events
	EventHookExecuted  auditkit.EventType = "hook_executed"
	EventHookTriggered auditkit.EventType = "hook_triggered"
	EventHookFailed    auditkit.EventType = "hook_failed"
	EventHookTimeout   auditkit.EventType = "hook_timeout"
	EventHookCancelled auditkit.EventType = "hook_cancelled"

	// Signature verification events
	EventSignatureValid   auditkit.EventType = "signature_valid"
	EventSignatureInvalid auditkit.EventType = "signature_invalid"

	// Access events
	EventHookNotFound      auditkit.EventType = "hook_not_found"
	EventMethodNotAllowed  auditkit.EventType = "method_not_allowed"
	EventRulesNotSatisfied auditkit.EventType = "rules_not_satisfied"
)

// Manager manages the audit logging lifecycle
type Manager struct {
	writer  *auditkit.Writer
	storage auditkit.Storage
	enabled bool
	maskIP  bool
	mu      sync.RWMutex
}

var (
	globalManager *Manager
	once          sync.Once
)

// Init initializes the global audit manager with the given configuration
func Init(appFlags flags.AppFlags) error {
	if !appFlags.AuditEnabled {
		logger.Debug("audit logging is disabled")
		return nil
	}

	var initErr error
	once.Do(func() {
		manager, err := NewManager(appFlags)
		if err != nil {
			initErr = err
			return
		}
		globalManager = manager
		logger.Infof("audit logging enabled: storage=%s, queue_size=%d, workers=%d",
			appFlags.AuditStorageType, appFlags.AuditQueueSize, appFlags.AuditWorkers)
	})

	return initErr
}

// NewManager creates a new audit manager
func NewManager(appFlags flags.AppFlags) (*Manager, error) {
	storageType := auditkit.ParseStorageType(appFlags.AuditStorageType)

	opts := &auditkit.StorageOptions{
		FilePath: appFlags.AuditFilePath,
	}

	// If Redis is enabled and audit storage type is redis, use the Redis client
	if storageType == auditkit.StorageTypeRedis && appFlags.RedisEnabled {
		// Redis storage will be configured separately if needed
		// For now, we just use file storage as fallback
		logger.Warn("Redis audit storage requested but Redis client not available, falling back to file storage")
		storageType = auditkit.StorageTypeFile
	}

	storage, err := auditkit.NewStorageFromType(storageType, opts)
	if err != nil {
		return nil, err
	}

	// If storage is nil (e.g., StorageTypeNone), use NoopStorage
	if storage == nil {
		storage = auditkit.NewNoopStorage()
	}

	writerConfig := &auditkit.WriterConfig{
		QueueSize:   appFlags.AuditQueueSize,
		Workers:     appFlags.AuditWorkers,
		StopTimeout: 10 * time.Second,
	}

	writer := auditkit.NewWriter(storage, writerConfig)
	writer.OnEnqueueFailed(func(record *auditkit.Record) {
		logger.Warnf("[audit] queue full, dropping record: event_type=%s, request_id=%s",
			record.EventType, record.RequestID)
	})
	writer.OnWriteFailed(func(record *auditkit.Record, err error) {
		logger.Errorf("[audit] failed to write record: event_type=%s, request_id=%s, error=%v",
			record.EventType, record.RequestID, err)
	})

	writer.Start()

	return &Manager{
		writer:  writer,
		storage: storage,
		enabled: true,
		maskIP:  appFlags.AuditMaskIP,
	}, nil
}

// IsEnabled returns whether audit logging is enabled
func IsEnabled() bool {
	if globalManager == nil {
		return false
	}
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()
	return globalManager.enabled
}

// Log logs an audit record
func Log(record *auditkit.Record) {
	if globalManager == nil || !globalManager.enabled {
		return
	}

	// Mask IP if configured
	if globalManager.maskIP && record.IP != "" {
		record.IP = auditkit.MaskIP(record.IP)
	}

	globalManager.writer.Enqueue(record)
}

// LogHookExecuted logs a successful hook execution
func LogHookExecuted(requestID, hookID, ip, userAgent string, durationMS int64) {
	record := auditkit.NewRecord(EventHookExecuted, auditkit.ResultSuccess).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithDuration(durationMS)
	Log(record)
}

// LogHookFailed logs a failed hook execution
func LogHookFailed(requestID, hookID, ip, userAgent, reason string, durationMS int64) {
	record := auditkit.NewRecord(EventHookFailed, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason(reason).
		WithDuration(durationMS)
	Log(record)
}

// LogHookTimeout logs a hook execution timeout
func LogHookTimeout(requestID, hookID, ip, userAgent string, durationMS int64) {
	record := auditkit.NewRecord(EventHookTimeout, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason("execution_timeout").
		WithDuration(durationMS)
	Log(record)
}

// LogHookCancelled logs a cancelled hook execution
func LogHookCancelled(requestID, hookID, ip, userAgent string, durationMS int64) {
	record := auditkit.NewRecord(EventHookCancelled, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason("execution_cancelled").
		WithDuration(durationMS)
	Log(record)
}

// LogHookTriggered logs when a hook is triggered (before execution)
func LogHookTriggered(requestID, hookID, ip, userAgent, method string) {
	record := auditkit.NewRecord(EventHookTriggered, auditkit.ResultSuccess).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithMetadata("method", method)
	Log(record)
}

// LogHookNotFound logs when a hook is not found
func LogHookNotFound(requestID, hookID, ip, userAgent string) {
	record := auditkit.NewRecord(EventHookNotFound, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason("hook_not_found")
	Log(record)
}

// LogMethodNotAllowed logs when HTTP method is not allowed
func LogMethodNotAllowed(requestID, hookID, ip, userAgent, method string) {
	record := auditkit.NewRecord(EventMethodNotAllowed, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason("method_not_allowed").
		WithMetadata("method", method)
	Log(record)
}

// LogRulesNotSatisfied logs when trigger rules are not satisfied
func LogRulesNotSatisfied(requestID, hookID, ip, userAgent string) {
	record := auditkit.NewRecord(EventRulesNotSatisfied, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason("rules_not_satisfied")
	Log(record)
}

// LogSignatureValid logs successful signature verification
func LogSignatureValid(requestID, hookID, ip, algorithm string) {
	record := auditkit.NewRecord(EventSignatureValid, auditkit.ResultSuccess).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithMetadata("algorithm", algorithm)
	Log(record)
}

// LogSignatureInvalid logs failed signature verification
func LogSignatureInvalid(requestID, hookID, ip, algorithm, reason string) {
	record := auditkit.NewRecord(EventSignatureInvalid, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithReason(reason).
		WithMetadata("algorithm", algorithm)
	Log(record)
}

// LogRateLimited logs when a request is rate limited
func LogRateLimited(requestID, hookID, ip, userAgent string) {
	record := auditkit.NewRecord(auditkit.EventRateLimited, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason("rate_limited")
	Log(record)
}

// LogAccessGranted logs successful access to a hook
func LogAccessGranted(requestID, hookID, ip, userAgent string) {
	record := auditkit.NewRecord(auditkit.EventAccessGranted, auditkit.ResultSuccess).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent)
	Log(record)
}

// LogAccessDenied logs denied access to a hook
func LogAccessDenied(requestID, hookID, ip, userAgent, reason string) {
	record := auditkit.NewRecord(auditkit.EventAccessDenied, auditkit.ResultFailure).
		WithRequestID(requestID).
		WithResource(hookID).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithReason(reason)
	Log(record)
}

// Shutdown gracefully shuts down the audit manager
func Shutdown(ctx context.Context) error {
	if globalManager == nil {
		return nil
	}

	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if !globalManager.enabled {
		return nil
	}

	globalManager.enabled = false
	logger.Info("shutting down audit logging...")

	if globalManager.writer != nil {
		if err := globalManager.writer.Stop(); err != nil {
			return err
		}
	}

	logger.Info("audit logging shutdown completed")
	return nil
}

// GetStats returns audit writer statistics (for monitoring)
func GetStats() *auditkit.Stats {
	if globalManager == nil || globalManager.writer == nil {
		return nil
	}
	stats := globalManager.writer.GetStats()
	return &stats
}
