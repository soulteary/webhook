# Webhook Parameters

This document describes all available command-line parameters and environment variables for configuring webhook.

## Command-Line Parameters

### Basic Configuration

| Flag | Description | Default |
|------|-------------|---------|
| `-ip string` | IP address the webhook should serve hooks on | `0.0.0.0` |
| `-port int` | Port the webhook should serve hooks on | `9000` |
| `-hooks value` | Path to the JSON/YAML file containing hook definitions (can be used multiple times) | - |
| `-urlprefix string` | URL prefix for served hooks (protocol://yourserver:port/PREFIX/:hook-id) | `hooks` |

### Logging and Debugging

| Flag | Description | Default |
|------|-------------|---------|
| `-verbose` | Show verbose output | `false` |
| `-debug` | Show debug output | `false` |
| `-logfile string` | Send log output to a file; implicitly enables verbose logging | - |
| `-nopanic` | Do not panic if hooks cannot be loaded when not in verbose mode | `false` |
| `-log-request-body` | Log request body in debug mode (SECURITY: may expose sensitive data) | `false` |

### Feature Options

| Flag | Description | Default |
|------|-------------|---------|
| `-hotreload` | Watch hooks file for changes and reload automatically | `false` |
| `-template` | Parse hooks file as a Go template | `false` |
| `-http-methods string` | Set default allowed HTTP methods (e.g., "POST"); separate with comma | - |
| `-max-multipart-mem int` | Maximum memory in bytes for parsing multipart form data before disk caching | `1048576` (1MB) |
| `-max-request-body-size int` | Maximum size in bytes for request body | `10485760` (10MB) |

### Request ID

| Flag | Description | Default |
|------|-------------|---------|
| `-x-request-id` | Use X-Request-Id header, if present, as request ID | `false` |
| `-x-request-id-limit int` | Truncate X-Request-Id header to limit | `0` (no limit) |

### Response Headers

| Flag | Description | Default |
|------|-------------|---------|
| `-header value` | Response header to return (format: name=value); can be used multiple times | - |

### Process Management

| Flag | Description | Default |
|------|-------------|---------|
| `-pidfile string` | Create PID file at the given path | - |
| `-setuid int` | Set user ID after opening listening port; must be used with setgid | `0` |
| `-setgid int` | Set group ID after opening listening port; must be used with setuid | `0` |

### Internationalization

| Flag | Description | Default |
|------|-------------|---------|
| `-lang string` | Set the language code for the webhook | `en-US` |
| `-lang-dir string` | Set the directory for i18n files | `./locales` |

### Hook Execution Control

| Flag | Description | Default |
|------|-------------|---------|
| `-hook-timeout-seconds int` | Default timeout in seconds for hook execution | `30` |
| `-max-concurrent-hooks int` | Maximum number of concurrent hook executions | `10` |
| `-hook-execution-timeout int` | Timeout in seconds for acquiring execution slot when max concurrent hooks reached | `5` |
| `-allow-auto-chmod` | Allow automatically modifying file permissions when permission denied (SECURITY RISK) | `false` |

### Rate Limiting

| Flag | Description | Default |
|------|-------------|---------|
| `-rate-limit-enabled` | Enable rate limiting | `false` |
| `-rate-limit-rps int` | Rate limit requests per second | `100` |
| `-rate-limit-burst int` | Rate limit burst size | `10` |

### HTTP Server Timeouts

| Flag | Description | Default |
|------|-------------|---------|
| `-read-header-timeout-seconds int` | Timeout in seconds for reading request headers | `5` |
| `-read-timeout-seconds int` | Timeout in seconds for reading request body | `10` |
| `-write-timeout-seconds int` | Timeout in seconds for writing response | `30` |
| `-idle-timeout-seconds int` | Timeout in seconds for idle connections | `90` |
| `-max-header-bytes int` | Maximum size in bytes for request headers | `1048576` (1MB) |

### Security Configuration

| Flag | Description | Default |
|------|-------------|---------|
| `-allowed-command-paths string` | Comma-separated list of allowed command paths for whitelist; empty means no check | - |
| `-max-arg-length int` | Maximum length for a single command argument in bytes | `1048576` (1MB) |
| `-max-total-args-length int` | Maximum total length for all command arguments in bytes | `10485760` (10MB) |
| `-max-args-count int` | Maximum number of command arguments | `1000` |
| `-strict-mode` | Reject arguments containing potentially dangerous characters | `false` |

### Other

| Flag | Description | Default |
|------|-------------|---------|
| `-version` | Display webhook version and quit | - |
| `-validate-config` | Validate configuration and exit (does not start server) | - |

## Environment Variables

All command-line parameters can also be set via environment variables:

### Basic Configuration

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `HOST` | `-ip` | Listen IP address | `0.0.0.0` |
| `PORT` | `-port` | Listen port | `9000` |
| `HOOKS` | `-hooks` | Hook file paths (comma-separated) | - |
| `URL_PREFIX` | `-urlprefix` | URL prefix | `hooks` |

### Logging and Debugging

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `VERBOSE` | `-verbose` | Verbose output | `false` |
| `DEBUG` | `-debug` | Debug output | `false` |
| `LOG_PATH` | `-logfile` | Log file path | - |
| `NO_PANIC` | `-nopanic` | Don't panic | `false` |
| `LOG_REQUEST_BODY` | `-log-request-body` | Log request body (debug) | `false` |

### Feature Options

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `HOT_RELOAD` | `-hotreload` | Hot reload | `false` |
| `TEMPLATE` | `-template` | Template mode | `false` |
| `HTTP_METHODS` | `-http-methods` | HTTP methods | - |
| `MAX_MPART_MEM` | `-max-multipart-mem` | Max multipart memory | `1048576` |
| `MAX_REQUEST_BODY_SIZE` | `-max-request-body-size` | Max request body size | `10485760` |
| `X_REQUEST_ID` | `-x-request-id` | Use X-Request-Id | `false` |
| `X_REQUEST_ID_LIMIT` | `-x-request-id-limit` | X-Request-Id limit | `0` |
| `HEADER` | `-header` | Response header | - |

### Process Management

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `PID_FILE` | `-pidfile` | PID file path | - |
| `UID` | `-setuid` | User ID | `0` |
| `GID` | `-setgid` | Group ID | `0` |

### Internationalization

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `LANGUAGE` | `-lang` | Language code | `en-US` |
| `LANG_DIR` | `-lang-dir` | i18n directory | `./locales` |

### Hook Execution Control

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `HOOK_TIMEOUT_SECONDS` | `-hook-timeout-seconds` | Hook execution timeout (sec) | `30` |
| `MAX_CONCURRENT_HOOKS` | `-max-concurrent-hooks` | Max concurrent hooks | `10` |
| `HOOK_EXECUTION_TIMEOUT` | `-hook-execution-timeout` | Execution slot timeout (sec) | `5` |
| `ALLOW_AUTO_CHMOD` | `-allow-auto-chmod` | Allow auto chmod | `false` |

### Rate Limiting

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `RATE_LIMIT_ENABLED` | `-rate-limit-enabled` | Enable rate limiting | `false` |
| `RATE_LIMIT_RPS` | `-rate-limit-rps` | Requests per second | `100` |
| `RATE_LIMIT_BURST` | `-rate-limit-burst` | Burst size | `10` |

### HTTP Server Timeouts

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `READ_HEADER_TIMEOUT_SECONDS` | `-read-header-timeout-seconds` | Read header timeout (sec) | `5` |
| `READ_TIMEOUT_SECONDS` | `-read-timeout-seconds` | Read body timeout (sec) | `10` |
| `WRITE_TIMEOUT_SECONDS` | `-write-timeout-seconds` | Write response timeout (sec) | `30` |
| `IDLE_TIMEOUT_SECONDS` | `-idle-timeout-seconds` | Idle connection timeout (sec) | `90` |
| `MAX_HEADER_BYTES` | `-max-header-bytes` | Max header size (bytes) | `1048576` |

### Security Configuration

| Environment Variable | CLI Flag | Description | Default |
|---------------------|----------|-------------|---------|
| `ALLOWED_COMMAND_PATHS` | `-allowed-command-paths` | Allowed command paths (comma-separated) | - |
| `MAX_ARG_LENGTH` | `-max-arg-length` | Max single argument length (bytes) | `1048576` |
| `MAX_TOTAL_ARGS_LENGTH` | `-max-total-args-length` | Max total arguments length (bytes) | `10485760` |
| `MAX_ARGS_COUNT` | `-max-args-count` | Max argument count | `1000` |
| `STRICT_MODE` | `-strict-mode` | Strict mode | `false` |

## Security Best Practices

### Command Path Whitelisting

Use `-allowed-command-paths` to restrict which commands can be executed:

```bash
# Allow commands from specific directories
-allowed-command-paths="/usr/bin,/opt/scripts"

# Allow specific files only
-allowed-command-paths="/usr/bin/git,/opt/scripts/deploy.sh"
```

### Strict Mode

Enable `-strict-mode` to reject arguments containing potentially dangerous shell characters (`;`, `|`, `&`, `` ` ``, `$`, `()`, `{}`, etc.).

### Rate Limiting

Enable rate limiting to protect against DoS attacks:

```bash
-rate-limit-enabled -rate-limit-rps=100 -rate-limit-burst=10
```

## Configuration Priority

When using both environment variables and command-line flags, **command-line flags take priority**. The resolution order is:

1. Default values
2. Environment variables (override defaults)
3. Command-line flags (override environment variables)

## Live Reloading Hooks

If your OS supports HUP or USR1 signals, you can use them to trigger hook reload without restarting:

```bash
kill -USR1 <webhook_pid>
# or
kill -HUP <webhook_pid>
```

Alternatively, use `-hotreload` (or `HOT_RELOAD=true`) for automatic hot reloading when hook files change.

## Example Usage

```bash
# Basic usage
./webhook -hooks hooks.json -verbose

# With security settings
./webhook -hooks hooks.json \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -rate-limit-enabled \
  -rate-limit-rps=100

# With HTTP server tuning
./webhook -hooks hooks.json \
  -read-header-timeout-seconds=10 \
  -read-timeout-seconds=30 \
  -write-timeout-seconds=60 \
  -max-concurrent-hooks=20

# Using environment variables
export PORT=8080
export HOOKS=/path/to/hooks.json
export RATE_LIMIT_ENABLED=true
./webhook
```
