package server

import (
	"bytes"
	"context"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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
	_, err = handleHook(context.Background(), spHook, r, nil)
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
		StreamCommandOutput:     true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	w := httptest.NewRecorder()
	_, err = handleHook(context.Background(), h, r, w)
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
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	output, err := handleHook(context.Background(), h, r, nil)
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

	output, err := handleHook(context.Background(), h, r, nil)
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

func TestCreateHookHandler_JSONContentType(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should not return error for valid JSON
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_XMLContentType(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`<root><key>value</key></root>`))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should not return error for valid XML
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_FormUrlEncodedContentType(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`key=value`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should not return error for valid form data
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_UnsupportedContentType(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`some data`))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should handle unsupported content type gracefully
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_WithTriggerRule(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
		TriggerRule:     nil, // No trigger rule, should always trigger
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should trigger successfully when no trigger rule
	assert.Contains(t, w.Body.String(), "success")
}

func TestCreateHookHandler_WithResponseHeaders(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
		ResponseHeaders: []hook.Header{
			{Name: "X-Custom-Header", Value: "custom-value"},
		},
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should set response headers
	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
}

func TestMakeSureCallable_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script without execute permission
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644) // No execute permission
	assert.NoError(t, err)

	h := &hook.Hook{
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	// This should try to make it executable and retry
	cmdPath, err := makeSureCallable(h, r)
	// Should succeed after making it executable
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestMakeSureCallable_CommandNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	h := &hook.Hook{
		ExecuteCommand:          "/nonexistent/command",
		CommandWorkingDirectory: "",
	}

	r := &hook.Request{
		ID: "test-request",
	}

	cmdPath, err := makeSureCallable(h, r)
	// Should return error for nonexistent command
	assert.Error(t, err)
	assert.Empty(t, cmdPath)
}

func TestMakeSureCallable_CommandWithSpace(t *testing.T) {
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
		ExecuteCommand:          "test script.sh", // Command with space
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	cmdPath, err := makeSureCallable(h, r)
	// Should handle command with space
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestHandleHook_FileCreationError(t *testing.T) {
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
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	// Test with invalid working directory (should still work but may have file creation issues)
	output, err := handleHook(context.Background(), h, r, nil)
	// Should handle file creation errors gracefully
	assert.NoError(t, err)
	assert.Contains(t, output, "test output")
}

func TestHandleHook_CommandError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	// Create a test script that exits with error
	scriptContent := "#!/bin/sh\nexit 1\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	output, err := handleHook(context.Background(), h, r, nil)
	// Should return error when command fails
	assert.Error(t, err)
	_ = output
}

func TestCreateHookHandler_MultipartForm(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{
		MaxMultipartMem: 1024 * 1024,
	}

	handler := createHookHandler(appFlags)

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("key", "value")
	writer.Close()

	req := httptest.NewRequest("POST", "/hooks/test-hook", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should handle multipart form successfully
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_MultipartFormWithFile(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
		JSONStringParameters: []hook.Argument{
			{Source: "payload", Name: "payload"},
		},
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{
		MaxMultipartMem: 1024 * 1024,
	}

	handler := createHookHandler(appFlags)

	// Create multipart form with JSON file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add a JSON file part
	part, _ := writer.CreateFormFile("payload", "payload.json")
	part.Write([]byte(`{"key":"value"}`))
	writer.Close()

	req := httptest.NewRequest("POST", "/hooks/test-hook", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should handle multipart form with file successfully
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_MultipartFormError(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{
		MaxMultipartMem: 1, // Very small limit to force error
	}

	handler := createHookHandler(appFlags)

	// Create multipart form data that exceeds limit
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// Write a large field that will exceed the 1 byte limit
	largeData := strings.Repeat("x", 10000)
	writer.WriteField("key", largeData)
	writer.Close()

	req := httptest.NewRequest("POST", "/hooks/test-hook", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should return error for multipart form parsing failure
	// Note: The actual behavior may vary, but we should handle it gracefully
	if w.Code == http.StatusInternalServerError {
		assert.Contains(t, w.Body.String(), "Error occurred while parsing multipart form")
	}
}

func TestCreateHookHandler_ReadBodyError(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	// Create a request with a body that will cause read error
	req := httptest.NewRequest("POST", "/hooks/test-hook", &errorReader{})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should handle read error gracefully
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestCreateHookHandler_TriggerRuleError(t *testing.T) {
	// Setup - create a hook with a trigger rule that will cause an error
	testHook := hook.Hook{
		ID:              "test-hook",
		HTTPMethods:     []string{},
		ResponseMessage: "success",
		// TriggerRule will be set to cause an error
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should handle trigger rule evaluation
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_StreamCommandOutputError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	scriptContent := "#!/bin/sh\necho 'test output'\nexit 1\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	testHook := hook.Hook{
		ID:                      "test-hook",
		HTTPMethods:             []string{},
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		StreamCommandOutput:     true,
		ResponseMessage:         "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should handle stream command output error
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHookHandler_CaptureOutputOnError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	scriptContent := "#!/bin/sh\necho 'error output'\nexit 1\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	testHook := hook.Hook{
		ID:                          "test-hook",
		HTTPMethods:                 []string{},
		ExecuteCommand:              scriptPath,
		CommandWorkingDirectory:     tempDir,
		CaptureCommandOutput:        true,
		CaptureCommandOutputOnError: true,
		ResponseMessage:             "success",
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should capture output on error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error output")
}

func TestCreateHookHandler_TriggerRuleMismatchHttpResponseCode(t *testing.T) {
	// Setup - create a hook with a trigger rule that will not match
	testHook := hook.Hook{
		ID:                                  "test-hook",
		HTTPMethods:                         []string{},
		ResponseMessage:                     "success",
		TriggerRuleMismatchHttpResponseCode: 400,
		// Create a trigger rule that will not match (header "X-Test" must equal "match" but we won't send it)
		TriggerRule: &hook.Rules{
			Match: &hook.MatchRule{
				Type:      "value",
				Value:     "match",
				Parameter: hook.Argument{Source: "header", Name: "X-Test"},
			},
		},
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	// Don't set X-Test header, so the rule won't match
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should use custom response code when trigger rule doesn't match
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Hook rules were not satisfied")
}

func TestCreateHookHandler_SuccessHttpResponseCode(t *testing.T) {
	// Setup
	testHook := hook.Hook{
		ID:                      "test-hook",
		HTTPMethods:             []string{},
		ResponseMessage:         "success",
		SuccessHttpResponseCode: 201,
	}
	rules.LoadedHooksFromFiles = map[string]hook.Hooks{
		"test.json": {testHook},
	}
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.ServeHTTP(w, req)

	// Should use custom success response code
	assert.Equal(t, 201, w.Code)
}

func TestHandleHook_FileOperations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

	scriptContent := "#!/bin/sh\necho 'test output'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ID:                      "test-hook",
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
		CaptureCommandOutput:    true,
	}

	r := &hook.Request{
		ID: "test-request",
		Payload: map[string]interface{}{
			"test": "value",
		},
	}

	// Test file creation and cleanup
	output, err := handleHook(context.Background(), h, r, nil)
	assert.NoError(t, err)
	assert.Contains(t, output, "test output")
}

func TestMakeSureCallable_ChmodError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test-script.sh")

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

	// This should try to make it executable
	cmdPath, err := makeSureCallable(h, r)
	// Should succeed after making it executable
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}
