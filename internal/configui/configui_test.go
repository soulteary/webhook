package configui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuccessCode(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{0, 200},
		{-1, 200},
		{200, 200},
		{201, 201},
		{404, 404},
		{999, 999},
		{1000, 200},
		{1001, 200},
	}
	for _, tt := range tests {
		got := successCode(tt.in)
		if got != tt.want {
			t.Errorf("successCode(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseHTTPMethods(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{"POST", []string{"POST"}},
		{"post", []string{"POST"}},
		{"post, get", []string{"POST", "GET"}},
		{" POST ,  GET ", []string{"POST", "GET"}},
	}
	for _, tt := range tests {
		got := parseHTTPMethods(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("parseHTTPMethods(%q) = %v, want %v", tt.in, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseHTTPMethods(%q) = %v, want %v", tt.in, got, tt.want)
				break
			}
		}
	}
}

func TestValidateOptionalJSON(t *testing.T) {
	tests := []struct {
		name string
		req  *generateRequest
		want string
	}{
		{"nil", nil, ""},
		{"empty", &generateRequest{}, ""},
		{"invalid response-headers", &generateRequest{ResponseHeadersJSON: "not json"}, "invalid response-headers JSON"},
		{"valid response-headers", &generateRequest{ResponseHeadersJSON: `[{"name":"X","value":"y"}]`}, ""},
		{"invalid trigger-rule", &generateRequest{TriggerRuleJSON: "{"}, "invalid trigger-rule JSON"},
		{"valid trigger-rule", &generateRequest{TriggerRuleJSON: `{"match":{"type":"value","parameter":{"source":"header","name":"X"},"value":"v"}}`}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateOptionalJSON(tt.req)
			if tt.want == "" && got != "" {
				t.Errorf("validateOptionalJSON() = %q, want empty", got)
			}
			if tt.want != "" && (got == "" || !strings.Contains(got, tt.want)) {
				t.Errorf("validateOptionalJSON() = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestRequestToHook(t *testing.T) {
	req := &generateRequest{
		ID:                      "test-hook",
		ExecuteCommand:          "/bin/true",
		ResponseMessage:         "OK",
		HTTPMethods:             "POST",
		SuccessHTTPResponseCode: 200,
	}
	h := requestToHook(req)
	if h == nil {
		t.Fatal("requestToHook() returned nil")
	}
	if h.ID != "test-hook" || h.ExecuteCommand != "/bin/true" || h.ResponseMessage != "OK" {
		t.Errorf("requestToHook() = %+v", h)
	}
	if len(h.HTTPMethods) != 1 || h.HTTPMethods[0] != "POST" {
		t.Errorf("HTTPMethods = %v", h.HTTPMethods)
	}
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusBadRequest, "id is required")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body["error"] != "id is required" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestHandler(t *testing.T) {
	h, err := Handler("/config-ui", "http://localhost:9000")
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	if h == nil {
		t.Fatal("Handler returned nil")
	}

	// Index page
	req := httptest.NewRequest(http.MethodGet, "http://test/config-ui", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /config-ui: status %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("GET /config-ui: Content-Type %q", rec.Header().Get("Content-Type"))
	}
	if rec.Body.Len() == 0 {
		t.Error("GET /config-ui: empty body")
	}

	// Index with trailing slash
	req2 := httptest.NewRequest(http.MethodGet, "http://test/config-ui/", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("GET /config-ui/: status %d, want 200", rec2.Code)
	}

	// 404 for unknown path under base
	req3 := httptest.NewRequest(http.MethodGet, "http://test/config-ui/unknown", nil)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNotFound {
		t.Errorf("GET /config-ui/unknown: status %d, want 404", rec3.Code)
	}
}

func TestHandlerRootAPIGenerate(t *testing.T) {
	h, err := Handler("/", "http://localhost:9080")
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	body := `{"id":"test-hook","execute-command":"/bin/true"}`
	req := httptest.NewRequest(http.MethodPost, "http://test/api/generate", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST /api/generate: status %d, want 200, body: %s", w.Code, w.Body.Bytes())
		return
	}
	var res generateResponse
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if res.CallURL == "" || !strings.Contains(res.CallURL, "/hooks/test-hook") {
		t.Errorf("CallURL = %q", res.CallURL)
	}
	if res.CurlExample == "" || !strings.Contains(res.CurlExample, "test-hook") {
		t.Errorf("CurlExample = %q", res.CurlExample)
	}
	if res.YAML == "" || !strings.Contains(res.YAML, "test-hook") {
		t.Errorf("YAML missing or invalid")
	}
	if res.JSON == "" || !strings.Contains(res.JSON, "test-hook") {
		t.Errorf("JSON missing or invalid")
	}
}

func TestHandlerRootAPIGenerateBadRequest(t *testing.T) {
	h, err := Handler("/", "http://localhost:9080")
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	tests := []struct {
		name string
		body string
		want int
	}{
		{"missing id", `{"execute-command":"/bin/true"}`, http.StatusBadRequest},
		{"missing command", `{"id":"x"}`, http.StatusBadRequest},
		{"invalid json", `{`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://test/api/generate", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("Code = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
