package flags

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/i18n"
	"github.com/soulteary/webhook/internal/rules"
)

// ValidationError 表示配置验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult 包含所有验证错误
type ValidationResult struct {
	Errors []error
}

// AddError 添加一个验证错误
func (r *ValidationResult) AddError(field, message string) {
	r.Errors = append(r.Errors, &ValidationError{Field: field, Message: message})
}

// HasErrors 检查是否有错误
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Validate 验证配置的有效性
func Validate(flags AppFlags) *ValidationResult {
	result := &ValidationResult{}

	// 验证端口范围
	if flags.Port < 1 || flags.Port > 65535 {
		result.AddError("port", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_PORT, flags.Port))
	}

	// 验证日志文件路径
	if flags.LogPath != "" {
		validateFilePath(result, "log-path", flags.LogPath, true, false)
	}

	// 验证 PID 文件路径
	if flags.PidPath != "" {
		validateFilePath(result, "pid-path", flags.PidPath, true, false)
	}

	// 验证 I18n 目录
	if flags.I18nDir != "" {
		validateDirectory(result, "i18n-dir", flags.I18nDir, false)
	}

	// 验证 Hook 文件
	validateHookFiles(result, flags)

	// 验证超时配置
	if flags.ReadHeaderTimeoutSeconds < 0 {
		result.AddError("read-header-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "read-header-timeout-seconds"))
	}
	if flags.ReadTimeoutSeconds < 0 {
		result.AddError("read-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "read-timeout-seconds"))
	}
	if flags.WriteTimeoutSeconds < 0 {
		result.AddError("write-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "write-timeout-seconds"))
	}
	if flags.IdleTimeoutSeconds < 0 {
		result.AddError("idle-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "idle-timeout-seconds"))
	}

	// 验证超时时间逻辑关系
	if flags.ReadTimeoutSeconds > 0 && flags.ReadHeaderTimeoutSeconds > 0 {
		if flags.ReadHeaderTimeoutSeconds > flags.ReadTimeoutSeconds {
			result.AddError("timeout-config", i18n.Sprintf(i18n.ERR_VALIDATE_TIMEOUT_LOGIC, "read-header-timeout", "read-timeout"))
		}
	}

	// 验证限流配置
	if flags.RateLimitEnabled {
		if flags.RateLimitRPS <= 0 {
			result.AddError("rate-limit-rps", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_RATE_LIMIT, "rate-limit-rps"))
		}
		if flags.RateLimitBurst <= 0 {
			result.AddError("rate-limit-burst", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_RATE_LIMIT, "rate-limit-burst"))
		}
	}

	// 验证 Hook 执行配置
	if flags.HookTimeoutSeconds < 0 {
		result.AddError("hook-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "hook-timeout-seconds"))
	}
	if flags.MaxConcurrentHooks <= 0 {
		result.AddError("max-concurrent-hooks", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-concurrent-hooks"))
	}
	if flags.HookExecutionTimeout < 0 {
		result.AddError("hook-execution-timeout", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "hook-execution-timeout"))
	}

	// 验证安全配置
	if flags.MaxArgLength <= 0 {
		result.AddError("max-arg-length", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-arg-length"))
	}
	if flags.MaxTotalArgsLength <= 0 {
		result.AddError("max-total-args-length", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-total-args-length"))
	}
	if flags.MaxArgsCount <= 0 {
		result.AddError("max-args-count", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-args-count"))
	}

	// 验证大小限制
	if flags.MaxMultipartMem <= 0 {
		result.AddError("max-multipart-mem", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-multipart-mem"))
	}
	if flags.MaxRequestBodySize <= 0 {
		result.AddError("max-request-body-size", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-request-body-size"))
	}
	if flags.MaxHeaderBytes <= 0 {
		result.AddError("max-header-bytes", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-header-bytes"))
	}

	return result
}

// validateFilePath 验证文件路径
func validateFilePath(result *ValidationResult, field, path string, checkWritable, mustExist bool) {
	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)

	// 检查目录是否存在
	dirInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_NOT_EXIST, dir))
		} else {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_ACCESS_ERROR, dir, err))
		}
		return
	}

	if !dirInfo.IsDir() {
		result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_NOT_DIRECTORY, dir))
		return
	}

	// 检查目录是否可写
	if checkWritable {
		if !isWritable(dir) {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_NOT_WRITABLE, dir))
		}
	}

	// 如果文件必须存在，检查文件是否存在
	if mustExist {
		fileInfo, err := os.Stat(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_FILE_NOT_EXIST, cleanPath))
			} else {
				result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_FILE_ACCESS_ERROR, cleanPath, err))
			}
			return
		}

		if fileInfo.IsDir() {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_NOT_FILE, cleanPath))
			return
		}

		// 检查文件是否可读
		if !isReadable(cleanPath) {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_FILE_NOT_READABLE, cleanPath))
		}
	}
}

