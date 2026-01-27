package logger

import (
	"fmt"
	"io"
	"os"

	loggerkit "github.com/soulteary/logger-kit"
)

var (
	// DefaultLogger 是默认的日志记录器
	DefaultLogger *loggerkit.Logger
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
	level := loggerkit.InfoLevel
	if debug {
		level = loggerkit.DebugLevel
	}

	// 设置日志格式
	format := loggerkit.FormatConsole
	if jsonFormat {
		format = loggerkit.FormatJSON
	}

	// 创建 logger-kit 配置
	config := loggerkit.DefaultConfig()
	config.Level = level
	config.Output = writer
	config.Format = format
	config.CallerEnabled = debug // 调试模式下添加源码位置

	// 创建 logger
	DefaultLogger = loggerkit.New(config)
	defaultWriter = writer

	return nil
}

// InitWithWriter 初始化日志系统并指定写入器（用于测试）
func InitWithWriter(writer io.Writer, verbose, debug bool, jsonFormat bool) error {
	// 设置日志级别
	level := loggerkit.InfoLevel
	if debug {
		level = loggerkit.DebugLevel
	}

	// 设置日志格式
	format := loggerkit.FormatConsole
	if jsonFormat {
		format = loggerkit.FormatJSON
	}

	// 创建 logger-kit 配置
	config := loggerkit.DefaultConfig()
	config.Level = level
	config.Output = writer
	config.Format = format
	config.CallerEnabled = debug

	// 创建 logger
	DefaultLogger = loggerkit.New(config)
	defaultWriter = writer

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
func SetDefault(logger *loggerkit.Logger) {
	DefaultLogger = logger
}

// With 返回一个带有附加属性的新日志记录器
func With(args ...any) *loggerkit.Logger {
	if DefaultLogger == nil {
		// 如果未初始化，使用默认配置初始化
		_ = Init(true, false, "", false)
	}
	fields := make(map[string]interface{})
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}
	return DefaultLogger.WithFields(fields)
}

// WithRequestID 返回一个带有请求 ID 的日志记录器
func WithRequestID(requestID string) *loggerkit.Logger {
	if DefaultLogger == nil {
		// 如果未初始化，使用默认配置初始化
		_ = Init(true, false, "", false)
	}
	if requestID != "" {
		return DefaultLogger.WithStr("request_id", requestID)
	}
	return DefaultLogger
}

// WithContext 返回一个带有请求 ID 的日志记录器（从 context 中提取）
func WithContext(ctx interface{}) *loggerkit.Logger {
	if DefaultLogger == nil {
		// 如果未初始化，使用默认配置初始化
		_ = Init(true, false, "", false)
	}
	// logger-kit 的中间件会自动处理 context
	return DefaultLogger
}

// DebugContext 使用 context 记录调试级别日志（自动包含请求 ID）
func DebugContext(ctx interface{}, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		event := logger.Debug()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// InfoContext 使用 context 记录信息级别日志（自动包含请求 ID）
func InfoContext(ctx interface{}, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		event := logger.Info()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// WarnContext 使用 context 记录警告级别日志（自动包含请求 ID）
func WarnContext(ctx interface{}, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		event := logger.Warn()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// ErrorContext 使用 context 记录错误级别日志（自动包含请求 ID）
func ErrorContext(ctx interface{}, msg string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		event := logger.Error()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// DebugfContext 使用 context 和格式化字符串记录调试级别日志（自动包含请求 ID）
func DebugfContext(ctx interface{}, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Debug().Msgf(format, args...)
	}
}

// InfofContext 使用 context 和格式化字符串记录信息级别日志（自动包含请求 ID）
func InfofContext(ctx interface{}, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Info().Msgf(format, args...)
	}
}

// WarnfContext 使用 context 和格式化字符串记录警告级别日志（自动包含请求 ID）
func WarnfContext(ctx interface{}, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Warn().Msgf(format, args...)
	}
}

// ErrorfContext 使用 context 和格式化字符串记录错误级别日志（自动包含请求 ID）
func ErrorfContext(ctx interface{}, format string, args ...any) {
	if DefaultLogger != nil {
		logger := WithContext(ctx)
		logger.Error().Msgf(format, args...)
	}
}

// Debug 记录调试级别日志
func Debug(msg string, args ...any) {
	if DefaultLogger != nil {
		event := DefaultLogger.Debug()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// Info 记录信息级别日志
func Info(msg string, args ...any) {
	if DefaultLogger != nil {
		event := DefaultLogger.Info()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// Warn 记录警告级别日志
func Warn(msg string, args ...any) {
	if DefaultLogger != nil {
		event := DefaultLogger.Warn()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// Error 记录错误级别日志
func Error(msg string, args ...any) {
	if DefaultLogger != nil {
		event := DefaultLogger.Error()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
}

// Debugf 使用格式化字符串记录调试级别日志
func Debugf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Debug().Msgf(format, args...)
	}
}

// Infof 使用格式化字符串记录信息级别日志
func Infof(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Info().Msgf(format, args...)
	}
}

// Warnf 使用格式化字符串记录警告级别日志
func Warnf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Warn().Msgf(format, args...)
	}
}

// Errorf 使用格式化字符串记录错误级别日志
func Errorf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Error().Msgf(format, args...)
	}
}

// Fatal 记录错误级别日志并退出程序
func Fatal(msg string, args ...any) {
	if DefaultLogger != nil {
		event := DefaultLogger.Fatal()
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				event = event.Interface(key, args[i+1])
			}
		}
		event.Msg(msg)
	}
	os.Exit(1)
}

// Fatalf 使用格式化字符串记录错误级别日志并退出程序
func Fatalf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Fatal().Msgf(format, args...)
	}
	os.Exit(1)
}

// Fatalln 记录错误级别日志并退出程序（兼容标准 log 包）
func Fatalln(args ...any) {
	if DefaultLogger != nil {
		msg := fmt.Sprint(args...)
		DefaultLogger.Fatal().Msg(msg)
	}
	os.Exit(1)
}

// Print 记录信息级别日志（兼容标准 log 包）
func Print(args ...any) {
	if DefaultLogger != nil {
		msg := fmt.Sprint(args...)
		DefaultLogger.Info().Msg(msg)
	}
}

// Printf 使用格式化字符串记录信息级别日志（兼容标准 log 包）
func Printf(format string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Info().Msgf(format, args...)
	}
}

// Println 记录信息级别日志（兼容标准 log 包）
func Println(args ...any) {
	if DefaultLogger != nil {
		msg := fmt.Sprintln(args...)
		// 移除末尾的换行符，因为 zerolog 会自动添加
		if len(msg) > 0 && msg[len(msg)-1] == '\n' {
			msg = msg[:len(msg)-1]
		}
		DefaultLogger.Info().Msg(msg)
	}
}
