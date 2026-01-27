package i18n

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	i18nkit "github.com/soulteary/i18n-kit"
	"github.com/soulteary/webhook/internal/fn"
)

var (
	// GLOBAL_BUNDLE 全局翻译 bundle
	GLOBAL_BUNDLE *i18nkit.Bundle
	// GLOBAL_LANG 全局语言设置
	GLOBAL_LANG string
)

// InitLocaleByFiles 从文件初始化翻译 bundle
func InitLocaleByFiles(localesDir string, webhookLocalesEmbed embed.FS) error {
	// 创建 bundle，默认语言为英文
	bundle := i18nkit.NewBundle(i18nkit.LangEN)

	// 首先尝试从目录加载
	var localesFiles []string
	localesFiles = append(localesFiles, fn.ScanDirByExt(localesDir, ".yaml")...)
	localesFiles = append(localesFiles, fn.ScanDirByExt(localesDir, ".yml")...)
	localesFiles = append(localesFiles, fn.ScanDirByExt(localesDir, ".json")...)
	if len(localesFiles) > 0 {
		// 从目录加载
		for _, file := range localesFiles {
			fileName := filepath.Base(file)
			ext := strings.ToLower(filepath.Ext(fileName))
			langCode := strings.TrimSuffix(fileName, ext)

			// 验证语言代码
			verified := fn.GetVerifiedLocalCode(langCode)
			if verified == "" {
				continue
			}

			// 解析语言
			lang, ok := i18nkit.ParseLanguage(verified)
			if !ok {
				// 如果无法解析，尝试使用别名
				lang = i18nkit.LangEN // 默认使用英文
			}

			// 根据文件扩展名加载
			switch ext {
			case ".yaml", ".yml":
				if err := bundle.LoadYAMLFile(lang, file); err != nil {
					fmt.Printf("failed to load YAML file %s: %v\n", file, err)
					continue
				}
			case ".json":
				if err := bundle.LoadJSONFile(lang, file); err != nil {
					fmt.Printf("failed to load JSON file %s: %v\n", file, err)
					continue
				}
			}
		}
	} else {
		// 从 embed FS 加载
		files, err := webhookLocalesEmbed.ReadDir("locales")
		if err != nil {
			return fmt.Errorf("failed to read embed locales: %w", err)
		}

		for _, file := range files {
			fileName := file.Name()
			ext := strings.ToLower(filepath.Ext(fileName))
			if ext != ".yaml" && ext != ".yml" && ext != ".json" {
				continue
			}

			langCode := strings.TrimSuffix(fileName, ext)
			verified := fn.GetVerifiedLocalCode(langCode)
			if verified == "" {
				continue
			}

			lang, ok := i18nkit.ParseLanguage(verified)
			if !ok {
				lang = i18nkit.LangEN
			}

			data, err := webhookLocalesEmbed.ReadFile("locales/" + fileName)
			if err != nil {
				fmt.Printf("failed to read embed file %s: %v\n", fileName, err)
				continue
			}

			switch ext {
			case ".yaml", ".yml":
				if err := bundle.LoadYAML(lang, data); err != nil {
					fmt.Printf("failed to load YAML from embed %s: %v\n", fileName, err)
					continue
				}
			case ".json":
				if err := bundle.LoadJSON(lang, data); err != nil {
					fmt.Printf("failed to load JSON from embed %s: %v\n", fileName, err)
					continue
				}
			}
		}
	}

	GLOBAL_BUNDLE = bundle
	return nil
}

// SetGlobalLocale 设置全局语言
func SetGlobalLocale(lang string) {
	GLOBAL_LANG = lang
}

// GetMessage 获取翻译消息
func GetMessage(messageID string) string {
	if GLOBAL_BUNDLE == nil {
		return messageID
	}

	// 解析语言
	lang, ok := i18nkit.ParseLanguage(GLOBAL_LANG)
	if !ok {
		lang = i18nkit.LangEN // 默认使用英文
	}

	return GLOBAL_BUNDLE.GetTranslation(lang, messageID)
}

