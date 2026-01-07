# API Reference

This document describes all HTTP endpoints provided by the Webhook server.

## Base URL

By default, the Webhook server runs on port `9000`. The base URL format is:

```
http://your-server:9000
```

You can customize the IP address and port using the `-ip` and `-port` command-line arguments or environment variables.

## Endpoints

### 1. Root Endpoint

**Endpoint:** `GET /`

**Description:** Simple health check endpoint that returns "OK".

**Response:**
- **Status Code:** `200 OK`
- **Content-Type:** `text/plain`
- **Body:** `OK`

**Example:**
```bash
curl http://localhost:9000/
```

---

### 2. Health Check Endpoint

**Endpoint:** `GET /health`

**Description:** Health check endpoint that returns server status in JSON format.

**Response:**
- **Status Code:** `200 OK`
- **Content-Type:** `application/json`
- **Body:**
```json
{
  "status": "ok"
}
```

**Example:**
```bash
curl http://localhost:9000/health
```

---

### 3. Metrics Endpoint

**Endpoint:** `GET /metrics`

**Description:** Prometheus metrics endpoint for monitoring and observability.

**Response:**
- **Status Code:** `200 OK`
- **Content-Type:** `text/plain; version=0.0.4; charset=utf-8`
- **Body:** Prometheus metrics in text format

**Example:**
```bash
curl http://localhost:9000/metrics
```

**Available Metrics:**
- `webhook_http_requests_total`: Total number of HTTP requests
- `webhook_http_request_duration_seconds`: HTTP request duration histogram
- `webhook_hook_executions_total`: Total number of hook executions
- `webhook_hook_execution_duration_seconds`: Hook execution duration histogram
- `webhook_system_memory_bytes`: System memory usage
- `webhook_system_cpu_percent`: System CPU usage percentage

---

### 4. Hook Execution Endpoint

**Endpoint:** `POST|GET|PUT|DELETE /hooks/{hook-id}`

**Description:** Execute a configured hook. The HTTP methods allowed depend on the hook configuration and the `-http-methods` flag.

**URL Parameters:**
- `hook-id` (required): The ID of the hook to execute, as defined in your hooks configuration file.

**Request Headers:**
- `Content-Type`: Optional. Can be `application/json`, `application/x-www-form-urlencoded`, `multipart/form-data`, or `text/plain`.
- `X-Request-Id`: Optional. If provided and `-x-request-id` is enabled, this will be used as the request ID for logging.

**Request Body:**
The request body can contain:
- JSON data
- Form data (URL-encoded or multipart)
- Plain text
- Query parameters (for GET requests)

**Response:**
- **Status Code:** 
  - `200 OK`: Hook executed successfully
  - `400 Bad Request`: Invalid request (e.g., malformed JSON, missing required parameters)
  - `404 Not Found`: Hook ID not found
  - `405 Method Not Allowed`: HTTP method not allowed for this hook
  - `408 Request Timeout`: Request timeout
  - `429 Too Many Requests`: Rate limit exceeded (if rate limiting is enabled)
  - `500 Internal Server Error`: Server error during hook execution
  - `503 Service Unavailable`: Server is shutting down
  - Custom status code: As configured in `success-http-response-code` or `trigger-rule-mismatch-http-response-code`

- **Content-Type:** 
  - `text/plain` (default)
  - `application/json` (if error occurs)
  - As configured in `response-headers`

- **Response Body:**
  - Success: Custom message from `response-message`, command output (if `include-command-output-in-response` is enabled), or default message
  - Error: JSON error response with details

**Error Response Format:**
```json
{
  "error": "Error Type",
  "message": "Error message",
  "request_id": "request-id-here",
  "hook_id": "hook-id-here"
}
```

**Example - Successful Execution:**
```bash
# POST request with JSON body
curl -X POST http://localhost:9000/hooks/redeploy-webhook \
  -H "Content-Type: application/json" \
  -d '{"branch": "main", "commit": "abc123"}'

# GET request with query parameters
curl "http://localhost:9000/hooks/redeploy-webhook?branch=main&commit=abc123"
```

