package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

var (
	// DefaultLogger 是默认的日志记录器
	DefaultLogger *slog.Logger
	// defaultHandler 是默认的处理器
	defaultHandler slog.Handler
	// defaultWriter 是默认的写入器
	defaultWriter io.Writer
)

// Init 初始化日志系统
// verbose: 是否启用详细日志
// debug: 是否启用调试日志
// logPath: 日志文件路径，为空则输出到标准错误输出（stderr）以兼容测试
// jsonFormat: 是否使用 JSON 格式
func Init(verbose, debug bool, logPath string, jsonFormat bool) error {
	var writer io.Writer = os.Stderr // 默认输出到 stderr 以兼容测试

	// 如果 verbose 为 false，则禁用日志输出
	if !verbose {
		writer = io.Discard
	} else if logPath != "" {
		// 打开日志文件
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		writer = logFile
	}

	// 设置日志级别
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	// 创建处理器
	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:     level,
			AddSource: debug, // 调试模式下添加源码位置
		})
	} else {
		// 使用自定义的简单文本处理器，只输出消息内容，兼容旧格式
		handler = newSimpleTextHandler(writer, level)
	}

	defaultHandler = handler
	defaultWriter = writer
	DefaultLogger = slog.New(handler)
	slog.SetDefault(DefaultLogger)

	return nil
}

// InitWithWriter 初始化日志系统并指定写入器（用于测试）
func InitWithWriter(writer io.Writer, verbose, debug bool, jsonFormat bool) error {
	// 设置日志级别
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	// 创建处理器
	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:     level,
			AddSource: debug,
		})
	} else {
		handler = newSimpleTextHandler(writer, level)
	}

	defaultHandler = handler
	defaultWriter = writer
	DefaultLogger = slog.New(handler)
	slog.SetDefault(DefaultLogger)

	return nil
}

// Writer 返回日志写入器（用于兼容需要 io.Writer 的场景）
func Writer() io.Writer {
	if defaultWriter != nil {
		return defaultWriter
	}
	return os.Stderr
}

// SetDefault 设置默认日志记录器
func SetDefault(logger *slog.Logger) {
	DefaultLogger = logger
	slog.SetDefault(logger)
}

// With 返回一个带有附加属性的新日志记录器
func With(args ...any) *slog.Logger {
	if DefaultLogger == nil {
		// 如果未初始化，使用默认配置初始化
		Init(true, false, "", false)
	}
	return DefaultLogger.With(args...)
}

// WithRequestID 返回一个带有请求 ID 的日志记录器
func WithRequestID(requestID string) *slog.Logger {
	if DefaultLogger == nil {
		// 如果未初始化，使用默认配置初始化
		Init(true, false, "", false)
	}
	if requestID != "" {
		return DefaultLogger.With("request_id", requestID)
	}
	return DefaultLogger
}

// WithContext 返回一个带有请求 ID 的日志记录器（从 context 中提取）
// 注意：context 中必须包含通过 middleware.RequestIDKey 设置的请求 ID
// 如果需要使用此函数，请确保 context 中已设置请求 ID
func WithContext(ctx context.Context) *slog.Logger {
	if DefaultLogger == nil {
		// 如果未初始化，使用默认配置初始化
		Init(true, false, "", false)
	}
	if ctx == nil {
		return DefaultLogger
	}
	// 从 context 中提取请求 ID（使用与 middleware.RequestIDKey 相同的 key）
	type ctxKeyRequestID int
	const RequestIDKey ctxKeyRequestID = 0
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return DefaultLogger.With("request_id", reqID)
	}
	return DefaultLogger
}

// DebugContext 使用 context 记录调试级别日志（自动包含请求 ID）
func DebugContext(ctx context.Context, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Debug(msg, args...)
	}
}

// InfoContext 使用 context 记录信息级别日志（自动包含请求 ID）
func InfoContext(ctx context.Context, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Info(msg, args...)
	}
}

// WarnContext 使用 context 记录警告级别日志（自动包含请求 ID）
func WarnContext(ctx context.Context, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Warn(msg, args...)
	}
}

// ErrorContext 使用 context 记录错误级别日志（自动包含请求 ID）
func ErrorContext(ctx context.Context, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Error(msg, args...)
	}
}

// DebugfContext 使用 context 和格式化字符串记录调试级别日志（自动包含请求 ID）
func DebugfContext(ctx context.Context, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Debug(fmt.Sprintf(format, args...))
	}
}

