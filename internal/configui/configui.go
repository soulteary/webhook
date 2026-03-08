// Package configui provides the webhook config generator Web UI (HTML + API).
// It can be mounted under a path on the main server when ConfigUI is enabled.
// Located under internal so it is not part of the public API.
package configui

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/yaml"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/hooksdir"
)

func init() {
	// Ensure CSS/JS from embed are served with correct MIME types (some environments default to text/plain).
	_ = mime.AddExtensionType(".css", "text/css")
	_ = mime.AddExtensionType(".js", "application/javascript")
}

//go:embed static
var staticFS embed.FS

//go:embed config
var configFS embed.FS

const (
	pageYAMLPath     = "config/page.yaml"
	maxGenerateBytes = 256 * 1024 // 256KB
)

type pageData struct {
	I18N           template.JS
	Title          string
	Lang           string
	BasePath       string // e.g. /config-ui, used for <base href> so relative URLs work
	ConfigSections []configSection
}

type configSection struct {
	TitleKey    string         `yaml:"titleKey"`
	Options     []configOption `yaml:"options"`
	Collapsible bool           `yaml:"collapsible"`
}

type configOption struct {
	Type        string `yaml:"type"`
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	LabelKey    string `yaml:"labelKey"`
	DescKey     string `yaml:"descKey"`
	Placeholder string `yaml:"placeholder"`
	Default     string `yaml:"default"`
}

type pageYAML struct {
	I18N           map[string]map[string]string `yaml:"i18n"`
	ConfigSections []configSection              `yaml:"configSections"`
}

type generateRequest struct {
	ID                             string `json:"id"`
	ExecuteCommand                 string `json:"execute-command"`
	CommandWorkingDirectory        string `json:"command-working-directory"`
	ResponseMessage                string `json:"response-message"`
	HTTPMethods                    string `json:"http-methods"`
	SuccessHTTPResponseCode        int    `json:"success-http-response-code"`
	IncludeCommandOutputInResponse bool   `json:"include-command-output-in-response"`
	WebhookBaseURL                 string `json:"webhook_base_url"`
	ResponseHeadersJSON            string `json:"response-headers"`
	PassArgumentsToCommandJSON     string `json:"pass-arguments-to-command"`
	PassEnvironmentToCommandJSON   string `json:"pass-environment-to-command"`
	TriggerRuleJSON                string `json:"trigger-rule"`
	IncomingPayloadContentType     string `json:"incoming-payload-content-type"`
}

