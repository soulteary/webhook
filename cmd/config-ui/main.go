// Package main provides the webhook config generator WebUI (standalone binary).
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/invopop/yaml"
	"github.com/soulteary/webhook/internal/hook"
)

//go:embed static
var staticFS embed.FS

//go:embed config
var configFS embed.FS

const (
	pageYAMLPath     = "config/page.yaml"
	defaultPort      = "9080"
	maxGenerateBytes = 256 * 1024 // 256KB
)

type pageData struct {
	I18N            template.JS
	Title           string
	Lang            string
	ConfigSections  []configSection
}

type configSection struct {
	TitleKey   string         `yaml:"titleKey"`
	Options    []configOption `yaml:"options"`
	Collapsible bool         `yaml:"collapsible"`
}

type configOption struct {
	Type       string `yaml:"type"`
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	LabelKey   string `yaml:"labelKey"`
	DescKey    string `yaml:"descKey"`
	Placeholder string `yaml:"placeholder"`
	Default    string `yaml:"default"`
}

type pageYAML struct {
	I18N           map[string]map[string]string `yaml:"i18n"`
	ConfigSections []configSection             `yaml:"configSections"`
}

type generateRequest struct {
	ID                              string `json:"id"`
	ExecuteCommand                  string `json:"execute-command"`
	CommandWorkingDirectory        string `json:"command-working-directory"`
	ResponseMessage                string `json:"response-message"`
	HTTPMethods                    string `json:"http-methods"` // comma-separated or single
	SuccessHTTPResponseCode        int    `json:"success-http-response-code"`
	IncludeCommandOutputInResponse  bool   `json:"include-command-output-in-response"`
	WebhookBaseURL                 string `json:"webhook_base_url"` // e.g. http://localhost:9000
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

func cacheControlHandler(value string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		h.ServeHTTP(w, r)
	})
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
		I18N:           template.JS(jsonI18N),
		Title:          title,
		Lang:           "zh-CN",
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

// validateOptionalJSON returns a non-empty error message if any optional JSON field is invalid.
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
		ID:                          strings.TrimSpace(req.ID),
		ExecuteCommand:              strings.TrimSpace(req.ExecuteCommand),
		CommandWorkingDirectory:     strings.TrimSpace(req.CommandWorkingDirectory),
		ResponseMessage:             strings.TrimSpace(req.ResponseMessage),
		HTTPMethods:                 parseHTTPMethods(req.HTTPMethods),
		SuccessHttpResponseCode:     successCode(req.SuccessHTTPResponseCode),
		CaptureCommandOutput:        req.IncludeCommandOutputInResponse,
		IncomingPayloadContentType:  strings.TrimSpace(req.IncomingPayloadContentType),
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

func runGenerate(w http.ResponseWriter, r *http.Request, port string) {
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
	if baseURL != "" && (strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://")) {
		// use provided webhook base URL
	} else {
		host := r.Host
		if idx := strings.Index(host, ":"); idx > 0 {
			host = host[:idx]
		}
		if host == "" {
			host = "localhost"
		}
		baseURL = fmt.Sprintf("http://%s:%s", host, port)
	}
	callURL := fmt.Sprintf("%s/hooks/%s", baseURL, h.ID)
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

func main() {
	port := defaultPort
	if p := os.Getenv("PORT"); p != "" {
		port = strings.TrimSpace(p)
	}
	flagSet := flag.NewFlagSet("config-ui", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	portFlag := flagSet.String("port", port, "HTTP port for the config UI (default "+defaultPort+")")
	_ = flagSet.Parse(os.Args[1:])
	if *portFlag != "" {
		port = strings.TrimSpace(*portFlag)
	}
	if port == "" {
		port = defaultPort
	}

	var page *pageData
	if data, err := fs.ReadFile(configFS, pageYAMLPath); err == nil {
		var raw pageYAML
		if err := yaml.Unmarshal(data, &raw); err != nil {
			fmt.Fprintf(os.Stderr, "parse embedded config: %v\n", err)
			os.Exit(1)
		}
		jsonI18N, _ := json.Marshal(raw.I18N)
		title := "Webhook - Config Generator"
		if t, ok := raw.I18N["zh"]["title"]; ok && t != "" {
			title = t
		}
		page = &pageData{
			I18N:           template.JS(jsonI18N),
			Title:          title,
			Lang:           "zh-CN",
			ConfigSections: raw.ConfigSections,
		}
	} else {
		// Try current dir or project root
		paths := []string{pageYAMLPath, "cmd/config-ui/" + pageYAMLPath}
		for _, p := range paths {
			if pd, err := loadPageData(os.DirFS("."), p); err == nil {
				page = pd
				break
			}
		}
		if page == nil {
			wd, _ := os.Getwd()
			for _, p := range []string{filepath.Join(wd, pageYAMLPath), filepath.Join(wd, "config", "page.yaml")} {
				data, err := os.ReadFile(p)
				if err != nil {
					continue
				}
				var raw pageYAML
				if err := yaml.Unmarshal(data, &raw); err != nil {
					continue
				}
				jsonI18N, _ := json.Marshal(raw.I18N)
				title := "Webhook - Config Generator"
				if t, ok := raw.I18N["zh"]["title"]; ok && t != "" {
					title = t
				}
				page = &pageData{I18N: template.JS(jsonI18N), Title: title, Lang: "zh-CN", ConfigSections: raw.ConfigSections}
				break
			}
		}
		if page == nil {
			fmt.Fprintf(os.Stderr, "load page config: %v (tried embedded and %s)\n", err, pageYAMLPath)
			os.Exit(1)
		}
	}

	tmpl, err := template.ParseFS(staticFS, "static/index.html.tmpl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse template: %v\n", err)
		os.Exit(1)
	}
	subFS, _ := fs.Sub(staticFS, "static")
	staticHandler := cacheControlHandler("public, max-age=3600", http.FileServer(http.FS(subFS)))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, page); err != nil {
			fmt.Fprintf(os.Stderr, "template execute: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	})
	mux.Handle("/static/", http.StripPrefix("/static", staticHandler))
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		runGenerate(w, r, port)
	})

	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		}
	}()
	fmt.Printf("Webhook Config UI: http://localhost%s\n", addr)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		os.Exit(1)
	}
}
