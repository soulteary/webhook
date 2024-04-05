package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/soulteary/webhook/internal/fn"
	"golang.org/x/text/language"
)

// get verified local code
func getVerifiedLocalCode(targetCode string) string {
	var tag language.Tag
	err := tag.UnmarshalText([]byte(targetCode))
	if err != nil {
		return ""
	}
	b, err := tag.MarshalText()
	if err != nil {
		return ""
	}

	verified := string(b)
	if verified != targetCode {
		return ""
	}
	return verified
}

type WebHookLocals struct {
	FileName string
	Name     string
	Content  []byte
}

// get alive locales
func LoadLocaleFiles(localesDir string) (aliveLocales []WebHookLocals) {
	// TODO custom locales directory
	localesFiles := fn.ScanDirByExt(localesDir, ".toml")
	if len(localesFiles) == 0 {
		return
	}

	for _, file := range localesFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Println(file, err)
			continue
		}

		localeNameFromFile := strings.Replace(filepath.Base(file), ".toml", "", -1)
		verified := getVerifiedLocalCode(localeNameFromFile)
		if verified != "" {
			aliveLocales = append(aliveLocales, WebHookLocals{
				FileName: file,
				Name:     localeNameFromFile,
				Content:  content,
			})
		}
	}
	return aliveLocales
}

type WebHookLocalizer struct {
	FileName  string
	Name      string
	Bundle    *i18n.Bundle
	Localizer *i18n.Localizer
}

var GLOBAL_LOCALES map[string]WebHookLocalizer
var GLOBAL_LANG string

func SetGlobalLocale(lang string) {
	GLOBAL_LANG = lang
}

func InitLocaleByFiles(aliveLocales []WebHookLocals) (bundleMaps map[string]WebHookLocalizer) {
	bundleMaps = make(map[string]WebHookLocalizer)
	for _, locale := range aliveLocales {
		bundle := i18n.NewBundle(language.MustParse(locale.Name))
		bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
		bundle.MustParseMessageFileBytes(locale.Content, locale.FileName)
		bundleMaps[locale.Name] = WebHookLocalizer{
			FileName:  locale.FileName,
			Name:      locale.Name,
			Bundle:    bundle,
			Localizer: i18n.NewLocalizer(bundle, locale.Name),
		}
	}
	return bundleMaps
}

func GetMessage(messageID string) string {
	locale := GLOBAL_LANG
	localizer, ok := GLOBAL_LOCALES[locale]
	if !ok {
		return fmt.Sprintf("locale %s not found", locale)
	}
	return localizer.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: messageID})
}
