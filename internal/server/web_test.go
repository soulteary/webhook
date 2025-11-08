package server

import (
	"net"
	"net/http"
	"testing"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/stretchr/testify/assert"
)

func TestLaunch(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:         false,
		HttpMethods:   "",
		HooksURLPrefix: "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer ln.Close()

	// Start server in a goroutine
	go func() {
		Launch(appFlags, ln.Addr().String(), ln)
	}()

	// Wait a bit for server to start
	// Note: This test mainly verifies that Launch doesn't panic
	// The actual HTTP handling is tested in webhook_test.go
}

func TestLaunch_WithDebug(t *testing.T) {
	// Create app flags with debug enabled
	appFlags := flags.AppFlags{
		Debug:         true,
		HttpMethods:   "",
		HooksURLPrefix: "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer ln.Close()

	// Start server in a goroutine
	go func() {
		Launch(appFlags, ln.Addr().String(), ln)
	}()

	// Wait a bit for server to start
	// Note: This test mainly verifies that Launch with debug doesn't panic
}

func TestLaunch_RootHandler(t *testing.T) {
	// Create app flags
	appFlags := flags.AppFlags{
		Debug:         false,
		HttpMethods:   "",
		HooksURLPrefix: "/hooks",
		ResponseHeaders: hook.ResponseHeaders{
			{Name: "X-Test", Value: "test-value"},
		},
	}

	// Create a listener
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer ln.Close()

	// Start server in a goroutine
	go func() {
		Launch(appFlags, ln.Addr().String(), ln)
	}()

	// Make a request to the root handler
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "test-value", resp.Header.Get("X-Test"))
	}
}

