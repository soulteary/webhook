package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/require"
)

func TestProviderRecipesEndToEnd(t *testing.T) {
	const secret = "ci-provider-secret"
	tests := []struct {
		name       string
		hookID     string
		secretEnv  string
		headers    map[string]string
		hmacHeader string
	}{
		{name: "github", hookID: "github-push", secretEnv: "GITHUB_WEBHOOK_SECRET", headers: map[string]string{"X-GitHub-Event": "push"}, hmacHeader: "X-Hub-Signature-256"},
		{name: "gitlab", hookID: "gitlab-push", secretEnv: "GITLAB_WEBHOOK_TOKEN", headers: map[string]string{"X-Gitlab-Event": "Push Hook", "X-Gitlab-Token": secret}},
		{name: "gitea", hookID: "gitea-push", secretEnv: "GITEA_WEBHOOK_SECRET", headers: map[string]string{"X-Gitea-Event": "push"}, hmacHeader: "X-Gitea-Signature"},
		{name: "harbor", hookID: "harbor-push", secretEnv: "HARBOR_WEBHOOK_TOKEN", headers: map[string]string{"Authorization": "Bearer " + secret}},
		{name: "alertmanager", hookID: "alertmanager-firing", secretEnv: "ALERTMANAGER_WEBHOOK_TOKEN", headers: map[string]string{"Authorization": "Bearer " + secret}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.secretEnv, secret)
			base := filepath.Join("..", "..", "example", "providers", tt.name)
			payload, err := os.ReadFile(filepath.Join(base, "request.json"))
			require.NoError(t, err)
			var configuredHooks hook.Hooks
			require.NoError(t, configuredHooks.LoadFromFileStrict(filepath.Join(base, "hooks.yaml"), true))
			require.Len(t, configuredHooks, 1)

			rules.LoadedHooksFromFiles = map[string]hook.Hooks{tt.name: configuredHooks}
			rules.BuildIndex()
			handler := createHookHandler(flags.AppFlags{MaxRequestBodySize: 1024 * 1024}, nil)
			app := testHookApp(handler)

			headers := make(map[string]string, len(tt.headers)+1)
			for key, value := range tt.headers {
				headers[key] = value
			}
			if tt.hmacHeader != "" {
				headers[tt.hmacHeader] = hmacSHA256(payload, secret, tt.name == "github")
			}
			valid := providerRequest(t, app, tt.hookID, payload, headers)
			require.Equal(t, http.StatusOK, valid.StatusCode)
			validBody, err := io.ReadAll(valid.Body)
			require.NoError(t, err)
			require.NoError(t, valid.Body.Close())
			require.Contains(t, string(validBody), "accepted")

			if tt.hmacHeader != "" {
				headers[tt.hmacHeader] = "sha256=invalid"
			} else {
				headers["Authorization"] = "Bearer invalid"
				if tt.name == "gitlab" {
					headers["X-Gitlab-Token"] = "invalid"
				}
			}
			rejected := providerRequest(t, app, tt.hookID, payload, headers)
			require.NotEqual(t, http.StatusOK, rejected.StatusCode)
			require.NoError(t, rejected.Body.Close())
		})
	}
}

func TestExampleTemplateSecretsRoundTrip(t *testing.T) {
	const secret = "quote='\" newline=\n slash=\\"
	tests := []struct {
		name      string
		path      string
		secretEnv string
		prefix    string
		usesValue bool
	}{
		{name: "github", path: filepath.Join("providers", "github", "hooks.yaml"), secretEnv: "GITHUB_WEBHOOK_SECRET"},
		{name: "gitlab", path: filepath.Join("providers", "gitlab", "hooks.yaml"), secretEnv: "GITLAB_WEBHOOK_TOKEN", usesValue: true},
		{name: "gitea", path: filepath.Join("providers", "gitea", "hooks.yaml"), secretEnv: "GITEA_WEBHOOK_SECRET"},
		{name: "harbor", path: filepath.Join("providers", "harbor", "hooks.yaml"), secretEnv: "HARBOR_WEBHOOK_TOKEN", prefix: "Bearer ", usesValue: true},
		{name: "alertmanager", path: filepath.Join("providers", "alertmanager", "hooks.yaml"), secretEnv: "ALERTMANAGER_WEBHOOK_TOKEN", prefix: "Bearer ", usesValue: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.secretEnv, secret)
			var configuredHooks hook.Hooks
			path := filepath.Join("..", "..", "example", tt.path)
			require.NoError(t, configuredHooks.LoadFromFileStrict(path, true))
			require.Len(t, configuredHooks, 1)
			require.NotNil(t, configuredHooks[0].TriggerRule)

			rule := configuredHooks[0].TriggerRule
			if rule.And != nil {
				require.NotEmpty(t, *rule.And)
				rule = &(*rule.And)[0]
			}
			require.NotNil(t, rule.Match)
			if tt.usesValue {
				require.Equal(t, tt.prefix+secret, rule.Match.Value)
			} else {
				require.Equal(t, secret, rule.Match.Secret)
			}
		})
	}
}

func TestExampleTemplatesRejectMissingSecrets(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		secretEnv string
	}{
		{name: "github", path: filepath.Join("providers", "github", "hooks.yaml"), secretEnv: "GITHUB_WEBHOOK_SECRET"},
		{name: "gitlab", path: filepath.Join("providers", "gitlab", "hooks.yaml"), secretEnv: "GITLAB_WEBHOOK_TOKEN"},
		{name: "gitea", path: filepath.Join("providers", "gitea", "hooks.yaml"), secretEnv: "GITEA_WEBHOOK_SECRET"},
		{name: "harbor", path: filepath.Join("providers", "harbor", "hooks.yaml"), secretEnv: "HARBOR_WEBHOOK_TOKEN"},
		{name: "alertmanager", path: filepath.Join("providers", "alertmanager", "hooks.yaml"), secretEnv: "ALERTMANAGER_WEBHOOK_TOKEN"},
		{name: "quickstart", path: filepath.Join("quickstart", "hooks", "hooks.yaml"), secretEnv: "DEMO_SECRET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.secretEnv, "")
			var configuredHooks hook.Hooks
			path := filepath.Join("..", "..", "example", tt.path)
			err := configuredHooks.LoadFromFileStrict(path, true)
			require.ErrorContains(t, err, "required environment variable "+tt.secretEnv+" is empty")
		})
	}
}

func providerRequest(t *testing.T, app *fiber.App, hookID string, payload []byte, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/hooks/"+hookID, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	require.NoError(t, err)
	return resp
}

func hmacSHA256(payload []byte, secret string, withPrefix bool) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	if withPrefix {
		return "sha256=" + signature
	}
	return signature
}
