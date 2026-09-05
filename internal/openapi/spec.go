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
			"get": versionOp(),
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

func versionOp() map[string]any {
	stringProperty := func(description string) map[string]any {
		return map[string]any{
			"type":        "string",
			"description": description,
		}
	}

	return map[string]any{
		"summary":     "Version",
		"description": "Server version and build info.",
		"responses": map[string]any{
			"200": map[string]any{
				"description": "Success",
				"headers": map[string]any{
					"X-Version":    map[string]any{"schema": map[string]any{"type": "string"}},
					"X-Commit":     map[string]any{"schema": map[string]any{"type": "string"}},
					"X-Build-Date": map[string]any{"schema": map[string]any{"type": "string"}},
					"X-Branch":     map[string]any{"schema": map[string]any{"type": "string"}},
				},
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"type":     "object",
							"required": []string{"version", "go_version", "platform", "compiler"},
							"properties": map[string]any{
								"version":    stringProperty("Application version."),
								"commit":     stringProperty("Git commit hash."),
								"build_date": stringProperty("Build timestamp."),
								"branch":     stringProperty("Git branch name."),
								"go_version": stringProperty("Go runtime version."),
								"platform":   stringProperty("Runtime operating system and architecture."),
								"compiler":   stringProperty("Go compiler."),
							},
						},
					},
				},
			},
		},
	}
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
	allowAnyMethod := methods == ""
	if methods == "" {
		methods = "GET,HEAD,POST,PUT,PATCH,DELETE,CONNECT,OPTIONS,TRACE"
	}
	parts := strings.Split(methods, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.ToUpper(p))
	}
	if len(parts) == 0 {
		parts = []string{"POST"}
	}

	ops := make(map[string]any)
	additionalMethods := make([]string, 0)
	for _, m := range parts {
		if m == "" {
			continue
		}
		switch m {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "TRACE":
			// OpenAPI 3.0 path-item operations.
		default:
			additionalMethods = append(additionalMethods, m)
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
				"400": plainTextResponse("Bad request or hook rules not satisfied"),
				"404": plainTextResponse("Hook not found"),
				"405": methodNotAllowedResponse(),
				"408": plainTextResponse("Hook execution timed out"),
				"429": plainTextResponse("Rate limit exceeded"),
				"500": plainTextResponse("Internal server error during hook execution"),
				"503": plainTextResponse("Server shutting down"),
			},
		}
	}
	if len(additionalMethods) > 0 {
		ops["x-webhook-additional-methods"] = additionalMethods
	}
	if allowAnyMethod {
		ops["x-webhook-allow-any-method"] = true
	}

	return ops
}

func plainTextResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"text/plain": map[string]any{"schema": map[string]any{"type": "string"}},
		},
	}
}

func methodNotAllowedResponse() map[string]any {
	response := plainTextResponse("Method not allowed for this hook")
	response["headers"] = map[string]any{
		"Allow": map[string]any{
			"description": "Methods configured for the matched hook.",
			"schema":      map[string]any{"type": "string"},
		},
	}
	return response
}
