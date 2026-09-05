package openapi

import (
	"encoding/json"
	"testing"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpec(t *testing.T) {
	appFlags := flags.AppFlags{
		HooksURLPrefix: "hooks",
		HttpMethods:    "POST,PUT",
	}
	out, err := Spec(appFlags, "http://localhost:9000")
	require.NoError(t, err)

	var spec map[string]any
	err = json.Unmarshal(out, &spec)
	require.NoError(t, err)

	assert.Equal(t, "3.0.3", spec["openapi"])
	info, ok := spec["info"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, info["title"])
	assert.NotEmpty(t, info["version"])

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, paths, "/")
	assert.Contains(t, paths, "/health")
	assert.Contains(t, paths, "/livez")
	assert.Contains(t, paths, "/readyz")
	assert.Contains(t, paths, "/version")
	assert.Contains(t, paths, "/metrics")
	assert.Contains(t, paths, "/hooks/{id}")

	versionPath, ok := paths["/version"].(map[string]any)
	require.True(t, ok)
	versionGet, ok := versionPath["get"].(map[string]any)
	require.True(t, ok)
	responses, ok := versionGet["responses"].(map[string]any)
	require.True(t, ok)
	versionOK, ok := responses["200"].(map[string]any)
	require.True(t, ok)
	content, ok := versionOK["content"].(map[string]any)
	require.True(t, ok)
	jsonContent, ok := content["application/json"].(map[string]any)
	require.True(t, ok)
	schema, ok := jsonContent["schema"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", schema["type"])
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "version")
	assert.Contains(t, properties, "commit")
	assert.Contains(t, properties, "build_date")
	assert.NotContains(t, properties, "buildDate")
	assert.Contains(t, properties, "branch")
	assert.Contains(t, properties, "go_version")
	assert.Contains(t, properties, "platform")
	assert.Contains(t, properties, "compiler")
}

func TestSpec_WithServerURL(t *testing.T) {
	appFlags := flags.AppFlags{HooksURLPrefix: "hooks"}
	out, err := Spec(appFlags, "https://example.com")
	require.NoError(t, err)

	var spec map[string]any
	err = json.Unmarshal(out, &spec)
	require.NoError(t, err)

	servers, ok := spec["servers"].([]any)
	require.True(t, ok)
	require.Len(t, servers, 1)
	srv, ok := servers[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://example.com", srv["url"])
}

func TestSpec_DefaultHookMethodsMatchRuntime(t *testing.T) {
	out, err := Spec(flags.AppFlags{HooksURLPrefix: "hooks"}, "")
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(out, &spec))
	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)
	hookPath, ok := paths["/hooks/{id}"].(map[string]any)
	require.True(t, ok)

	for _, method := range []string{"get", "head", "post", "put", "patch", "delete", "options", "trace"} {
		assert.Contains(t, hookPath, method)
	}
	assert.Equal(t, []any{"CONNECT"}, hookPath["x-webhook-additional-methods"])
	assert.Equal(t, true, hookPath["x-webhook-allow-any-method"])
}

func TestSpec_OnlyRoutableNonOpenAPIMethodUsesExtension(t *testing.T) {
	out, err := Spec(flags.AppFlags{HooksURLPrefix: "hooks", HttpMethods: "POST,CONNECT,CUSTOM"}, "")
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(out, &spec))
	paths := spec["paths"].(map[string]any)
	hookPath := paths["/hooks/{id}"].(map[string]any)
	assert.Contains(t, hookPath, "post")
	assert.NotContains(t, hookPath, "connect")
	assert.NotContains(t, hookPath, "custom")
	assert.Equal(t, []any{"CONNECT"}, hookPath["x-webhook-additional-methods"])
}

func TestSpec_DoesNotAdvertiseUnroutableCustomMethod(t *testing.T) {
	out, err := Spec(flags.AppFlags{HooksURLPrefix: "hooks", HttpMethods: "CUSTOM"}, "")
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(out, &spec))
	paths := spec["paths"].(map[string]any)
	hookPath := paths["/hooks/{id}"].(map[string]any)
	assert.NotContains(t, hookPath, "x-webhook-additional-methods")
}

func TestSpec_HookErrorsDescribePlainTextAndAllowHeader(t *testing.T) {
	out, err := Spec(flags.AppFlags{HooksURLPrefix: "hooks", HttpMethods: "POST"}, "")
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(out, &spec))
	paths := spec["paths"].(map[string]any)
	hookPath := paths["/hooks/{id}"].(map[string]any)
	post := hookPath["post"].(map[string]any)
	responses := post["responses"].(map[string]any)

	for _, status := range []string{"400", "404", "405", "408", "429", "500", "503"} {
		response := responses[status].(map[string]any)
		content := response["content"].(map[string]any)
		assert.Contains(t, content, "text/plain")
	}
	methodNotAllowed := responses["405"].(map[string]any)
	headers := methodNotAllowed["headers"].(map[string]any)
	assert.Contains(t, headers, "Allow")
}

func TestSpec_CustomURLPrefix(t *testing.T) {
	appFlags := flags.AppFlags{
		HooksURLPrefix: "api/webhooks",
	}
	out, err := Spec(appFlags, "")
	require.NoError(t, err)

	var spec map[string]any
	err = json.Unmarshal(out, &spec)
	require.NoError(t, err)

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, paths, "/api/webhooks/{id}")
}
