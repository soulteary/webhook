package flags

import (
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/soulteary/cli-kit/validator"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/i18n"
	"github.com/soulteary/webhook/internal/platform"
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

	switch flags.Profile {
	case "", "compat", "secure":
	default:
		result.AddError("profile", "must be one of: compat, secure")
	}
	if flags.Profile == "secure" && !hasAllowedCommandPath(flags.AllowedCommandPaths) {
		result.AddError("allowed-command-paths", "is required when profile is secure")
	}
	if (flags.SetUID != 0) != (flags.SetGID != 0) {
		result.AddError("setuid/setgid", "must be used together")
	} else if flags.SetUID != 0 && !platform.SupportsPrivilegeDrop() {
		result.AddError("setuid/setgid", "is not supported on this platform")
	}

	// 验证端口范围 - 使用 cli-kit/validator
	if err := validator.ValidatePort(flags.Port); err != nil {
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

	// 验证超时配置 - 使用 cli-kit/validator
	if err := validator.ValidateNonNegative(flags.ReadHeaderTimeoutSeconds); err != nil {
		result.AddError("read-header-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "read-header-timeout-seconds"))
	}
	if err := validator.ValidateNonNegative(flags.ReadTimeoutSeconds); err != nil {
		result.AddError("read-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "read-timeout-seconds"))
	}
	if err := validator.ValidateNonNegative(flags.WriteTimeoutSeconds); err != nil {
		result.AddError("write-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "write-timeout-seconds"))
	}
	if err := validator.ValidateNonNegative(flags.IdleTimeoutSeconds); err != nil {
		result.AddError("idle-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "idle-timeout-seconds"))
	}

	// 验证超时时间逻辑关系
	if flags.ReadTimeoutSeconds > 0 && flags.ReadHeaderTimeoutSeconds > 0 {
		if flags.ReadHeaderTimeoutSeconds > flags.ReadTimeoutSeconds {
			result.AddError("timeout-config", i18n.Sprintf(i18n.ERR_VALIDATE_TIMEOUT_LOGIC, "read-header-timeout", "read-timeout"))
		}
	}

	// 验证限流配置 - 使用 cli-kit/validator
	if flags.RateLimitEnabled {
		if err := validator.ValidatePositive(flags.RateLimitRPS); err != nil {
			result.AddError("rate-limit-rps", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_RATE_LIMIT, "rate-limit-rps"))
		}
		if err := validator.ValidatePositive(flags.RateLimitBurst); err != nil {
			result.AddError("rate-limit-burst", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_RATE_LIMIT, "rate-limit-burst"))
		}
	}

	// 验证 Hook 执行配置 - 使用 cli-kit/validator
	if err := validator.ValidateNonNegative(flags.HookTimeoutSeconds); err != nil {
		result.AddError("hook-timeout-seconds", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "hook-timeout-seconds"))
	}
	if err := validator.ValidatePositive(flags.MaxConcurrentHooks); err != nil {
		result.AddError("max-concurrent-hooks", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-concurrent-hooks"))
	}
	if err := validator.ValidateNonNegative(flags.HookExecutionTimeout); err != nil {
		result.AddError("hook-execution-timeout", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_TIMEOUT, "hook-execution-timeout"))
	}

	// 验证安全配置 - 使用 cli-kit/validator
	if err := validator.ValidatePositive(flags.MaxArgLength); err != nil {
		result.AddError("max-arg-length", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-arg-length"))
	}
	if err := validator.ValidatePositive(flags.MaxTotalArgsLength); err != nil {
		result.AddError("max-total-args-length", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-total-args-length"))
	}
	if err := validator.ValidatePositive(flags.MaxArgsCount); err != nil {
		result.AddError("max-args-count", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-args-count"))
	}

	// 验证大小限制 - 使用 cli-kit/validator
	if err := validator.ValidatePositiveInt64(flags.MaxMultipartMem); err != nil {
		result.AddError("max-multipart-mem", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-multipart-mem"))
	}
	if err := validator.ValidatePositiveInt64(flags.MaxRequestBodySize); err != nil {
		result.AddError("max-request-body-size", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-request-body-size"))
	}
	if err := validator.ValidatePositive(flags.MaxHeaderBytes); err != nil {
		result.AddError("max-header-bytes", i18n.Sprintf(i18n.ERR_VALIDATE_INVALID_POSITIVE_INT, "max-header-bytes"))
	}

	return result
}

func hasAllowedCommandPath(value string) bool {
	for _, path := range strings.Split(value, ",") {
		if strings.TrimSpace(path) != "" {
			return true
		}
	}
	return false
}

// validateFilePath 验证文件路径
func validateFilePath(result *ValidationResult, field, path string, checkWritable, mustExist bool) {
	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)

	// 检查目录是否存在 - 使用 cli-kit/validator
	if err := validator.ValidateDirExists(dir); err != nil {
		result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_NOT_EXIST, dir))
		return
	}

	// 检查目录是否可写 - 使用 cli-kit/validator
	if checkWritable {
		if err := validator.ValidateDirWritable(dir); err != nil {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_NOT_WRITABLE, dir))
		}
	}

	// 如果文件必须存在，检查文件是否存在和可读 - 使用 cli-kit/validator
	if mustExist {
		if err := validator.ValidateFileReadable(cleanPath); err != nil {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_FILE_NOT_EXIST, cleanPath))
		}
	}
}

