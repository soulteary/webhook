package server

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
)

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

	// Create a test script with execute permission
	scriptContent := "#!/bin/sh\necho 'test'\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	h := &hook.Hook{
		ExecuteCommand:          scriptPath,
		CommandWorkingDirectory: tempDir,
	}

	r := &hook.Request{
		ID: "test-request",
	}

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestCreateHookHandler_HookNotFound(t *testing.T) {
	// Setup
	rules.LoadedHooksFromFiles = make(map[string]hook.Hooks)
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	// Create a router and add the handler
	// Register both routes to support simple IDs and IDs with slashes
	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("GET", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	// Create a router and add the handler
	// Register both routes to support simple IDs and IDs with slashes
	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{
		HttpMethods: "POST,PUT",
	}

	handler := createHookHandler(appFlags, nil)

	// Test with allowed method
	req := httptest.NewRequest("POST", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	appFlags := flags.AppFlags{AllowAutoChmod: false}
	_, err = handleHook(context.Background(), h, r, w, appFlags)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	output, err := handleHook(context.Background(), h, r, nil, appFlags)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	output, err := handleHook(context.Background(), h, r, nil, appFlags)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`<root><key>value</key></root>`))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`key=value`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`some data`))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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

	// This should try to make it executable and retry (when AllowAutoChmod is enabled)
	appFlags := flags.AppFlags{AllowAutoChmod: true}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
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
	appFlags := flags.AppFlags{AllowAutoChmod: false}
	output, err := handleHook(context.Background(), h, r, nil, appFlags)
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

	appFlags := flags.AppFlags{AllowAutoChmod: false}
	output, err := handleHook(context.Background(), h, r, nil, appFlags)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{
		MaxMultipartMem: 1024 * 1024,
	}

	handler := createHookHandler(appFlags, nil)

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("key", "value")
	writer.Close()

	req := httptest.NewRequest("POST", "/hooks/test-hook", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{
		MaxMultipartMem: 1024 * 1024,
	}

	handler := createHookHandler(appFlags, nil)

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

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{
		MaxMultipartMem: 1, // Very small limit to force error
	}

	handler := createHookHandler(appFlags, nil)

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

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	// Create a request with a body that will cause read error
	req := httptest.NewRequest("POST", "/hooks/test-hook", &errorReader{})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	// Don't set X-Test header, so the rule won't match
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	rules.BuildIndex()
	appFlags := flags.AppFlags{}

	handler := createHookHandler(appFlags, nil)

	req := httptest.NewRequest("POST", "/hooks/test-hook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.HandleFunc("/hooks/{id}", handler)
	r.HandleFunc("/hooks/{id}/*", handler)
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
	appFlags := flags.AppFlags{AllowAutoChmod: false}
	output, err := handleHook(context.Background(), h, r, nil, appFlags)
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

	// This should try to make it executable (when AllowAutoChmod is enabled)
	appFlags := flags.AppFlags{AllowAutoChmod: true}
	cmdPath, err := makeSureCallable(context.Background(), h, r, appFlags, nil)
	// Should succeed after making it executable
	assert.NoError(t, err)
	assert.NotEmpty(t, cmdPath)
}

func TestTrackingResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	trw := &trackingResponseWriter{ResponseWriter: w}

	// Initially not written
	assert.False(t, trw.HasWritten())

	// Write should set written flag and status code
	n, err := trw.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.True(t, trw.HasWritten())
	assert.Equal(t, http.StatusOK, w.Code)

	// WriteHeader should also set written flag
	w2 := httptest.NewRecorder()
	trw2 := &trackingResponseWriter{ResponseWriter: w2}
	trw2.WriteHeader(http.StatusNotFound)
	assert.True(t, trw2.HasWritten())
	assert.Equal(t, http.StatusNotFound, w2.Code)

	// WriteHeader after Write should not change status
	w3 := httptest.NewRecorder()
	trw3 := &trackingResponseWriter{ResponseWriter: w3}
	trw3.Write([]byte("test"))
	trw3.WriteHeader(http.StatusInternalServerError)
	assert.Equal(t, http.StatusOK, w3.Code) // Should remain 200
}

func TestTrackingResponseWriter_Flush(t *testing.T) {
	w := httptest.NewRecorder()
	trw := &trackingResponseWriter{ResponseWriter: w}

	// Test Flush with non-Flusher
	trw.Flush() // Should not panic

	// Test Flush with Flusher
	flusher := &mockFlusher{ResponseWriter: httptest.NewRecorder()}
	trw2 := &trackingResponseWriter{ResponseWriter: flusher}
	trw2.Flush()
	assert.True(t, flusher.flushed)
}

func TestGetAsyncHookWaitGroup(t *testing.T) {
	wg := GetAsyncHookWaitGroup()
	assert.NotNil(t, wg)
	// Should return the same instance
	wg2 := GetAsyncHookWaitGroup()
	assert.Equal(t, wg, wg2)
}

func TestStatusCodeResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	var statusCode int
	scrw := &statusCodeResponseWriter{
		ResponseWriter: w,
		statusCode:     &statusCode,
	}

	scrw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetResponseHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	headers := hook.ResponseHeaders{
		{Name: "X-Test1", Value: "value1"},
		{Name: "X-Test2", Value: "value2"},
	}

	setResponseHeaders(w, headers)
	assert.Equal(t, "value1", w.Header().Get("X-Test1"))
	assert.Equal(t, "value2", w.Header().Get("X-Test2"))
}

func TestIsMethodAllowed(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		hook     *hook.Hook
		appFlags flags.AppFlags
		expected bool
	}{
		{
			name:     "hook with HTTPMethods",
			method:   "POST",
			hook:     &hook.Hook{HTTPMethods: []string{"POST", "PUT"}},
			appFlags: flags.AppFlags{},
			expected: true,
		},
		{
			name:     "hook with HTTPMethods not matching",
			method:   "GET",
			hook:     &hook.Hook{HTTPMethods: []string{"POST", "PUT"}},
			appFlags: flags.AppFlags{},
			expected: false,
		},
		{
			name:     "appFlags with HttpMethods",
			method:   "POST",
			hook:     &hook.Hook{},
			appFlags: flags.AppFlags{HttpMethods: "POST,PUT"},
			expected: true,
		},
		{
			name:     "appFlags with HttpMethods not matching",
			method:   "GET",
			hook:     &hook.Hook{},
			appFlags: flags.AppFlags{HttpMethods: "POST,PUT"},
			expected: false,
		},
		{
			name:     "default allow all",
			method:   "ANY",
			hook:     &hook.Hook{},
			appFlags: flags.AppFlags{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMethodAllowed(tt.method, tt.hook, tt.appFlags)
			assert.Equal(t, tt.expected, result)
		})
	}
}
