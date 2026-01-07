# API 参考文档

本文档描述了 Webhook 服务器提供的所有 HTTP 端点。

## 基础 URL

默认情况下，Webhook 服务器运行在端口 `9000`。基础 URL 格式为：

```
http://your-server:9000
```

您可以使用 `-ip` 和 `-port` 命令行参数或环境变量来自定义 IP 地址和端口。

## 端点

### 1. 根端点

**端点:** `GET /`

**描述:** 简单的健康检查端点，返回 "OK"。

**响应:**
- **状态码:** `200 OK`
- **Content-Type:** `text/plain`
- **响应体:** `OK`

**示例:**
```bash
curl http://localhost:9000/
```

---

### 2. 健康检查端点

**端点:** `GET /health`

**描述:** 健康检查端点，以 JSON 格式返回服务器状态。

**响应:**
- **状态码:** `200 OK`
- **Content-Type:** `application/json`
- **响应体:**
```json
{
  "status": "ok"
}
```

**示例:**
```bash
curl http://localhost:9000/health
```

---

### 3. 指标端点

**端点:** `GET /metrics`

**描述:** Prometheus 指标端点，用于监控和可观测性。

**响应:**
- **状态码:** `200 OK`
- **Content-Type:** `text/plain; version=0.0.4; charset=utf-8`
- **响应体:** Prometheus 格式的文本指标

**示例:**
```bash
curl http://localhost:9000/metrics
```

**可用指标:**
- `webhook_http_requests_total`: HTTP 请求总数
- `webhook_http_request_duration_seconds`: HTTP 请求持续时间直方图
- `webhook_hook_executions_total`: Hook 执行总数
- `webhook_hook_execution_duration_seconds`: Hook 执行持续时间直方图
- `webhook_system_memory_bytes`: 系统内存使用量
- `webhook_system_cpu_percent`: 系统 CPU 使用百分比

---

### 4. Hook 执行端点

**端点:** `POST|GET|PUT|DELETE /hooks/{hook-id}`

**描述:** 执行配置的 hook。允许的 HTTP 方法取决于 hook 配置和 `-http-methods` 标志。

**URL 参数:**
- `hook-id` (必需): 要执行的 hook 的 ID，在您的 hooks 配置文件中定义。

**请求头:**
- `Content-Type`: 可选。可以是 `application/json`、`application/x-www-form-urlencoded`、`multipart/form-data` 或 `text/plain`。
- `X-Request-Id`: 可选。如果提供且启用了 `-x-request-id`，将用作日志记录的请求 ID。

**请求体:**
请求体可以包含：
- JSON 数据
- 表单数据（URL 编码或 multipart）
- 纯文本
- 查询参数（用于 GET 请求）

**响应:**
- **状态码:** 
  - `200 OK`: Hook 执行成功
  - `400 Bad Request`: 无效请求（例如，格式错误的 JSON、缺少必需参数）
  - `404 Not Found`: 未找到 Hook ID
  - `405 Method Not Allowed`: 此 hook 不允许的 HTTP 方法
  - `408 Request Timeout`: 请求超时
  - `429 Too Many Requests`: 超过速率限制（如果启用了速率限制）
  - `500 Internal Server Error`: Hook 执行期间的服务器错误
  - `503 Service Unavailable`: 服务器正在关闭
  - 自定义状态码: 在 `success-http-response-code` 或 `trigger-rule-mismatch-http-response-code` 中配置

- **Content-Type:** 
  - `text/plain` (默认)
  - `application/json` (如果发生错误)
  - 在 `response-headers` 中配置

- **响应体:**
  - 成功: 来自 `response-message` 的自定义消息、命令输出（如果启用了 `include-command-output-in-response`）或默认消息
  - 错误: 包含详细信息的 JSON 错误响应

**错误响应格式:**
```json
{
  "error": "错误类型",
  "message": "错误消息",
  "request_id": "请求-id-这里",
  "hook_id": "hook-id-这里"
}
```

**示例 - 成功执行:**
```bash
# 带 JSON 体的 POST 请求
curl -X POST http://localhost:9000/hooks/redeploy-webhook \
  -H "Content-Type: application/json" \
  -d '{"branch": "main", "commit": "abc123"}'

# 带查询参数的 GET 请求
curl "http://localhost:9000/hooks/redeploy-webhook?branch=main&commit=abc123"
```

