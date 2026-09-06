package hook

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	defaultSecretDirectory = "/run/secrets"
	templateDirectoryEnv   = "WEBHOOK_SECRET_DIR"
	maxSecretFileSize      = int64(1 << 20)
)

func templateFunctions() template.FuncMap {
	return template.FuncMap{
		"getenv":         getenv,
		"getenvRequired": getenvRequired,
		"getSecret":      getSecret,
		"trimSpace":      strings.TrimSpace,
	}
}

// getSecret reads one named secret from the configured secret directory.
// Names, rather than arbitrary paths, keep template file access confined to
// the directory supplied by the operator.
func getSecret(name string) (string, error) {
	if name == "" || name == "." || name == ".." || filepath.IsAbs(name) ||
		strings.ContainsAny(name, `/\`) || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid secret name %q: use a single file name", name)
	}

	secretDirectory := os.Getenv(templateDirectoryEnv)
	if secretDirectory == "" {
		secretDirectory = defaultSecretDirectory
	}
	if !filepath.IsAbs(secretDirectory) {
		return "", fmt.Errorf("%s must be an absolute path", templateDirectoryEnv)
	}

	secretRoot, err := os.OpenRoot(filepath.Clean(secretDirectory))
	if err != nil {
		return "", fmt.Errorf("open secret directory: %w", err)
	}
	defer secretRoot.Close()

	secretFile, err := secretRoot.Open(name)
	if err != nil {
		return "", fmt.Errorf("open secret %q: %w", name, err)
	}
	defer secretFile.Close()

	secretInfo, err := secretFile.Stat()
	if err != nil {
		return "", fmt.Errorf("stat secret %q: %w", name, err)
	}
	if !secretInfo.Mode().IsRegular() {
		return "", fmt.Errorf("secret %q is not a regular file", name)
	}
	if secretInfo.Size() > maxSecretFileSize {
		return "", fmt.Errorf("secret %q exceeds the %d-byte size limit", name, maxSecretFileSize)
	}

	contents, err := io.ReadAll(io.LimitReader(secretFile, maxSecretFileSize+1))
	if err != nil {
		return "", fmt.Errorf("read secret %q: %w", name, err)
	}
	if int64(len(contents)) > maxSecretFileSize {
		return "", fmt.Errorf("secret %q exceeds the %d-byte size limit", name, maxSecretFileSize)
	}

	return string(contents), nil
}
