package owlmail_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestHookTemplateRendersValidJSON(t *testing.T) {
	data, err := os.ReadFile("hooks.json.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	values := map[string]string{
		"OWLMAIL_WEBHOOK_SECRET":  "test-secret",
		"OWLMAIL_WEBHOOK_COMMAND": "/app/print-email.sh",
		"OWLMAIL_WEBHOOK_WORKDIR": "/app",
	}
	tmpl, err := template.New("hooks").Funcs(template.FuncMap{
		"getenv": func(name string) string { return values[name] },
	}).Parse(string(data))
	if err != nil {
		t.Fatalf("parse hook template: %v", err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, nil); err != nil {
		t.Fatalf("render hook template: %v", err)
	}

	var hooks []map[string]any
	if err := json.Unmarshal(rendered.Bytes(), &hooks); err != nil {
		t.Fatalf("rendered hook config is invalid JSON: %v\n%s", err, rendered.String())
	}
	if len(hooks) != 1 || hooks[0]["id"] != "owlmail" {
		t.Fatalf("unexpected hooks: %#v", hooks)
	}

	rule := hooks[0]["trigger-rule"].(map[string]any)["match"].(map[string]any)
	if rule["type"] != "payload-hmac-sha256" || rule["secret"] != "test-secret" {
		t.Fatalf("unexpected HMAC rule: %#v", rule)
	}
}

func TestOwlMailConfigAndDocumentation(t *testing.T) {
	data, err := os.ReadFile("owlmail.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Version int `json:"version"`
		Targets []struct {
			URL    string `json:"url"`
			Secret string `json:"secret"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid OwlMail config: %v", err)
	}
	if config.Version != 1 || len(config.Targets) != 1 {
		t.Fatalf("unexpected OwlMail config: %#v", config)
	}
	if config.Targets[0].URL != "${OWLMAIL_WEBHOOK_URL}" || config.Targets[0].Secret != "${OWLMAIL_WEBHOOK_SECRET}" {
		t.Fatalf("OwlMail runtime variables are not preserved: %#v", config.Targets[0])
	}

	compose, err := os.ReadFile("compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"soulteary/webhook:extend-7.0.0",
		"soulteary/owlmail:0.5.0",
		"127.0.0.1:9000:9000",
		"127.0.0.1:1025:1025",
		"127.0.0.1:1080:1080",
		"OWLMAIL_WEBHOOK_MAX_CONCURRENCY: \"8\"",
		"OWLMAIL_WEBHOOK_SECRET:?set OWLMAIL_WEBHOOK_SECRET",
		"DEBUG: \"true\"",
		"LOG_REQUEST_BODY: \"false\"",
		"ALLOWED_COMMAND_PATHS",
		"HOOK_EXECUTION_TIMEOUT: \"8\"",
		"STRICT_MODE",
	} {
		if !strings.Contains(string(compose), required) {
			t.Errorf("compose.yaml does not contain %q", required)
		}
	}
	if strings.Contains(string(compose), "github.com/soulteary/owlmail.git#main") {
		t.Error("compose.yaml must use the released OwlMail image instead of an unpinned main build")
	}
	for _, unsafe := range []string{
		`- "9000:9000"`,
		`- "1025:1025"`,
		`- "1080:1080"`,
	} {
		if strings.Contains(string(compose), unsafe) {
			t.Errorf("compose.yaml publishes a demo port on every host interface: %s", unsafe)
		}
	}

	for _, path := range []string{
		"README.md",
		"README.zh-CN.md",
		filepath.Join("..", "..", "docs", "en-US", "OwlMail-Integration.md"),
		filepath.Join("..", "..", "docs", "zh-CN", "OwlMail-Integration.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("linked documentation %q is unavailable: %v", path, err)
		}
	}
}
