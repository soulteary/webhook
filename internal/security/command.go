package security

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// DefaultMaxArgLength 默认单个参数最大长度（1MB）
	DefaultMaxArgLength = 1024 * 1024
	// DefaultMaxTotalArgsLength 默认所有参数总长度限制（10MB）
	DefaultMaxTotalArgsLength = 10 * 1024 * 1024
	// DefaultMaxArgsCount 默认最大参数数量
	DefaultMaxArgsCount = 1000
)

var (
	// 危险字符模式，用于检测潜在的注入攻击
	dangerousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`[;&|` + "`" + `$(){}]`), // shell 特殊字符
		regexp.MustCompile(`\$\{`),                  // 变量展开
		regexp.MustCompile(`\$\w+`),                 // 变量引用
		regexp.MustCompile(`<|>`),                   // 重定向
		regexp.MustCompile(`\n`),                    // 换行符
	}
)

// CommandValidator 命令验证器
type CommandValidator struct {
	// AllowedPaths 允许的命令路径白名单（目录或文件路径）
	// 如果为空，则不进行白名单检查
	AllowedPaths []string
	// MaxArgLength 单个参数最大长度
	MaxArgLength int
	// MaxTotalArgsLength 所有参数总长度限制
	MaxTotalArgsLength int
	// MaxArgsCount 最大参数数量
	MaxArgsCount int
	// StrictMode 严格模式：如果为 true，则禁止任何包含危险字符的参数
	StrictMode bool
	// SensitivePatterns 敏感信息模式（用于日志脱敏）
	SensitivePatterns []*regexp.Regexp
}

// NewCommandValidator 创建新的命令验证器
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		MaxArgLength:       DefaultMaxArgLength,
		MaxTotalArgsLength: DefaultMaxTotalArgsLength,
		MaxArgsCount:       DefaultMaxArgsCount,
		StrictMode:         false,
		SensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|key|api[_-]?key|auth[_-]?token)\s*[:=]\s*([^\s"']+)`),
			regexp.MustCompile(`(?i)(bearer\s+)([a-zA-Z0-9\-._~+/]+)`),
		},
	}
}

// ValidateCommandPath 验证命令路径是否在白名单中
func (cv *CommandValidator) ValidateCommandPath(cmdPath string) error {
	if len(cv.AllowedPaths) == 0 {
		// 如果没有配置白名单，则允许所有路径（向后兼容）
		return nil
	}

	// 将路径转换为绝对路径
	absPath, err := filepath.Abs(cmdPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %s: %w", cmdPath, err)
	}

	// 检查是否匹配白名单中的任何路径
	for _, allowedPath := range cv.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowedPath)
		if err != nil {
			log.Printf("WARNING: invalid allowed path in whitelist: %s", allowedPath)
			continue
		}

		// 如果是目录，检查命令是否在该目录下
		if strings.HasSuffix(allowedAbs, string(filepath.Separator)) || isDirectory(allowedAbs) {
			if strings.HasPrefix(absPath, allowedAbs) {
				return nil
			}
		} else {
			// 如果是文件路径，精确匹配
			if absPath == allowedAbs {
				return nil
			}
		}
	}

	return fmt.Errorf("command path %s is not in the allowed whitelist", cmdPath)
}

// ValidateArgs 验证命令参数
func (cv *CommandValidator) ValidateArgs(args []string) error {
	// 检查参数数量
	if len(args) > cv.MaxArgsCount {
		return fmt.Errorf("too many arguments: %d (max: %d)", len(args), cv.MaxArgsCount)
	}

	totalLength := 0
	for i, arg := range args {
		// 检查单个参数长度
		argLen := len(arg)
		if argLen > cv.MaxArgLength {
			return fmt.Errorf("argument %d exceeds maximum length: %d (max: %d)", i, argLen, cv.MaxArgLength)
		}

		totalLength += argLen
		if totalLength > cv.MaxTotalArgsLength {
			return fmt.Errorf("total arguments length exceeds maximum: %d (max: %d)", totalLength, cv.MaxTotalArgsLength)
		}

		// 在严格模式下，检查危险字符
		if cv.StrictMode {
			for _, pattern := range dangerousPatterns {
				if pattern.MatchString(arg) {
					return fmt.Errorf("argument %d contains potentially dangerous characters: %s", i, sanitizeForLog(arg))
				}
			}
		}
	}

	return nil
}