**示例 - Hook 未找到:**
```bash
curl -X POST http://localhost:9000/hooks/non-existent-hook
```

**响应:**
```json
{
  "error": "Not Found",
  "message": "Hook not found.",
  "request_id": "req-123",
  "hook_id": "non-existent-hook"
}
```

**示例 - 方法不允许:**
```bash
# 如果 hook 只允许 POST，但我们发送 GET
curl -X GET http://localhost:9000/hooks/post-only-hook
```

**响应:**
```json
{
  "error": "Method Not Allowed",
  "message": "HTTP GET method not allowed for hook \"post-only-hook\"",
  "request_id": "req-456",
  "hook_id": "post-only-hook"
}
```

---

## 请求 ID

Webhook 自动为每个请求生成唯一的请求 ID。此 ID 用于：
- 日志记录和追踪
- 错误响应
- 请求关联

您可以自定义请求 ID 行为：
- 使用 `-x-request-id` 在存在时使用 `X-Request-Id` 请求头
- 使用 `-x-request-id-limit` 限制 `X-Request-Id` 请求头的长度

请求 ID 出现在：
- 服务器日志
- 错误响应
- 调试输出

---

## 响应头

### 自定义响应头

您可以使用 `-header` 标志设置自定义响应头：

```bash
webhook -header "X-Custom-Header=value" -header "X-Another-Header=another-value"
```

这些响应头将包含在所有响应中。

### Hook 特定的响应头

您还可以在 hooks 配置中为每个 hook 配置响应头：

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

## 速率限制

如果启用了速率限制（通过 `-rate-limit-enabled`、`-rate-limit-rps` 和 `-rate-limit-burst`），当超过速率限制时，服务器将返回 `429 Too Many Requests`。

**响应头:**
- `X-RateLimit-Limit`: 允许的最大请求数
- `X-RateLimit-Remaining`: 当前窗口中剩余的请求数
- `X-RateLimit-Reset`: 速率限制重置的时间

---

## CORS 支持

要启用 CORS，使用 `-header` 标志设置 CORS 响应头：

```bash
webhook -header "Access-Control-Allow-Origin=*" \
        -header "Access-Control-Allow-Methods=GET,POST,OPTIONS" \
        -header "Access-Control-Allow-Headers=Content-Type"
```

---

## 超时

服务器具有可配置的超时：
- `-read-header-timeout-seconds`: 读取请求头的时间（默认：5 秒）
- `-read-timeout-seconds`: 读取请求体的时间（默认：10 秒）
- `-write-timeout-seconds`: 写入响应的时间（默认：30 秒）
- `-idle-timeout-seconds`: 保持空闲连接的时间（默认：90 秒）
- `-hook-timeout-seconds`: Hook 执行时间（默认：30 秒）

如果发生超时，服务器将返回适当的错误响应。

---

## 流式输出

如果 hook 启用了 `stream-command-output`，命令的 stdout 和 stderr 会实时流式传输到 HTTP 响应。这对于长时间运行的命令很有用。

**示例:**
```bash
curl -X POST http://localhost:9000/hooks/long-running-hook
```

响应将流式传输命令输出。

---

## 状态码摘要

| 状态码 | 描述 |
|--------|------|
| 200 | 成功 |
| 400 | 错误请求 - 无效的请求格式或参数 |
| 404 | 未找到 - 未找到 Hook ID |
| 405 | 方法不允许 - 此 hook 不允许的 HTTP 方法 |
| 408 | 请求超时 |
| 429 | 请求过多 - 超过速率限制 |
| 500 | 内部服务器错误 - 执行期间的服务器错误 |
| 503 | 服务不可用 - 服务器正在关闭 |

---

## 最佳实践

1. **使用 HTTPS**: 在生产环境中始终使用反向代理（nginx、Traefik 等）提供 HTTPS。

2. **设置请求超时**: 根据您的用例配置适当的超时。

3. **启用速率限制**: 使用速率限制防止滥用。

4. **使用请求 ID**: 包含 `X-Request-Id` 请求头以获得更好的可追踪性。

5. **监控指标**: 使用 `/metrics` 端点进行监控和告警。

6. **优雅处理错误**: 检查状态码并适当解析错误响应。

7. **使用健康检查**: 监控 `/health` 端点以了解服务可用性。