// InfofContext 使用 context 和格式化字符串记录信息级别日志（自动包含请求 ID）
func InfofContext(ctx context.Context, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Info(fmt.Sprintf(format, args...))
	}
}

// WarnfContext 使用 context 和格式化字符串记录警告级别日志（自动包含请求 ID）
func WarnfContext(ctx context.Context, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Warn(fmt.Sprintf(format, args...))
	}
}

// ErrorfContext 使用 context 和格式化字符串记录错误级别日志（自动包含请求 ID）
func ErrorfContext(ctx context.Context, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Error(fmt.Sprintf(format, args...))
	}
}

// Debug 记录调试级别日志
func Debug(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(msg, args...)
	}
}

// Info 记录信息级别日志
func Info(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Info(msg, args...)
	}
}

// Warn 记录警告级别日志
func Warn(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(msg, args...)
	}
}

// Error 记录错误级别日志
func Error(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Error(msg, args...)
	}
}

// Debugf 使用格式化字符串记录调试级别日志
func Debugf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(fmt.Sprintf(format, args...))
	}
}

// Infof 使用格式化字符串记录信息级别日志
func Infof(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Info(fmt.Sprintf(format, args...))
	}
}

// Warnf 使用格式化字符串记录警告级别日志
func Warnf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(fmt.Sprintf(format, args...))
	}
}

// Errorf 使用格式化字符串记录错误级别日志
func Errorf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Error(fmt.Sprintf(format, args...))
	}
}

// Fatal 记录错误级别日志并退出程序
func Fatal(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Error(msg, args...)
	}
	os.Exit(1)
}

// Fatalf 使用格式化字符串记录错误级别日志并退出程序
func Fatalf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Error(fmt.Sprintf(format, args...))
	}
	os.Exit(1)
}

// Fatalln 记录错误级别日志并退出程序（兼容标准 log 包）
func Fatalln(args ...any) {
	if DefaultLogger != nil {
		msg := fmt.Sprint(args...)
		DefaultLogger.Error(msg)
	}
	os.Exit(1)
}

// Print 记录信息级别日志（兼容标准 log 包）
func Print(args ...any) {
	if DefaultLogger != nil {
		msg := fmt.Sprint(args...)
		DefaultLogger.Info(msg)
	}
}

// Printf 使用格式化字符串记录信息级别日志（兼容标准 log 包）
func Printf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Info(fmt.Sprintf(format, args...))
	}
}

// Println 记录信息级别日志（兼容标准 log 包）
func Println(args ...any) {
	if DefaultLogger != nil {
		msg := fmt.Sprintln(args...)
		// 移除末尾的换行符，因为 slog 会自动添加
		if len(msg) > 0 && msg[len(msg)-1] == '\n' {
			msg = msg[:len(msg)-1]
		}
		DefaultLogger.Info(msg)
	}
}

// simpleTextHandler 是一个文本处理器，输出统一格式的日志（包含时间戳、级别、消息和属性）
type simpleTextHandler struct {
	writer io.Writer
	level  slog.Level
}

func newSimpleTextHandler(writer io.Writer, level slog.Level) slog.Handler {
	return &simpleTextHandler{
		writer: writer,
		level:  level,
	}
}

func (h *simpleTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *simpleTextHandler) Handle(ctx context.Context, record slog.Record) error {
	// 统一日志格式：时间戳 | 级别 | 消息 | 属性
	// 时间戳格式：2006-01-02T15:04:05.000Z07:00 (RFC3339 with milliseconds)
	timestamp := record.Time.Format(time.RFC3339Nano)
	levelStr := record.Level.String()

	// 构建日志行
	var buf []byte
	buf = append(buf, timestamp...)
	buf = append(buf, " | "...)
	buf = append(buf, levelStr...)
	buf = append(buf, " | "...)
	buf = append(buf, record.Message...)

	// 添加属性
	record.Attrs(func(a slog.Attr) bool {
		if a.Key != "" {
			buf = append(buf, " | "...)
			buf = append(buf, a.Key...)
			buf = append(buf, "="...)
			buf = append(buf, fmt.Sprintf("%v", a.Value.Any())...)
		}
		return true
	})

	buf = append(buf, '\n')
	_, err := h.writer.Write(buf)
	return err
}

func (h *simpleTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 创建一个新的处理器，包含额外的属性
	// 注意：slog 会在调用 Handle 时自动合并这些属性
	return &simpleTextHandler{
		writer: h.writer,
		level:  h.level,
	}
}

func (h *simpleTextHandler) WithGroup(name string) slog.Handler {
	// 对于简单处理器，忽略分组
	return h
}
