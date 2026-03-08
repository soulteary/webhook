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