// Println 打印翻译消息
func Println(messageID string, a ...any) {
	if len(a) == 0 {
		fmt.Println(GetMessage(messageID))
	} else {
		args := []any{GetMessage(messageID)}
		args = append(args, a...)
		fmt.Println(args...)
	}
}

// Sprintf 格式化翻译消息
func Sprintf(messageID string, a ...any) string {
	return fmt.Sprintf(GetMessage(messageID), a...)
}

// 规范消息键常量（与 locale 文件 key 一致，统一为 MSG_* / ERR_*）
const (
	MSG_WEBHOOK_VERSION               = "MSG_WEBHOOK_VERSION"
	MSG_SERVER_IS_STARTING            = "MSG_SERVER_IS_STARTING"
	MSG_CONFIG_VALIDATION_PASSED      = "MSG_CONFIG_VALIDATION_PASSED"
	MSG_CONFIG_VALIDATION_FAILED      = "MSG_CONFIG_VALIDATION_FAILED"
	MSG_SETUID_OR_SETGID_ERROR        = "MSG_SETUID_OR_SETGID_ERROR"
	ERR_SERVER_LISTENING_PORT         = "ERR_SERVER_LISTENING_PORT"
	ERR_SERVER_LISTENING_PRIVILEGES   = "ERR_SERVER_LISTENING_PRIVILEGES"
	ERR_SERVER_OPENING_LOG_FILE       = "ERR_SERVER_OPENING_LOG_FILE"
	ERR_CREATING_PID_FILE             = "ERR_CREATING_PID_FILE"
	ERR_COULD_NOT_LOAD_ANY_HOOKS      = "ERR_COULD_NOT_LOAD_ANY_HOOKS"
	ERR_VALIDATE_INVALID_PORT         = "ERR_VALIDATE_INVALID_PORT"
	ERR_VALIDATE_DIR_NOT_EXIST        = "ERR_VALIDATE_DIR_NOT_EXIST"
	ERR_VALIDATE_DIR_ACCESS_ERROR     = "ERR_VALIDATE_DIR_ACCESS_ERROR"
	ERR_VALIDATE_DIR_NOT_WRITABLE     = "ERR_VALIDATE_DIR_NOT_WRITABLE"
	ERR_VALIDATE_NOT_DIRECTORY        = "ERR_VALIDATE_NOT_DIRECTORY"
	ERR_VALIDATE_FILE_NOT_EXIST       = "ERR_VALIDATE_FILE_NOT_EXIST"
	ERR_VALIDATE_FILE_ACCESS_ERROR    = "ERR_VALIDATE_FILE_ACCESS_ERROR"
	ERR_VALIDATE_NOT_FILE             = "ERR_VALIDATE_NOT_FILE"
	ERR_VALIDATE_FILE_NOT_READABLE    = "ERR_VALIDATE_FILE_NOT_READABLE"
	ERR_VALIDATE_INVALID_TIMEOUT      = "ERR_VALIDATE_INVALID_TIMEOUT"
	ERR_VALIDATE_TIMEOUT_LOGIC        = "ERR_VALIDATE_TIMEOUT_LOGIC"
	ERR_VALIDATE_INVALID_RATE_LIMIT   = "ERR_VALIDATE_INVALID_RATE_LIMIT"
	ERR_VALIDATE_INVALID_POSITIVE_INT = "ERR_VALIDATE_INVALID_POSITIVE_INT"
	ERR_VALIDATE_HOOK_FILE_LOAD_ERROR = "ERR_VALIDATE_HOOK_FILE_LOAD_ERROR"
	ERR_VALIDATE_HOOK_ID_EMPTY        = "ERR_VALIDATE_HOOK_ID_EMPTY"
	ERR_VALIDATE_HOOK_ID_DUPLICATE    = "ERR_VALIDATE_HOOK_ID_DUPLICATE"
)