// validateDirectory 验证目录路径
func validateDirectory(result *ValidationResult, field, path string, mustExist bool) {
	cleanPath := filepath.Clean(path)

	dirInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			if mustExist {
				result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_NOT_EXIST, cleanPath))
			}
		} else {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_ACCESS_ERROR, cleanPath, err))
		}
		return
	}

	if !dirInfo.IsDir() {
		result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_NOT_DIRECTORY, cleanPath))
	}
}

// validateHookFiles 验证 Hook 文件
func validateHookFiles(result *ValidationResult, flags AppFlags) {
	// 获取 Hook 文件列表
	rules.RLockHooksFiles()
	hooksFiles := make(hook.HooksFiles, len(rules.HooksFiles))
	copy(hooksFiles, rules.HooksFiles)
	rules.RUnlockHooksFiles()

	// 如果没有指定 Hook 文件，使用默认值
	if len(hooksFiles) == 0 {
		hooksFiles = hook.HooksFiles{"hooks.json"}
	}

	// 合并命令行和环境的 Hook 文件
	if len(flags.HooksFiles) > 0 {
		hooksFiles = append(hooksFiles, flags.HooksFiles...)
	}

	// 去重
	seen := make(map[string]bool)
	uniqueFiles := make(hook.HooksFiles, 0, len(hooksFiles))
	for _, file := range hooksFiles {
		if !seen[file] {
			seen[file] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}

	// 验证每个 Hook 文件
	for _, hookFile := range uniqueFiles {
		if hookFile == "" {
			continue
		}

		// 验证文件路径
		validateFilePath(result, fmt.Sprintf("hook-file[%s]", hookFile), hookFile, false, true)

		// 尝试加载 Hook 文件以验证格式
		var hooks hook.Hooks
		err := hooks.LoadFromFile(hookFile, flags.AsTemplate)
		if err != nil {
			result.AddError(fmt.Sprintf("hook-file[%s]", hookFile),
				i18n.Sprintf(i18n.ERR_VALIDATE_HOOK_FILE_LOAD_ERROR, hookFile, err))
			continue
		}

		// 验证 Hook 内容
		validateHookContent(result, hookFile, hooks)
	}
}

// validateHookContent 验证 Hook 内容
func validateHookContent(result *ValidationResult, hookFile string, hooks hook.Hooks) {
	hookIDs := make(map[string]bool)

	for i, h := range hooks {
		// 验证 Hook ID
		if h.ID == "" {
			result.AddError(fmt.Sprintf("hook-file[%s].hooks[%d].id", hookFile, i),
				i18n.Sprintf(i18n.ERR_VALIDATE_HOOK_ID_EMPTY))
			continue
		}

		// 检查重复的 Hook ID
		if hookIDs[h.ID] {
			result.AddError(fmt.Sprintf("hook-file[%s].hooks[%d].id", hookFile, i),
				i18n.Sprintf(i18n.ERR_VALIDATE_HOOK_ID_DUPLICATE, h.ID))
		}
		hookIDs[h.ID] = true

		// 验证命令路径（如果指定了允许的命令路径）
		// 注意：这里只做基本验证，实际执行时的安全检查在 security 模块中
	}
}

// isWritable 检查目录是否可写
// 使用跨平台方法：尝试创建临时文件来检查写入权限
func isWritable(path string) bool {
	testFile := filepath.Join(path, ".webhook_write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}

// isReadable 检查文件是否可读
func isReadable(path string) bool {
	// 尝试打开文件进行读取
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
