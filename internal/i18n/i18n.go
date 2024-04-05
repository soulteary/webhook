package i18n

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/soulteary/webhook/internal/fn"
	"golang.org/x/text/language"
)

type WebHookLocales struct {
	FileName string
	Name     string
	Content  []byte
}

// get alive locales
func LoadLocaleFiles(localesDir string, webhookLocalesEmbed embed.FS) (aliveLocales []WebHookLocales) {
	localesFiles := fn.ScanDirByExt(localesDir, ".toml")
	// when no locales files found, use the embed locales files
	if len(localesFiles) == 0 {
		files, err := webhookLocalesEmbed.ReadDir("locales")
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range files {
			fileName := file.Name()
			data, err := webhookLocalesEmbed.ReadFile("locales/" + fileName)
			if err != nil {
				fmt.Println(file, err)
				continue
			}
			locales, err := GetWebHookLocaleObject(fileName, data)
			if err != nil {
				fmt.Println(file, err)
				continue
			}
			aliveLocales = append(aliveLocales, locales)
		}
		return aliveLocales
	}

	for _, file := range localesFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Println(file, err)
			continue
		}

		locales, err := GetWebHookLocaleObject(file, content)
		if err != nil {
			fmt.Println(file, err)
			continue
		}
		aliveLocales = append(aliveLocales, locales)
	}
	return aliveLocales
}

func GetWebHookLocaleObject(fileName string, content []byte) (result WebHookLocales, err error) {
	localeNameFromFile := strings.Replace(filepath.Base(fileName), ".toml", "", -1)
	verified := fn.GetVerifiedLocalCode(localeNameFromFile)

	if verified == "" {
		return result, fmt.Errorf("invalid locale name")
	}

	return WebHookLocales{
		FileName: fileName,
		Name:     localeNameFromFile,
		Content:  content,
	}, nil
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

func InitLocaleByFiles(aliveLocales []WebHookLocales) (bundleMaps map[string]WebHookLocalizer) {
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

func Println(messageID string, a ...any) {
	if len(a) == 0 {
		fmt.Println(GetMessage(messageID))
	} else {
		fmt.Println(GetMessage(messageID), a)
	}
}

func Sprintf(messageID string, a ...any) string {
	return fmt.Sprintf(GetMessage(messageID), a)
}
