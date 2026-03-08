package openapi

import (
	"encoding/json"
	"strings"

	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/link"
	"github.com/soulteary/webhook/internal/version"
)

// Spec generates an OpenAPI 3.0.x specification for the webhook HTTP API.
// serverURL is optional; if non-empty it is used as servers[0].url.
func Spec(appFlags flags.AppFlags, serverURL string) ([]byte, error) {
	hookBase := link.MakeBaseURL(&appFlags.HooksURLPrefix)
	if hookBase == "" {
		hookBase = "/hooks"
	}
	hookPath := hookBase + "/{id}"

	paths := map[string]any{
		"/": map[string]any{
			"get": op("Root", "Returns OK when the server is running.", "text/plain"),
		},
		"/health": map[string]any{
			"get": op("Health check", "Aggregated health status (health-kit).", "application/json"),
		},
		"/livez": map[string]any{
			"get": op("Liveness", "Kubernetes-style liveness probe.", "application/json"),
		},
		"/readyz": map[string]any{
			"get": op("Readiness", "Kubernetes-style readiness probe.", "application/json"),
		},
		"/version": map[string]any{
			"get": op("Version", "Server version and build info.", "application/json"),
		},
		"/metrics": map[string]any{
			"get": op("Metrics", "Prometheus metrics.", "text/plain"),
		},
		hookPath: hooksPathOp(appFlags),
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Webhook API",
			"description": "HTTP API for webhook: trigger hooks by ID. Health, version, and metrics endpoints are also available.",
			"version":     version.Version,
		},
		"paths": paths,
	}

	if serverURL != "" {
		spec["servers"] = []map[string]any{
			{"url": strings.TrimSuffix(serverURL, "/")},
		}
	}

	return json.MarshalIndent(spec, "", "  ")
}

func op(summary, description, contentType string) map[string]any {
	return map[string]any{
		"summary":     summary,
		"description": description,
		"responses": map[string]any{
			"200": map[string]any{
				"description": "Success",
				"content": map[string]any{
					contentType: map[string]any{"schema": map[string]any{"type": "string"}},
				},
			},
		},
	}
}

func hooksPathOp(appFlags flags.AppFlags) map[string]any {
	methods := appFlags.HttpMethods
	if methods == "" {
		methods = "POST,PUT,PATCH"
	}
	parts := strings.Split(methods, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.ToUpper(p))
	}
	if len(parts) == 0 {
		parts = []string{"POST"}
	}

	ops := make(map[string]any)
	for _, m := range parts {
		if m == "" {
			continue
		}
		ops[strings.ToLower(m)] = map[string]any{
			"summary":     "Trigger hook by ID",
			"description": "Executes the hook identified by {id}. Request body is optional; supported content types: application/json, application/x-www-form-urlencoded, multipart/form-data. Response body and status are determined by hook configuration.",
			"parameters": []map[string]any{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"description": "Hook identifier (may include slashes, e.g. sendgrid/event)",
					"schema":      map[string]any{"type": "string"},
				},
			},
			"requestBody": map[string]any{
				"required": false,
				"content": map[string]any{
					"application/json":                  map[string]any{"schema": map[string]any{"type": "object"}},
					"application/x-www-form-urlencoded": map[string]any{"schema": map[string]any{"type": "object"}},
					"multipart/form-data":               map[string]any{"schema": map[string]any{"type": "object"}},
				},
			},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Hook executed successfully (or rule not matched; body depends on hook config)",
					"content": map[string]any{
						"text/plain":       map[string]any{"schema": map[string]any{"type": "string"}},
						"application/json": map[string]any{"schema": map[string]any{"type": "object"}},
					},
				},
				"400": map[string]any{"description": "Bad request or hook rules not satisfied"},
				"404": map[string]any{"description": "Hook not found"},
				"405": map[string]any{"description": "Method not allowed for this hook"},
				"500": map[string]any{"description": "Internal server error during hook execution"},
				"503": map[string]any{"description": "Server shutting down"},
			},
		}
	}

	return ops
}
