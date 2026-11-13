package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func requestIDSeenBy(t *testing.T, useHeader bool, limit int, sent string) string {
	t.Helper()
	app := fiber.New()
	app.Use(requestIDPolicy(useHeader, limit))
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(c.Get("X-Request-Id")) })
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", sent)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n])
}

func TestRequestIDPolicy(t *testing.T) {
	if got := requestIDSeenBy(t, false, 0, "client-supplied"); got != "" {
		t.Errorf("client request ID kept while -x-request-id is off: %q", got)
	}
	if got := requestIDSeenBy(t, true, 0, "client-supplied"); got != "client-supplied" {
		t.Errorf("request ID changed with no limit: %q", got)
	}
	if got := requestIDSeenBy(t, true, 6, "client-supplied"); got != "client" {
		t.Errorf("request ID not truncated to limit: %q", got)
	}
}
