package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signSHA256(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// multipart 请求的 HMAC 必须覆盖真实请求体：空 body 的签名不能用来重放任意表单参数。
func TestCreateHookHandler_MultipartSignatureCoversBody(t *testing.T) {
	const secret = "topsecret"
	testHook := hook.Hook{
		ID:              "signed-hook",
		ResponseMessage: "signed-ok",
		TriggerRule: &hook.Rules{Match: &hook.MatchRule{
			Type:      hook.MatchHMACSHA256,
			Secret:    secret,
			Parameter: hook.Argument{Source: "header", Name: "X-Signature"},
		}},
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{"test.json": {testHook}}
	rules.BuildIndex()
	handler := createHookHandler(flags.AppFlags{MaxMultipartMem: 1024 * 1024}, nil)

	send := func(signature func(body []byte) string) string {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("key", "value")
		_ = writer.Close()
		raw := body.Bytes()

		req := httptest.NewRequest("POST", "/hooks/signed-hook", bytes.NewReader(raw))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-Signature", signature(raw))
		resp, err := testHookApp(handler).Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		out, _ := io.ReadAll(resp.Body)
		return string(out)
	}

	assert.Contains(t, send(func(body []byte) string { return signSHA256(secret, body) }), "signed-ok",
		"signature over the real multipart body must be accepted")
	assert.NotContains(t, send(func([]byte) string { return signSHA256(secret, nil) }), "signed-ok",
		"signature over an empty body must not authorize a multipart request")
}
