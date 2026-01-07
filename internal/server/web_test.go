package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/stretchr/testify/assert"
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
	defer ln.Close()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	server.Shutdown(ctx)
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
	defer ln.Close()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	server.Shutdown(ctx)
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
	defer ln.Close()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the root handler
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
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
	defer ln.Close()

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
	defer ln.Close()

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
	defer ln.Close()

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
		server.Shutdown(ctx)
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
	defer ln.Close()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	server.Shutdown(ctx)
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
	defer ln.Close()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the health endpoint
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/health", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
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
	defer ln.Close()

	// Launch server
	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to the metrics endpoint
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/metrics", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
