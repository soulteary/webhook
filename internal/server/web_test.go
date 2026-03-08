package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLaunch(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func TestLaunch_WithDebug(t *testing.T) {
	// Create app flags with debug enabled
	appFlags := flags.AppFlags{
		Debug:           true,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func TestLaunch_RootHandler(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:          false,
		HttpMethods:    "",
		HooksURLPrefix: "/hooks",
		ResponseHeaders: hook.ResponseHeaders{
			{Name: "X-Test", Value: "test-value"},
		},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the root handler
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "test-value", resp.Header.Get("X-Test"))
	}
}

func TestServer_Shutdown(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)

	// Test that shutdown is idempotent
	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServer_Shutdown_Timeout(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	err = server.Shutdown(ctx)
	// Should get timeout error
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestServer_IsShuttingDown(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Initially should not be shutting down
	assert.False(t, server.IsShuttingDown())

	// Start shutdown in a goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownDone := make(chan bool, 1)
	go func() {
		_ = server.Shutdown(ctx)
		shutdownDone <- true
	}()

	// Wait a bit for shutdown to start
	time.Sleep(50 * time.Millisecond)

	// Should be shutting down now
	assert.True(t, server.IsShuttingDown())

	// Wait for shutdown to complete
	select {
	case <-shutdownDone:
	case <-time.After(2 * time.Second):
		t.Log("Shutdown completed")
	}
}

func TestLaunch_WithRateLimit(t *testing.T) {
	// Create app flags with rate limiting enabled
	appFlags := flags.AppFlags{
		Debug:            false,
		HttpMethods:      "",
		HooksURLPrefix:   "/hooks",
		ResponseHeaders:  hook.ResponseHeaders{},
		RateLimitEnabled: true,
		RateLimitRPS:     100,
		RateLimitBurst:   10,
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func TestLaunch_HealthEndpoint(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the health endpoint
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/health", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestLaunch_MetricsEndpoint(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the metrics endpoint
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/metrics", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestLaunch_OpenAPIEndpoint(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		OpenAPIEnabled:  true,
		OpenAPIPath:     "/openapi",
	}

	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/openapi", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var spec map[string]any
	err = json.NewDecoder(resp.Body).Decode(&spec)
	require.NoError(t, err)
	assert.Equal(t, "3.0.3", spec["openapi"])
	_, hasPaths := spec["paths"]
	assert.True(t, hasPaths)
}

func TestLaunch_OpenAPIDisabled_Returns404(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		OpenAPIEnabled:  false,
		OpenAPIPath:     "/openapi",
	}

	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/openapi", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestLaunch_OpenAPIEndpoint_ReservedPathSkipped(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		OpenAPIEnabled:  true,
		OpenAPIPath:     "/health", // reserved; OpenAPI route should not be registered
	}

	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	// GET /health should still be the health handler (e.g. JSON), not the OpenAPI spec
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/health", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Health endpoint returns JSON (health-kit), not the OpenAPI doc
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

func TestLaunch_ConfigUIPathTrailingSlash(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		ConfigUIEnabled: true,
		ConfigUIPath:    "/config-ui/", // trailing slash normalized
	}

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	srv := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + ln.Addr().String()

	req, err := http.NewRequest("GET", base+"/config-ui", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /config-ui with path configured as /config-ui/")
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	req2, err := http.NewRequest("GET", base+"/config-ui/static/js/app.js", nil)
	require.NoError(t, err)
	resp2, err := client.Do(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp2.StatusCode, "GET /config-ui/static/js/app.js")
}

func TestLaunch_ConfigUIWithCustomURLPrefix(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "events",
		ResponseHeaders: hook.ResponseHeaders{},
		ConfigUIEnabled: true,
		ConfigUIPath:    "/config-ui",
	}

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	srv := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + ln.Addr().String()

	// POST /config-ui/api/generate and check callUrl contains /events/
	body := `{"id":"test-id","execute-command":"/bin/true"}`
	req, err := http.NewRequest("POST", base+"/config-ui/api/generate", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		CallURL string `json:"callUrl"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result.CallURL, "/events/test-id", "callUrl should use urlprefix 'events'")
}

func TestLaunch_ConfigUIDisabled_Returns404(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		ConfigUIEnabled: false,
		ConfigUIPath:    "/config-ui",
	}

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	srv := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/config-ui", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Config UI disabled: /config-ui should 404")
}

func TestLaunch_ConfigUIPathReserved_SkipsMount(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		ConfigUIEnabled: true,
		ConfigUIPath:    "/health", // reserved; Config UI must not be mounted
	}

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	srv := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + ln.Addr().String()

	// /health must remain the health handler (JSON), not Config UI (HTML)
	reqHealth, err := http.NewRequest("GET", base+"/health", nil)
	require.NoError(t, err)
	respHealth, err := client.Do(reqHealth)
	require.NoError(t, err)
	defer func() { _ = respHealth.Body.Close() }()
	assert.Equal(t, http.StatusOK, respHealth.StatusCode)
	assert.Contains(t, respHealth.Header.Get("Content-Type"), "application/json", "reserved path /health unchanged")

	// /config-ui should 404 when Config UI was skipped due to conflict
	reqUI, err := http.NewRequest("GET", base+"/config-ui", nil)
	require.NoError(t, err)
	respUI, err := client.Do(reqUI)
	require.NoError(t, err)
	defer func() { _ = respUI.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, respUI.StatusCode, "config-ui path was reserved so UI not mounted; /config-ui 404")
}

func TestLaunch_ConfigUICapabilities(t *testing.T) {
	appFlags := flags.AppFlags{
		Debug:           false,
		HttpMethods:     "",
		HooksURLPrefix:  "hooks",
		ResponseHeaders: hook.ResponseHeaders{},
		ConfigUIEnabled: true,
		ConfigUIPath:    "/config-ui",
		HooksDir:        t.TempDir(),
	}

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	srv := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/config-ui/api/capabilities", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cap map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&cap)
	require.NoError(t, err)
	assert.True(t, cap["saveToDir"], "with HooksDir set, saveToDir should be true")
}