// validateDirectory 验证目录路径
func validateDirectory(result *ValidationResult, field, path string, mustExist bool) {
	cleanPath := filepath.Clean(path)

	// 使用 cli-kit/validator 验证目录
	err := validator.ValidateDirExists(cleanPath)
	if err != nil {
		// 如果路径是文件而非目录，始终报错
		if errors.Is(err, validator.ErrNotADirectory) {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_NOT_DIRECTORY, cleanPath))
			return
		}
		// 如果目录不存在，只有在 mustExist 为 true 时才报错
		if mustExist && errors.Is(err, validator.ErrDirNotFound) {
			result.AddError(field, i18n.Sprintf(i18n.ERR_VALIDATE_DIR_NOT_EXIST, cleanPath))
		}
	}
}

// validateHookFiles 验证 Hook 文件
func validateHookFiles(result *ValidationResult, flags AppFlags) {
	// Prefer the effective parsed flags. The global list mirrors this slice
	// after parsing and must not be merged with it, or every path appears twice.
	hooksFiles := make(hook.HooksFiles, len(flags.HooksFiles))
	copy(hooksFiles, flags.HooksFiles)
	if len(hooksFiles) == 0 {
		rules.RLockHooksFiles()
		hooksFiles = make(hook.HooksFiles, len(rules.HooksFiles))
		copy(hooksFiles, rules.HooksFiles)
		rules.RUnlockHooksFiles()
	}

	// -hooks-dir 且当前无文件时，不验证（空目录由监控后续发现新文件）
	if flags.HooksDir != "" && len(hooksFiles) == 0 {
		return
	}

	// 未提供 Hook 文件时跳过单文件校验（目录模式可在运行中发现新文件）。
	if len(hooksFiles) == 0 {
		return
	}

	// Reject duplicate paths because runtime would load their Hook IDs twice.
	seen := make(map[string]bool)
	uniqueFiles := make(hook.HooksFiles, 0, len(hooksFiles))
	for _, file := range hooksFiles {
		cleanFile := filepath.Clean(file)
		if seen[cleanFile] {
			result.AddError("hooks", fmt.Sprintf("duplicate hook file %q", file))
			continue
		}
		seen[cleanFile] = true
		uniqueFiles = append(uniqueFiles, file)
	}

	// Hook IDs are global because the HTTP route namespace is shared across files.
	hookOrigins := make(map[string]string)

	// 验证每个 Hook 文件
	for _, hookFile := range uniqueFiles {
		if hookFile == "" {
			continue
		}

		// 验证文件路径
		validateFilePath(result, fmt.Sprintf("hook-file[%s]", hookFile), hookFile, false, true)

		// 尝试加载 Hook 文件以验证格式
		var hooks hook.Hooks
		var err error
		if flags.ValidateStrict {
			err = hooks.LoadFromFileStrict(hookFile, flags.AsTemplate)
		} else {
			err = hooks.LoadFromFile(hookFile, flags.AsTemplate)
		}
		if err != nil {
			result.AddError(fmt.Sprintf("hook-file[%s]", hookFile),
				i18n.Sprintf(i18n.ERR_VALIDATE_HOOK_FILE_LOAD_ERROR, hookFile, err))
			continue
		}

		// 验证 Hook 内容
		validateHookContent(result, hookFile, hooks, hookOrigins)
	}
}

