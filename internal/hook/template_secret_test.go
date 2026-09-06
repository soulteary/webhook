package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSecretReadsNamedFileWithoutChangingContents(t *testing.T) {
	secretDirectory := t.TempDir()
	t.Setenv(templateDirectoryEnv, secretDirectory)
	require.NoError(t, os.WriteFile(filepath.Join(secretDirectory, "webhook-token"), []byte("secret-value\n"), 0o600))

	value, err := getSecret("webhook-token")
	require.NoError(t, err)
	assert.Equal(t, "secret-value\n", value)
}

func TestGetSecretRejectsInvalidNames(t *testing.T) {
	t.Setenv(templateDirectoryEnv, t.TempDir())

	for _, name := range []string{"", ".", "..", "../secret", "nested/secret", `nested\secret`, "/etc/passwd"} {
		t.Run(name, func(t *testing.T) {
			_, err := getSecret(name)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid secret name")
		})
	}
}

func TestGetSecretRequiresAbsoluteDirectory(t *testing.T) {
	t.Setenv(templateDirectoryEnv, "relative/secrets")

	_, err := getSecret("webhook-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), templateDirectoryEnv+" must be an absolute path")
}

func TestGetSecretRejectsPathEscapingSymlink(t *testing.T) {
	secretDirectory := t.TempDir()
	t.Setenv(templateDirectoryEnv, secretDirectory)
	outsideFile := filepath.Join(t.TempDir(), "outside-secret")
	require.NoError(t, os.WriteFile(outsideFile, []byte("must-not-be-read"), 0o600))
	if err := os.Symlink(outsideFile, filepath.Join(secretDirectory, "webhook-token")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	_, err := getSecret("webhook-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `open secret "webhook-token"`)
	assert.NotContains(t, err.Error(), "must-not-be-read")
}

func TestGetSecretRejectsNonRegularAndOversizedFiles(t *testing.T) {
	secretDirectory := t.TempDir()
	t.Setenv(templateDirectoryEnv, secretDirectory)
	require.NoError(t, os.Mkdir(filepath.Join(secretDirectory, "directory"), 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(secretDirectory, "oversized"),
		bytes.Repeat([]byte("x"), int(maxSecretFileSize)+1),
		0o600,
	))

	_, err := getSecret("directory")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a regular file")

	_, err = getSecret("oversized")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the")
}

func TestGetSecretReportsMissingFileWithoutLeakingContents(t *testing.T) {
	secretDirectory := t.TempDir()
	t.Setenv(templateDirectoryEnv, secretDirectory)

	_, err := getSecret("missing-secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `secret "missing-secret"`)
}

func TestHooksTemplateLoadsSecretFile(t *testing.T) {
	secretDirectory := t.TempDir()
	t.Setenv(templateDirectoryEnv, secretDirectory)
	require.NoError(t, os.WriteFile(filepath.Join(secretDirectory, "webhook-token"), []byte("secret-value\n"), 0o600))

	hooksPath := filepath.Join(t.TempDir(), "hooks.json.tmpl")
	templateConfig := `[{"id":"secret-template","execute-command":"/bin/echo","response-message":"{{ getSecret "webhook-token" | trimSpace | js }}"}]`
	require.NoError(t, os.WriteFile(hooksPath, []byte(templateConfig), 0o600))

	var hooks Hooks
	require.NoError(t, hooks.LoadFromFile(hooksPath, true))
	require.Len(t, hooks, 1)
	assert.Equal(t, "secret-value", hooks[0].ResponseMessage)
}

func TestHooksTemplateFailsClosedWhenSecretIsMissing(t *testing.T) {
	t.Setenv(templateDirectoryEnv, t.TempDir())
	hooksPath := filepath.Join(t.TempDir(), "hooks.json.tmpl")
	templateConfig := `[{"id":"secret-template","execute-command":"/bin/echo","response-message":"{{ getSecret "missing-secret" }}"}]`
	require.NoError(t, os.WriteFile(hooksPath, []byte(templateConfig), 0o600))

	var hooks Hooks
	err := hooks.LoadFromFile(hooksPath, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-secret")
}
