package i18n_test

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/webhook/internal/i18n"
	"github.com/stretchr/testify/assert"
)

var embedFS embed.FS

func createYAMLFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0o644)
	assert.NoError(t, err)
}

func TestInitLocaleByFiles_FromDir(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
MSG_WEBHOOK_VERSION: "webhook version "
`)
	createYAMLFile(t, tempDir, "zh-CN.yaml", `
MSG_WEBHOOK_VERSION: "网页钩子版本 "
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	assert.NotNil(t, i18n.GLOBAL_BUNDLE)
}

func TestSetGlobalLocale(t *testing.T) {
	i18n.SetGlobalLocale("en-US")
	assert.Equal(t, "en-US", i18n.GLOBAL_LANG)

	i18n.SetGlobalLocale("zh-CN")
	assert.Equal(t, "zh-CN", i18n.GLOBAL_LANG)
}

func TestGetMessage(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
MSG_WEBHOOK_VERSION: "webhook version "
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	i18n.SetGlobalLocale("en-US")

	message := i18n.GetMessage("MSG_WEBHOOK_VERSION")
	assert.Contains(t, message, "webhook version")

	// Non-existent message ID returns the ID
	message = i18n.GetMessage("NON_EXISTENT_KEY")
	assert.Equal(t, "NON_EXISTENT_KEY", message)
}

func TestPrintln(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
MSG_WEBHOOK_VERSION: "webhook version "
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	i18n.SetGlobalLocale("en-US")

	// Test Println without arguments (just ensure it doesn't panic)
	i18n.Println("MSG_WEBHOOK_VERSION")

	// Test Println with arguments
	i18n.Println("MSG_WEBHOOK_VERSION", "test")
}

func TestSprintf(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
SERVER_IS_STARTING: "version %s starting"
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	i18n.SetGlobalLocale("en-US")

	result := i18n.Sprintf("SERVER_IS_STARTING", "1.0.0")
	assert.Contains(t, result, "version")
	assert.Contains(t, result, "starting")
}

func TestInitLocaleByFiles_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	// Empty directory, empty embedFS -> will try embed "locales" and fail
	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read embed locales")
}

func TestInitLocaleByFiles_MultipleLocales(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
TEST: "English"
`)
	createYAMLFile(t, tempDir, "zh-CN.yaml", `
TEST: "中文"
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	assert.NotNil(t, i18n.GLOBAL_BUNDLE)

	i18n.SetGlobalLocale("en-US")
	msg := i18n.GetMessage("TEST")
	assert.Contains(t, msg, "English")

	i18n.SetGlobalLocale("zh-CN")
	msg = i18n.GetMessage("TEST")
	assert.Contains(t, msg, "中文")
}

func TestGetMessage_WithArgs(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
TEST_MESSAGE: "Hello %s"
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	i18n.SetGlobalLocale("en-US")

	message := i18n.GetMessage("TEST_MESSAGE")
	assert.Contains(t, message, "Hello")
}

func TestSprintf_WithMultipleArgs(t *testing.T) {
	tempDir := t.TempDir()

	createYAMLFile(t, tempDir, "en-US.yaml", `
FORMAT_MESSAGE: "Version %s started on %s"
`)

	err := i18n.InitLocaleByFiles(tempDir, embedFS)
	assert.NoError(t, err)
	i18n.SetGlobalLocale("en-US")

	result := i18n.Sprintf("FORMAT_MESSAGE", "1.0.0", "localhost")
	assert.Contains(t, result, "Version")
	assert.Contains(t, result, "started")
}
