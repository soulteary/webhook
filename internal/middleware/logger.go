package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/soulteary/webhook/internal/logger"
)

// Logger is a middleware that logs useful data about each HTTP request.
type Logger struct {
	Logger middleware.LoggerInterface
}

// NewLogger creates a new RequestLogger Handler.
func NewLogger() func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&Logger{})
}

// NewLogEntry creates a new LogEntry for the request.
func (l *Logger) NewLogEntry(r *http.Request) middleware.LogEntry {
	e := &LogEntry{
		req: r,
		buf: &bytes.Buffer{},
	}

	return e
}

// LogEntry represents an individual log entry.
type LogEntry struct {
	*Logger
	req *http.Request
	buf *bytes.Buffer
}

// Write constructs and writes the final log entry.
func (l *LogEntry) Write(status int, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	rid := GetReqID(l.req.Context())

	// 使用结构化日志格式
	args := []any{
		"status", status,
		"bytes", bytes,
		"size", humanize.IBytes(uint64(bytes)),
		"elapsed", elapsed.String(),
		"host", l.req.Host,
		"method", l.req.Method,
		"uri", l.req.RequestURI,
	}

	if rid != "" {
		args = append(args, "request_id", rid)
	}

	logger.Info("HTTP request completed", args...)
}

// Panic prints the call stack for a panic.
func (l *LogEntry) Panic(v interface{}, stack []byte) {
	rid := GetReqID(l.req.Context())

	args := []any{
		"panic_value", fmt.Sprintf("%#v", v),
		"stack", string(stack),
	}

	if rid != "" {
		args = append(args, "request_id", rid)
	}

	logger.Error("panic occurred", args...)
}
