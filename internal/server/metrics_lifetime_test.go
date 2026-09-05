package server

import (
	"context"
	"io"
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

func TestMetricsRemainCollectableAfterAdaptedRequests(t *testing.T) {
	appFlags := flags.AppFlags{
		HooksURLPrefix:  "/hooks",
		ResponseHeaders: hook.ResponseHeaders{},
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	server := Launch(appFlags, ln.Addr().String(), ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := "http://" + ln.Addr().String()
	require.Eventually(t, func() bool {
		resp, requestErr := client.Get(baseURL + "/")
		if requestErr != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, time.Second, 10*time.Millisecond)

	for _, path := range []string{"/health", "/livez", "/readyz", "/version"} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions} {
			req, requestErr := http.NewRequest(method, baseURL+path, nil)
			require.NoError(t, requestErr)
			resp, requestErr := client.Do(req)
			require.NoError(t, requestErr)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}

	resp, err := client.Get(baseURL + "/metrics")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	assert.NotContains(t, strings.ToLower(string(body)), "error has occurred")
}
