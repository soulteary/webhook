package i18n_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestLoadLocaleFiles(t *testing.T) {
	tempDir := t.TempDir()

	createTOMLFile(t, tempDir, "en.toml", `
		[webhook]
		title = "Webhook"
		description = "This is a webhook"
	`)
	createTOMLFile(t, tempDir, "zh-CN.toml", `
		[webhook]
		title = "网页钩子"
		description = "这是一个网页钩子"
	`)
	createTOMLFile(t, tempDir, "invalid.toml", `
		invalid content
	`)

	aliveLocales := i18n.LoadLocaleFiles(tempDir)
	assert.Len(t, aliveLocales, 2)

	assert.Equal(t, "en", aliveLocales[0].Name)
	assert.Contains(t, string(aliveLocales[0].Content), "Webhook")
	assert.Equal(t, "zh-CN", aliveLocales[1].Name)
	assert.Contains(t, string(aliveLocales[1].Content), "网页钩子")
}

func createTOMLFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	assert.NoError(t, err)
}