// SanitizeForLog 对命令和参数进行脱敏处理，用于日志记录
func (cv *CommandValidator) SanitizeForLog(cmdPath string, args []string) (string, []string) {
	// 脱敏命令路径（如果包含敏感信息）
	sanitizedCmd := sanitizeForLog(cmdPath)

	// 脱敏参数
	sanitizedArgs := make([]string, len(args))
	for i, arg := range args {
		sanitizedArgs[i] = cv.sanitizeArg(arg)
	}

	return sanitizedCmd, sanitizedArgs
}

// sanitizeArg 脱敏单个参数
func (cv *CommandValidator) sanitizeArg(arg string) string {
	// 如果参数太长，截断
	if len(arg) > 200 {
		return arg[:200] + "...[truncated]"
	}

	// 应用敏感信息模式脱敏
	sanitized := arg
	for _, pattern := range cv.SensitivePatterns {
		sanitized = pattern.ReplaceAllStringFunc(sanitized, func(match string) string {
			// 替换敏感值为 "***"
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ":***"
			}
			parts = strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=***"
			}
			// 如果是 bearer token 格式
			if strings.HasPrefix(strings.ToLower(match), "bearer ") {
				return "bearer ***"
			}
			return "***"
		})
	}

	return sanitized
}

// sanitizeForLog 通用的日志脱敏函数
func sanitizeForLog(s string) string {
	if len(s) > 500 {
		return s[:500] + "...[truncated]"
	}
	return s
}

// isDirectory 检查路径是否为目录
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// 如果路径不存在，检查是否以分隔符结尾（可能是目录路径）
		return strings.HasSuffix(path, string(filepath.Separator))
	}
	return info.IsDir()
}

// LogCommandExecution 安全地记录命令执行信息
func (cv *CommandValidator) LogCommandExecution(requestID, hookID, cmdPath string, args []string, envs []string) {
	sanitizedCmd, sanitizedArgs := cv.SanitizeForLog(cmdPath, args)

	// 脱敏环境变量
	sanitizedEnvs := make([]string, len(envs))
	for i, env := range envs {
		// 环境变量格式为 KEY=VALUE
		if idx := strings.Index(env, "="); idx > 0 {
			key := env[:idx]
			value := env[idx+1:]
			// 检查 key 是否包含敏感词
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "password") ||
				strings.Contains(lowerKey, "secret") ||
				strings.Contains(lowerKey, "token") ||
				strings.Contains(lowerKey, "key") ||
				strings.Contains(lowerKey, "auth") {
				sanitizedEnvs[i] = key + "=***"
			} else {
				sanitizedEnvs[i] = key + "=" + cv.sanitizeArg(value)
			}
		} else {
			sanitizedEnvs[i] = env
		}
	}

	log.Printf("[%s] [SECURITY] executing hook %s: command=%s, args=%v, envs=%v",
		requestID, hookID, sanitizedCmd, sanitizedArgs, sanitizedEnvs)
}

// ValidateCommand 综合验证命令路径和参数
func (cv *CommandValidator) ValidateCommand(cmdPath string, args []string) error {
	// 验证命令路径
	if err := cv.ValidateCommandPath(cmdPath); err != nil {
		return fmt.Errorf("command path validation failed: %w", err)
	}

	// 验证参数
	if err := cv.ValidateArgs(args); err != nil {
		return fmt.Errorf("arguments validation failed: %w", err)
	}

	return nil
}

// CommandValidationError 命令验证错误
type CommandValidationError struct {
	Type    string // "path" 或 "args"
	Message string
	Path    string
	Args    []string
}

func (e *CommandValidationError) Error() string {
	return fmt.Sprintf("command validation error [%s]: %s", e.Type, e.Message)
}

// NewCommandValidationError 创建命令验证错误
func NewCommandValidationError(validationType, message, path string, args []string) *CommandValidationError {
	return &CommandValidationError{
		Type:    validationType,
		Message: message,
		Path:    path,
		Args:    args,
	}
}

// IsCommandValidationError 检查错误是否为命令验证错误
func IsCommandValidationError(err error) bool {
	_, ok := err.(*CommandValidationError)
	return ok
}
