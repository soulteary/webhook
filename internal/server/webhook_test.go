package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
)

func TestStaticParams(t *testing.T) {
	// FIXME(moorereason): incorporate this test into TestWebhook.
	//   Need to be able to execute a binary with a space in the filename.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	spHeaders := make(map[string]interface{})
	spHeaders["User-Agent"] = "curl/7.54.0"
	spHeaders["Accept"] = "*/*"

	// case 2: binary with spaces in its name
	d1 := []byte("#!/bin/sh\n/bin/echo\n")
	err := os.WriteFile("/tmp/with space", d1, 0o755)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer os.Remove("/tmp/with space")

	spHook := &hook.Hook{
		ID:                      "static-params-name-space",
		ExecuteCommand:          "/tmp/with space",
		CommandWorkingDirectory: "/tmp",
		ResponseMessage:         "success",
		CaptureCommandOutput:    true,
		PassArgumentsToCommand: []hook.Argument{
			{Source: "string", Name: "passed"},
		},
	}

	b := &bytes.Buffer{}
	log.SetOutput(b)

	r := &hook.Request{
		ID:      "test",
		Headers: spHeaders,
	}
	_, err = handleHook(spHook, r, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}
	matched, _ := regexp.MatchString("(?s)command output: .*static-params-name-space", b.String())
	if !matched {
		t.Fatalf("Unexpected log output:\n%sn", b)
	}
}

func TestWriteHttpResponseCode(t *testing.T) {
	tests := []struct {
		name         string
		responseCode int
		shouldWrite  bool
	}{
		{"Valid 200", 200, true},
		{"Valid 404", 404, true},
		{"Valid 500", 500, true},
		{"Invalid code", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeHttpResponseCode(w, "test-request-id", "test-hook-id", tt.responseCode)

			if tt.shouldWrite {
				assert.Equal(t, tt.responseCode, w.Code)
			} else {
				// Invalid code should not write header (defaults to 200)
				assert.Equal(t, 200, w.Code)
			}
		})
	}
}

func TestFlushWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	fw := &flushWriter{w: buf}

	// Test Write without Flusher
	n, err := fw.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "test", buf.String())

	// Test Write with Flusher
	buf2 := &bytes.Buffer{}
	flusher := &mockFlusher{ResponseWriter: httptest.NewRecorder()}
	fw2 := &flushWriter{w: buf2, f: flusher}

	n, err = fw2.Write([]byte("test2"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, flusher.flushed)
}

type mockFlusher struct {
	http.ResponseWriter
	flushed bool
}

func (m *mockFlusher) Flush() {
	m.flushed = true
}

func TestMakeSureCallable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	assert.NoError(t, err)

	h := &hook.Hook{
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	cmdPath, err := makeSureCallable(h, r)
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestMakeSureCallable_RelativePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ExecuteCommand:          "test-script.sh",
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	cmdPath, err := makeSureCallable(h, r)
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestMakeSureCallable_WithSpace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test script.sh")

	// Create a test script with space in name
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ExecuteCommand:          "test script.sh",
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	cmdPath, err := makeSureCallable(h, r)
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestCreateHookHandler_HookNotFound(t *testing.T) {
	// Setup
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	// Create a router and add the handler
	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Hook not found")
}

func TestCreateHookHandler_MethodNotAllowed(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:          "test-hook",
		HTTPMethods: []string{"POST"},
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	// Create a router and add the handler
	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCreateHookHandler_AppFlagsHttpMethods(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:          "test-hook",
		HTTPMethods: []string{},
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{
		HttpMethods: "POST,PUT",
	}

	handler := createHookHandler(appFlags)

	// Test with allowed method
	req := httptest.NewRequest("POST", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should not return MethodNotAllowed for POST
	assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code)

	// Test with disallowed method
	req2 := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusMethodNotAllowed, w2.Code)
}

func TestHandleHook_StreamOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script
	scriptContent := "#!/bin/sh\necho 'test output'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		StreamCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	w := httptest.NewRecorder()
	_, err = handleHook(h, r, w)
	assert.NoError(t, err)
}

func TestHandleHook_CaptureOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script
	scriptContent := "#!/bin/sh\necho 'test output'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:   true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	output, err := handleHook(h, r, nil)
	assert.NoError(t, err)
	assert.Contains(t, output, "test output")
}

func TestHandleHook_Async(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script
	scriptContent := "#!/bin/sh\necho 'test output'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	output, err := handleHook(h, r, nil)
	assert.NoError(t, err)
	assert.Contains(t, output, "test output")
}

type mockResponseWriter struct {
	http.ResponseWriter
	header http.Header
	code   int
	body   *bytes.Buffer
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		header: make(http.Header),
		body:   &bytes.Buffer{},
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.body.Write(b)
}

func (m *mockResponseWriter) WriteHeader(code int) {
	m.code = code
}

type mockFlushWriter struct {
	io.Writer
	flushed bool
}

func (m *mockFlushWriter) Flush() {
	m.flushed = true
}

func TestFlushWriter_WithFlusher(t *testing.T) {
	buf := &bytes.Buffer{}
	flusher := &mockFlushWriter{Writer: buf}
	fw := &flushWriter{w: buf, f: flusher}

	n, err := fw.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.True(t, flusher.flushed)
}