type generateResponse struct {
	YAML        string `json:"yaml"`
	JSON        string `json:"json"`
	CallURL     string `json:"callUrl"`
	CurlExample string `json:"curlExample"`
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func loadPageData(fromFS fs.FS, path string) (*pageData, error) {
	data, err := fs.ReadFile(fromFS, path)
	if err != nil {
		return nil, err
	}
	var raw pageYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	jsonI18N, err := json.Marshal(raw.I18N)
	if err != nil {
		return nil, err
	}
	title := "Webhook - Config Generator"
	if t, ok := raw.I18N["zh"]["title"]; ok && t != "" {
		title = t
	}
	return &pageData{
		I18N:           template.JS(string(jsonI18N)),
		Title:          title,
		Lang:           "zh-CN",
		BasePath:       "",
		ConfigSections: raw.ConfigSections,
	}, nil
}

func successCode(v int) int {
	if v <= 0 || v >= 1000 {
		return 200
	}
	return v
}

func parseHTTPMethods(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(strings.ToUpper(v))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func validateOptionalJSON(req *generateRequest) string {
	if req == nil {
		return ""
	}
	if req.ResponseHeadersJSON != "" {
		var headers []hook.Header
		if err := json.Unmarshal([]byte(req.ResponseHeadersJSON), &headers); err != nil {
			return "invalid response-headers JSON: " + err.Error()
		}
	}
	if req.PassArgumentsToCommandJSON != "" {
		var args []hook.Argument
		if err := json.Unmarshal([]byte(req.PassArgumentsToCommandJSON), &args); err != nil {
			return "invalid pass-arguments-to-command JSON: " + err.Error()
		}
	}
	if req.PassEnvironmentToCommandJSON != "" {
		var args []hook.Argument
		if err := json.Unmarshal([]byte(req.PassEnvironmentToCommandJSON), &args); err != nil {
			return "invalid pass-environment-to-command JSON: " + err.Error()
		}
	}
	if req.TriggerRuleJSON != "" {
		var r hook.Rules
		if err := json.Unmarshal([]byte(req.TriggerRuleJSON), &r); err != nil {
			return "invalid trigger-rule JSON: " + err.Error()
		}
	}
	return ""
}

func requestToHook(req *generateRequest) *hook.Hook {
	if req == nil {
		return nil
	}
	h := &hook.Hook{
		ID:                         strings.TrimSpace(req.ID),
		ExecuteCommand:             strings.TrimSpace(req.ExecuteCommand),
		CommandWorkingDirectory:    strings.TrimSpace(req.CommandWorkingDirectory),
		ResponseMessage:            strings.TrimSpace(req.ResponseMessage),
		HTTPMethods:                parseHTTPMethods(req.HTTPMethods),
		SuccessHttpResponseCode:    successCode(req.SuccessHTTPResponseCode),
		CaptureCommandOutput:       req.IncludeCommandOutputInResponse,
		IncomingPayloadContentType: strings.TrimSpace(req.IncomingPayloadContentType),
	}
	if req.ResponseHeadersJSON != "" {
		var headers []hook.Header
		if err := json.Unmarshal([]byte(req.ResponseHeadersJSON), &headers); err == nil && len(headers) > 0 {
			h.ResponseHeaders = headers
		}
	}
	if req.PassArgumentsToCommandJSON != "" {
		var args []hook.Argument
		if err := json.Unmarshal([]byte(req.PassArgumentsToCommandJSON), &args); err == nil && len(args) > 0 {
			h.PassArgumentsToCommand = args
		}
	}
	if req.PassEnvironmentToCommandJSON != "" {
		var args []hook.Argument
		if err := json.Unmarshal([]byte(req.PassEnvironmentToCommandJSON), &args); err == nil && len(args) > 0 {
			h.PassEnvironmentToCommand = args
		}
	}
	if req.TriggerRuleJSON != "" {
		var r hook.Rules
		if err := json.Unmarshal([]byte(req.TriggerRuleJSON), &r); err == nil && (r.And != nil || r.Or != nil || r.Not != nil || r.Match != nil) {
			h.TriggerRule = &r
		}
	}
	return h
}

func runGenerate(w http.ResponseWriter, r *http.Request, webhookBaseURL string, hooksURLPrefix string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxGenerateBytes)
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeJSONError(w, http.StatusBadRequest, "id is required")
		return
	}
	if strings.TrimSpace(req.ExecuteCommand) == "" {
		writeJSONError(w, http.StatusBadRequest, "execute-command is required")
		return
	}
	if msg := validateOptionalJSON(&req); msg != "" {
		writeJSONError(w, http.StatusBadRequest, msg)
		return
	}
	h := requestToHook(&req)
	arr := []*hook.Hook{h}
	yamlOut, err := yaml.Marshal(arr)
	if err != nil {
		http.Error(w, "yaml marshal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOut, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		http.Error(w, "json marshal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	baseURL := strings.TrimSuffix(strings.TrimSpace(req.WebhookBaseURL), "/")
	if baseURL == "" || (!strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://")) {
		baseURL = webhookBaseURL
	}
	prefix := strings.TrimSuffix(strings.TrimSpace(hooksURLPrefix), "/")
	if prefix == "" || !strings.HasPrefix(prefix, "/") {
		prefix = "/hooks"
	}
	callURL := fmt.Sprintf("%s%s/%s", baseURL, prefix, h.ID)
	curlExample := fmt.Sprintf("curl -X POST %s -H \"Content-Type: application/json\" -d '{}'", callURL)
	res := generateResponse{
		YAML:        string(yamlOut),
		JSON:        string(jsonOut),
		CallURL:     callURL,
		CurlExample: curlExample,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

const maxSaveBytes = 64 * 1024 // 64KB

type saveRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Format   string `json:"format"`
}

func runSave(w http.ResponseWriter, r *http.Request, writeDir string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if writeDir == "" {
		writeJSONError(w, http.StatusNotImplemented, "save to directory is not enabled in single-file mode")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBytes)
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	trimmed := strings.TrimSpace(req.Filename)
	if strings.Contains(trimmed, "..") {
		writeJSONError(w, http.StatusBadRequest, "invalid filename")
		return
	}
	base := filepath.Base(trimmed)
	if base == "" || base == "." {
		writeJSONError(w, http.StatusBadRequest, "filename is required")
		return
	}
	ext := strings.ToLower(filepath.Ext(base))
	if !hooksdir.HookExts[ext] {
		writeJSONError(w, http.StatusBadRequest, "filename must have extension .json, .yaml or .yml")
		return
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		if ext == ".json" {
			format = "json"
		} else {
			format = "yaml"
		}
	}
	if format != "json" && format != "yaml" {
		writeJSONError(w, http.StatusBadRequest, "format must be json or yaml")
		return
	}
	if format == "json" && ext != ".json" {
		writeJSONError(w, http.StatusBadRequest, "json format requires .json filename")
		return
	}
	if format == "yaml" && ext == ".json" {
		writeJSONError(w, http.StatusBadRequest, "yaml format requires .yaml or .yml filename")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeJSONError(w, http.StatusBadRequest, "content is required")
		return
	}
	var hooks hook.Hooks
	switch format {
	case "json":
		if err := json.Unmarshal([]byte(content), &hooks); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid hook json: "+err.Error())
			return
		}
	default:
		if err := yaml.Unmarshal([]byte(content), &hooks); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid hook yaml: "+err.Error())
			return
		}
	}
	// Prevent path traversal
	if strings.Contains(base, "..") {
		writeJSONError(w, http.StatusBadRequest, "invalid filename")
		return
	}
	target := filepath.Join(writeDir, base)
	// Ensure target is under writeDir
	absTarget, err := filepath.Abs(target)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	absDir, err := filepath.Abs(writeDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	cleanDir := filepath.Clean(absDir)
	dirWithSep := cleanDir + string(filepath.Separator)
	if absTarget != cleanDir && !strings.HasPrefix(absTarget, dirWithSep) {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	if err := os.MkdirAll(writeDir, 0750); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create directory: "+err.Error())
		return
	}
	tmp, err := os.CreateTemp(writeDir, "."+base+".tmp-*")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create temp file: "+err.Error())
		return
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.WriteString(req.Content); err != nil {
		_ = tmp.Close()
		writeJSONError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
		return
	}
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		writeJSONError(w, http.StatusInternalServerError, "failed to set file mode: "+err.Error())
		return
	}
	if err := tmp.Close(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to finalize file: "+err.Error())
		return
	}
	if err := os.Rename(tmpPath, target); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": absTarget})
}

// normalizeBasePath returns a normalized base path (no trailing slash; empty input becomes "/").
// For template use: when result is "/", pass empty string so <base> is not rendered (avoids "//").
func normalizeBasePath(raw string) (normalized string, forTemplate string) {
	normalized = strings.TrimSpace(raw)
	normalized = strings.TrimSuffix(normalized, "/")
	if normalized == "" {
		normalized = "/"
	}
	forTemplate = normalized
	if normalized == "/" {
		forTemplate = ""
	}
	return normalized, forTemplate
}

// Handler returns an http.Handler that serves the config UI and API under the given basePath.
// basePath may be "/", "/config-ui", or "/config-ui/" (trailing slash is normalized). webhookBaseURL is used when
// the client does not provide a base URL (e.g. "http://localhost:9000").
// hooksURLPrefix is the URL path prefix for hooks (e.g. "/hooks" or "/events"); used when generating callUrl; if empty, "/hooks" is used.
// When writeDir is non-empty (e.g. -hooks-dir), the save API is enabled and configs can be written to that directory.
func Handler(basePath string, webhookBaseURL string, writeDir string, hooksURLPrefix string) (http.Handler, error) {
	basePath, baseForTemplate := normalizeBasePath(basePath)

	page, err := loadPageData(configFS, pageYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("load config-ui page config: %w", err)
	}
	page.BasePath = baseForTemplate
	tmpl, err := template.ParseFS(staticFS, "static/index.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse config-ui template: %w", err)
	}
	subFS, _ := fs.Sub(staticFS, "static")
	staticHandler := http.FileServer(http.FS(subFS))

	// When basePath is "/", subpaths must be "/static/" and "/api/generate" (no "//").
	pathPrefix := basePath
	if basePath == "/" {
		pathPrefix = ""
	}

	mux := http.NewServeMux()

	// Register more specific routes before "/" so they match when basePath is "/".
	// Static files: basePath/static/ or /static/ when basePath is "/"
	// Strip pathPrefix+"/static/" so path becomes "css/..." or "js/..." for subFS (root is static dir).
	mux.Handle(pathPrefix+"/static/", http.StripPrefix(pathPrefix+"/static/", staticHandler))

	// API: basePath/api/generate or /api/generate when basePath is "/"
	mux.HandleFunc(pathPrefix+"/api/generate", func(w http.ResponseWriter, r *http.Request) {
		runGenerate(w, r, webhookBaseURL, hooksURLPrefix)
	})

	// API: basePath/api/save — write generated config to writeDir (when -hooks-dir is set)
	mux.HandleFunc(pathPrefix+"/api/save", func(w http.ResponseWriter, r *http.Request) {
		runSave(w, r, writeDir)
	})

	// API: basePath/api/capabilities — whether save-to-dir is available
	mux.HandleFunc(pathPrefix+"/api/capabilities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"saveToDir": writeDir != ""})
	})

	// Index: exact basePath or basePath/
	indexHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != basePath && path != basePath+"/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if err := tmpl.Execute(w, page); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	mux.HandleFunc(basePath, indexHandler)
	mux.HandleFunc(basePath+"/", indexHandler)

	return mux, nil
}
