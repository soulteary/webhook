# 配置参数

程序支持两种调用方法，分别是"通过命令行参数"和"设置环境变量"。

关于命令行参数，只需要记得使用 `--help`，即可查看所有的支持设置参数。

而关于环境变量的设置，我们可以通过查看 [internal/flags/define.go](https://github.com/soulteary/webhook/blob/main/internal/flags/define.go) 中的配置项，来完成一致的程序行为设置。

## 命令行参数

以下是所有支持的命令行参数及其说明：

### 基础配置

- `-ip string`
  指定 webhook 服务监听的 IP 地址（默认值：`0.0.0.0`）

- `-port int`
  指定 webhook 服务监听的端口号（默认值：`9000`）

- `-hooks value`
  指定包含钩子定义的 JSON 或 YAML 文件路径，可以多次使用以从不同文件加载钩子

- `-urlprefix string`
  指定钩子 URL 的前缀（格式：`protocol://yourserver:port/PREFIX/:hook-id`，默认值：`hooks`）

### 日志和调试

- `-verbose`
  显示详细输出

- `-debug`
  显示调试输出

- `-logfile string`
  将日志输出发送到文件；启用此选项会自动启用详细日志记录

- `-nopanic`
  当 webhook 未在详细模式下运行时，如果无法加载钩子，不触发 panic

- `-log-request-body`
  在调试模式下记录请求体（默认值：`false`）
  
  **安全警告**：启用此选项可能会暴露敏感数据，仅在调试时使用。

### 功能选项

- `-hotreload`
  监视钩子文件的变化并自动重新加载

- `-template`
  将钩子文件解析为 Go 模板

- `-http-methods string`
  设置默认允许的 HTTP 方法（例如：`"POST"`）；多个方法用逗号分隔

- `-max-multipart-mem int`
  在磁盘缓存之前解析 multipart 表单数据的最大内存（字节，默认值：`1048576`，即 1MB）

- `-max-request-body-size int`
  设置请求体的最大大小（字节，默认值：`10485760`，即 10MB）

### 请求 ID

- `-x-request-id`
  如果存在，使用 `X-Request-Id` 请求头作为请求 ID

- `-x-request-id-limit int`
  截断 `X-Request-Id` 请求头的长度限制；默认无限制

  **说明**：长度限制当前仅能通过 `-x-request-id-limit` 配置；当前实现未提供单独的环境变量。

### 响应头

- `-header value`
  要返回的响应头，格式为 `name=value`，可多次使用以设置多个响应头

### 进程管理

- `-pidfile string`
  在指定路径创建 PID 文件

- `-setuid int`
  在打开监听端口后设置用户 ID；必须与 `setgid` 一起使用

- `-setgid int`
  在打开监听端口后设置组 ID；必须与 `setuid` 一起使用

### 国际化

- `-lang string`
  设置 webhook 的语言代码（默认值：`en-US`）

- `-lang-dir string`
  设置国际化文件的目录（默认值：`./locales`）

### Hook 执行控制

- `-hook-timeout-seconds int`
  设置 hook 执行的默认超时时间（秒，默认值：`30`）
  
  当 hook 执行时间超过此值时，将被强制终止，防止长时间运行的命令占用资源。

- `-max-concurrent-hooks int`
  设置同时执行的最大 hook 数量（默认值：`10`）
  
  用于限制并发执行的 hook 数量，防止资源耗尽。当达到最大并发数时，新的 hook 请求将等待执行槽位。

- `-hook-execution-timeout int`
  设置获取执行槽位的超时时间（秒，默认值：`5`）
  
  当达到最大并发数时，新请求等待执行槽位的最大时间。超过此时间仍未获得执行机会的请求将返回错误。

- `-allow-auto-chmod`
  允许在权限被拒绝时自动修改文件权限（安全风险：默认 `false`）
  
  **警告**：启用此选项会带来安全风险，不建议在生产环境中使用。建议手动设置正确的文件权限（`chmod +x`）。

### 限流配置

以下参数用于配置请求限流，防止服务被过多请求压垮：

- `-rate-limit-enabled`
  启用请求限流（默认值：`false`）

- `-rate-limit-rps int`
  设置每秒允许的请求数（默认值：`100`）

- `-rate-limit-burst int`
  设置突发请求的最大数量（默认值：`10`）

### Redis 分布式限流

以下参数用于配置基于 Redis 的分布式限流，适用于多实例部署：

- `-redis-enabled`
  启用 Redis 分布式限流（默认值：`false`）

- `-redis-addr string`
  Redis 服务器地址（默认值：`localhost:6379`）

- `-redis-password string`
  Redis 密码（默认值：空）

- `-redis-db int`
  Redis 数据库索引（默认值：`0`）

- `-redis-key-prefix string`
  限流键前缀（默认值：`webhook:ratelimit:`）

- `-rate-limit-window int`
  限流时间窗口（秒，默认值：`60`）

### HTTP 服务器超时配置

以下参数用于配置 HTTP 服务器的各种超时时间：

- `-read-header-timeout-seconds int`
  设置读取请求头的超时时间（秒，默认值：`5`）

- `-read-timeout-seconds int`
  设置读取请求体的超时时间（秒，默认值：`10`）

- `-write-timeout-seconds int`
  设置写入响应的超时时间（秒，默认值：`30`）

- `-idle-timeout-seconds int`
  设置空闲连接的超时时间（秒，默认值：`90`）

- `-max-header-bytes int`
  设置请求头的最大大小（字节，默认值：`1048576`，即 1MB）

### 安全配置

以下参数用于增强命令执行的安全性，防止命令注入攻击：

- `-allowed-command-paths string`
  指定允许执行的命令路径白名单（逗号分隔的目录或文件路径列表，默认值：空，表示不进行白名单检查）
  
  当配置此参数后，只有白名单中的命令才能被执行。可以指定目录（如 `/usr/bin`）或具体文件路径。
  
  **示例**：
  ```bash
  # 允许执行 /usr/bin 和 /opt/scripts 目录下的命令
  -allowed-command-paths="/usr/bin,/opt/scripts"
  
  # 允许执行特定文件
  -allowed-command-paths="/usr/bin/git,/opt/scripts/deploy.sh"
  ```

- `-max-arg-length int`
  设置单个命令参数的最大长度（字节，默认值：`1048576`，即 1MB）
  
  用于防止超长参数导致的内存问题。

- `-max-total-args-length int`
  设置所有命令参数的总长度限制（字节，默认值：`10485760`，即 10MB）
  
  用于限制所有参数的总大小，防止内存耗尽。

- `-max-args-count int`
  设置命令参数的最大数量（默认值：`1000`）
  
  用于限制参数数量，防止参数过多导致的性能问题。

- `-strict-mode`
  启用严格模式：拒绝包含潜在危险字符的参数（默认值：`false`）
  
  在严格模式下，包含 shell 特殊字符（如 `;`, `|`, `&`, `` ` ``, `$`, `()`, `{}` 等）的参数将被拒绝执行。

### 分布式追踪

以下参数用于配置 OpenTelemetry 分布式追踪：

- `-tracing-enabled`
  启用分布式追踪（默认值：`false`）

- `-otlp-endpoint string`
  OTLP 导出端点（例如 `localhost:4318`，默认值：空）

- `-tracing-service-name string`
  追踪服务名称（默认值：`webhook`）

### 审计日志

以下参数用于配置审计日志：

- `-audit-enabled`
  启用审计日志（默认值：`false`）

- `-audit-storage-type string`
  审计存储类型：file、redis 或 database（默认值：`file`）

- `-audit-file-path string`
  审计日志文件路径（当存储类型为 file 时，默认值：`./audit.log`）

- `-audit-queue-size int`
  异步写入队列大小（默认值：`1000`）

- `-audit-workers int`
  异步写入工作协程数（默认值：`2`）

- `-audit-mask-ip`
  在审计日志中脱敏 IP 地址（默认值：`true`）

### 其他

- `-version`
  显示 webhook 版本并退出

- `-validate-config`
  验证配置并退出（不启动服务器）
  
  用于检查配置文件和参数是否有效，在部署前进行配置验证。

## 环境变量

所有命令行参数都可以通过环境变量进行设置。环境变量名称与命令行参数对应关系如下：

### 基础配置

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `HOST` | `-ip` | 监听 IP 地址 | `0.0.0.0` |
| `PORT` | `-port` | 监听端口 | `9000` |
| `HOOKS` | `-hooks` | 钩子文件路径（多个用逗号分隔） | - |
| `URL_PREFIX` | `-urlprefix` | URL 前缀 | `hooks` |

### 日志和调试

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `VERBOSE` | `-verbose` | 详细输出 | `false` |
| `DEBUG` | `-debug` | 调试输出 | `false` |
| `LOG_PATH` | `-logfile` | 日志文件路径 | - |
| `NO_PANIC` | `-nopanic` | 不触发 panic | `false` |
| `LOG_REQUEST_BODY` | `-log-request-body` | 记录请求体（调试用） | `false` |

### 功能选项

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `HOT_RELOAD` | `-hotreload` | 热重载 | `false` |
| `TEMPLATE` | `-template` | 模板模式 | `false` |
| `HTTP_METHODS` | `-http-methods` | HTTP 方法 | - |
| `MAX_MPART_MEM` | `-max-multipart-mem` | 最大 multipart 内存 | `1048576` |
| `MAX_REQUEST_BODY_SIZE` | `-max-request-body-size` | 最大请求体大小 | `10485760` |
| `X_REQUEST_ID` | `-x-request-id` | 使用 X-Request-Id | `false` |
| `HEADER` | `-header` | 响应头（格式：`name=value`） | - |

**说明**：X-Request-Id 长度限制仅能通过 `-x-request-id-limit` 配置，无对应环境变量。

### 进程管理

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `PID_FILE` | `-pidfile` | PID 文件路径 | - |
| `UID` | `-setuid` | 用户 ID | `0` |
| `GID` | `-setgid` | 组 ID | `0` |

### 国际化

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `LANGUAGE` | `-lang` | 语言代码 | `en-US` |
| `LANG_DIR` | `-lang-dir` | 国际化文件目录 | `./locales` |

### Hook 执行控制

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `HOOK_TIMEOUT_SECONDS` | `-hook-timeout-seconds` | Hook 执行超时时间（秒） | `30` |
| `MAX_CONCURRENT_HOOKS` | `-max-concurrent-hooks` | 最大并发 hook 数量 | `10` |
| `HOOK_EXECUTION_TIMEOUT` | `-hook-execution-timeout` | 获取执行槽位超时时间（秒） | `5` |
| `ALLOW_AUTO_CHMOD` | `-allow-auto-chmod` | 允许自动修改文件权限 | `false` |

### 限流配置

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `RATE_LIMIT_ENABLED` | `-rate-limit-enabled` | 启用限流 | `false` |
| `RATE_LIMIT_RPS` | `-rate-limit-rps` | 每秒请求数限制 | `100` |
| `RATE_LIMIT_BURST` | `-rate-limit-burst` | 突发请求数限制 | `10` |

### Redis 分布式限流

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `REDIS_ENABLED` | `-redis-enabled` | 启用 Redis 分布式限流 | `false` |
| `REDIS_ADDR` | `-redis-addr` | Redis 服务器地址 | `localhost:6379` |
| `REDIS_PASSWORD` | `-redis-password` | Redis 密码 | （空） |
| `REDIS_DB` | `-redis-db` | Redis 数据库索引 | `0` |
| `REDIS_KEY_PREFIX` | `-redis-key-prefix` | 限流键前缀 | `webhook:ratelimit:` |
| `RATE_LIMIT_WINDOW` | `-rate-limit-window` | 限流时间窗口（秒） | `60` |

### HTTP 服务器超时配置

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `READ_HEADER_TIMEOUT_SECONDS` | `-read-header-timeout-seconds` | 读取请求头超时（秒） | `5` |
| `READ_TIMEOUT_SECONDS` | `-read-timeout-seconds` | 读取请求体超时（秒） | `10` |
| `WRITE_TIMEOUT_SECONDS` | `-write-timeout-seconds` | 写入响应超时（秒） | `30` |
| `IDLE_TIMEOUT_SECONDS` | `-idle-timeout-seconds` | 空闲连接超时（秒） | `90` |
| `MAX_HEADER_BYTES` | `-max-header-bytes` | 最大请求头大小（字节） | `1048576` |

### 安全配置

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `ALLOWED_COMMAND_PATHS` | `-allowed-command-paths` | 允许的命令路径白名单（逗号分隔） | - |
| `MAX_ARG_LENGTH` | `-max-arg-length` | 单个参数最大长度（字节） | `1048576` |
| `MAX_TOTAL_ARGS_LENGTH` | `-max-total-args-length` | 所有参数总长度限制（字节） | `10485760` |
| `MAX_ARGS_COUNT` | `-max-args-count` | 最大参数数量 | `1000` |
| `STRICT_MODE` | `-strict-mode` | 严格模式 | `false` |

### 分布式追踪

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `TRACING_ENABLED` | `-tracing-enabled` | 启用分布式追踪 | `false` |
| `OTLP_ENDPOINT` | `-otlp-endpoint` | OTLP 导出端点 | （空） |
| `TRACING_SERVICE_NAME` | `-tracing-service-name` | 追踪服务名称 | `webhook` |

### 审计日志

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `AUDIT_ENABLED` | `-audit-enabled` | 启用审计日志 | `false` |
| `AUDIT_STORAGE_TYPE` | `-audit-storage-type` | 审计存储类型（file/redis/database） | `file` |
| `AUDIT_FILE_PATH` | `-audit-file-path` | 审计日志文件路径 | `./audit.log` |
| `AUDIT_QUEUE_SIZE` | `-audit-queue-size` | 异步写入队列大小 | `1000` |
| `AUDIT_WORKERS` | `-audit-workers` | 异步写入工作协程数 | `2` |
| `AUDIT_MASK_IP` | `-audit-mask-ip` | 审计日志中脱敏 IP | `true` |

### 环境变量使用示例

```bash
# 基础配置
export HOST=0.0.0.0
export PORT=8080
export HOOKS=/path/to/hooks.json

# 启用详细输出和热重载
export VERBOSE=true
export HOT_RELOAD=true

# 设置语言为中文
export LANGUAGE=zh-CN

# 限流配置
export RATE_LIMIT_ENABLED=true
export RATE_LIMIT_RPS=200
export RATE_LIMIT_BURST=20

# HTTP 服务器超时配置
export READ_HEADER_TIMEOUT_SECONDS=10
export READ_TIMEOUT_SECONDS=30
export WRITE_TIMEOUT_SECONDS=60

# 安全配置
export ALLOWED_COMMAND_PATHS="/usr/bin,/opt/scripts"
export MAX_ARG_LENGTH=1048576
export STRICT_MODE=true

# 运行 webhook
./webhook
```

## 热重载钩子

如果你运行的操作系统支持 HUP 或 USR1 信号，可以使用这些信号来触发钩子重新加载，而无需重启 webhook 实例：

```bash
# 使用 USR1 信号
kill -USR1 <webhook_pid>

# 使用 HUP 信号
kill -HUP <webhook_pid>
```

或者，你可以使用 `-hotreload` 参数（或设置 `HOT_RELOAD=true` 环境变量）来启用自动热重载功能。启用后，webhook 会自动监视钩子文件的变化并重新加载。

## 优先级说明

当同时使用命令行参数和环境变量时，**命令行参数的优先级更高**。配置解析顺序为：

1. 首先读取默认值
2. 然后读取环境变量（覆盖默认值）
3. 最后读取命令行参数（覆盖环境变量）

这使得你可以在环境变量中设置基础配置，然后通过命令行参数进行临时覆盖。