// validateHookContent 验证 Hook 内容
func validateHookContent(result *ValidationResult, hookFile string, hooks hook.Hooks, hookOrigins map[string]string) {
	for i, h := range hooks {
		// 验证 Hook ID
		if h.ID == "" {
			result.AddError(fmt.Sprintf("hook-file[%s].hooks[%d].id", hookFile, i),
				i18n.Sprintf(i18n.ERR_VALIDATE_HOOK_ID_EMPTY))
			continue
		}
		if strings.TrimSpace(h.ExecuteCommand) == "" {
			result.AddError(fmt.Sprintf("hook-file[%s].hooks[%d].execute-command", hookFile, i),
				"must not be empty")
		}
		for field, code := range map[string]int{
			"success-http-response-code":               h.SuccessHttpResponseCode,
			"trigger-rule-mismatch-http-response-code": h.TriggerRuleMismatchHttpResponseCode,
		} {
			if code != 0 && (code < 100 || code > 599) {
				result.AddError(fmt.Sprintf("hook-file[%s].hooks[%d].%s", hookFile, i, field),
					"must be between 100 and 599")
			}
		}

		// 检查重复的 Hook ID
		if _, exists := hookOrigins[h.ID]; exists {
			result.AddError(fmt.Sprintf("hook-file[%s].hooks[%d].id", hookFile, i),
				i18n.Sprintf(i18n.ERR_VALIDATE_HOOK_ID_DUPLICATE, h.ID))
		} else {
			hookOrigins[h.ID] = hookFile
		}
		validateRuleContent(result, fmt.Sprintf("hook-file[%s].hooks[%d].trigger-rule", hookFile, i), h.TriggerRule)

		// 验证命令路径（如果指定了允许的命令路径）
		// 注意：这里只做基本验证，实际执行时的安全检查在 security 模块中
	}
}

func validateRuleContent(result *ValidationResult, field string, rule *hook.Rules) {
	if rule == nil {
		return
	}
	operatorCount := 0
	if rule.And != nil {
		operatorCount++
	}
	if rule.Or != nil {
		operatorCount++
	}
	if rule.Not != nil {
		operatorCount++
	}
	if rule.Match != nil {
		operatorCount++
	}
	if operatorCount != 1 {
		result.AddError(field, "must contain exactly one of: and, or, not, match")
	}
	if rule.Match != nil {
		switch rule.Match.Type {
		case hook.MatchHMACSHA1, hook.MatchHMACSHA256, hook.MatchHMACSHA512,
			hook.MatchHashSHA1, hook.MatchHashSHA256, hook.MatchHashSHA512,
			hook.ScalrSignature:
			if strings.TrimSpace(rule.Match.Secret) == "" {
				result.AddError(field+".match.secret", "must not be empty for a signature rule")
			}
		case hook.MSTeamsSignature:
			if strings.TrimSpace(rule.Match.Secret) == "" {
				result.AddError(field+".match.secret", "must not be empty for a signature rule")
			} else if _, err := base64.StdEncoding.DecodeString(rule.Match.Secret); err != nil {
				result.AddError(field+".match.secret", "must be valid base64 for an msteams-signature rule")
			}
		case hook.MatchRegex:
			if rule.Match.Regex == "" {
				result.AddError(field+".match.regex", "must not be empty for a regex rule")
			} else if _, err := regexp.Compile(rule.Match.Regex); err != nil {
				result.AddError(field+".match.regex", fmt.Sprintf("invalid regular expression: %v", err))
			}
		case hook.MatchValue, hook.IPWhitelist:
		default:
			result.AddError(field+".match.type", fmt.Sprintf("unsupported match type %q", rule.Match.Type))
		}
	}
	if rule.And != nil {
		if len(*rule.And) == 0 {
			result.AddError(field+".and", "must contain at least one rule")
		}
		for i := range *rule.And {
			validateRuleContent(result, fmt.Sprintf("%s.and[%d]", field, i), &(*rule.And)[i])
		}
	}
	if rule.Or != nil {
		if len(*rule.Or) == 0 {
			result.AddError(field+".or", "must contain at least one rule")
		}
		for i := range *rule.Or {
			validateRuleContent(result, fmt.Sprintf("%s.or[%d]", field, i), &(*rule.Or)[i])
		}
	}
	if rule.Not != nil {
		notRule := hook.Rules(*rule.Not)
		validateRuleContent(result, field+".not", &notRule)
	}
}

// isWritable and isReadable functions have been replaced by cli-kit/validator functions:
// - validator.ValidateDirWritable
// - validator.ValidateFileReadable