**Example - Hook Not Found:**
```bash
curl -X POST http://localhost:9000/hooks/non-existent-hook
```

**Response:**
```json
{
  "error": "Not Found",
  "message": "Hook not found.",
  "request_id": "req-123",
  "hook_id": "non-existent-hook"
}
```

**Example - Method Not Allowed:**
```bash
# If hook only allows POST, but we send GET
curl -X GET http://localhost:9000/hooks/post-only-hook
```

**Response:**
```json
{
  "error": "Method Not Allowed",
  "message": "HTTP GET method not allowed for hook \"post-only-hook\"",
  "request_id": "req-456",
  "hook_id": "post-only-hook"
}
```

---

## Request ID

Webhook automatically generates a unique request ID for each request. This ID is used for:
- Logging and tracing
- Error responses
- Request correlation

You can customize the request ID behavior:
- Use `-x-request-id` to use the `X-Request-Id` header if present
- Use `-x-request-id-limit` to limit the length of the `X-Request-Id` header

The request ID appears in:
- Server logs
- Error responses
- Debug output

---

## Response Headers

### Custom Response Headers

You can set custom response headers using the `-header` flag:

```bash
webhook -header "X-Custom-Header=value" -header "X-Another-Header=another-value"
```

These headers will be included in all responses.

### Hook-Specific Response Headers

You can also configure response headers per hook in your hooks configuration:

```json
{
  "id": "my-hook",
  "execute-command": "/path/to/script.sh",
  "response-headers": [
    {
      "name": "X-Custom-Header",
      "value": "custom-value"
    }
  ]
}
```

---

## Rate Limiting

If rate limiting is enabled (via `-rate-limit-enabled`, `-rate-limit-rps`, and `-rate-limit-burst`), the server will return `429 Too Many Requests` when the rate limit is exceeded.

**Response Headers:**
- `X-RateLimit-Limit`: Maximum number of requests allowed
- `X-RateLimit-Remaining`: Number of requests remaining in the current window
- `X-RateLimit-Reset`: Time when the rate limit resets

---

## CORS Support

To enable CORS, use the `-header` flag to set CORS headers:

```bash
webhook -header "Access-Control-Allow-Origin=*" \
        -header "Access-Control-Allow-Methods=GET,POST,OPTIONS" \
        -header "Access-Control-Allow-Headers=Content-Type"
```

---

## Timeouts

The server has configurable timeouts:
- `-read-header-timeout-seconds`: Time to read request headers (default: 5 seconds)
- `-read-timeout-seconds`: Time to read request body (default: 10 seconds)
- `-write-timeout-seconds`: Time to write response (default: 30 seconds)
- `-idle-timeout-seconds`: Time to keep idle connections (default: 90 seconds)
- `-hook-timeout-seconds`: Time for hook execution (default: 30 seconds)

If a timeout occurs, the server will return an appropriate error response.

---

## Streaming Output

If a hook has `stream-command-output` enabled, the command's stdout and stderr are streamed in real-time to the HTTP response. This is useful for long-running commands.

**Example:**
```bash
curl -X POST http://localhost:9000/hooks/long-running-hook
```

The response will stream the command output as it is produced.

---

## Status Codes Summary

| Status Code | Description |
|------------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid request format or parameters |
| 404 | Not Found - Hook ID not found |
| 405 | Method Not Allowed - HTTP method not allowed for this hook |
| 408 | Request Timeout |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error - Server error during execution |
| 503 | Service Unavailable - Server is shutting down |

---

## Best Practices

1. **Use HTTPS**: Always use a reverse proxy (nginx, Traefik, etc.) to provide HTTPS in production.

2. **Set Request Timeouts**: Configure appropriate timeouts based on your use case.

3. **Enable Rate Limiting**: Use rate limiting to prevent abuse.

4. **Use Request IDs**: Include `X-Request-Id` headers for better traceability.

5. **Monitor Metrics**: Use the `/metrics` endpoint for monitoring and alerting.

6. **Handle Errors Gracefully**: Check status codes and parse error responses appropriately.

7. **Use Health Checks**: Monitor the `/health` endpoint for service availability.

