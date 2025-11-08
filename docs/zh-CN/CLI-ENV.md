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

### 功能选项

- `-hotreload`
  监视钩子文件的变化并自动重新加载

- `-template`
  将钩子文件解析为 Go 模板

- `-http-methods string`
  设置默认允许的 HTTP 方法（例如：`"POST"`）；多个方法用逗号分隔

- `-max-multipart-mem int`
  在磁盘缓存之前解析 multipart 表单数据的最大内存（字节，默认值：`1048576`，即 1MB）

### 请求 ID

- `-x-request-id`
  如果存在，使用 `X-Request-Id` 请求头作为请求 ID

- `-x-request-id-limit int`
  截断 `X-Request-Id` 请求头的长度限制；默认无限制

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

### 其他

- `-version`
  显示 webhook 版本并退出

## 环境变量

所有命令行参数都可以通过环境变量进行设置。环境变量名称与命令行参数对应关系如下：

| 环境变量 | 命令行参数 | 说明 | 默认值 |
|---------|-----------|------|--------|
| `HOST` | `-ip` | 监听 IP 地址 | `0.0.0.0` |
| `PORT` | `-port` | 监听端口 | `9000` |
| `HOOKS` | `-hooks` | 钩子文件路径（多个用逗号分隔） | - |
| `URL_PREFIX` | `-urlprefix` | URL 前缀 | `hooks` |
| `VERBOSE` | `-verbose` | 详细输出 | `false` |
| `DEBUG` | `-debug` | 调试输出 | `false` |
| `LOG_PATH` | `-logfile` | 日志文件路径 | - |
| `NO_PANIC` | `-nopanic` | 不触发 panic | `false` |
| `HOT_RELOAD` | `-hotreload` | 热重载 | `false` |
| `TEMPLATE` | `-template` | 模板模式 | `false` |
| `HTTP_METHODS` | `-http-methods` | HTTP 方法 | - |
| `MAX_MPART_MEM` | `-max-multipart-mem` | 最大 multipart 内存 | `1048576` |
| `X_REQUEST_ID` | `-x-request-id` | 使用 X-Request-Id | `false` |
| `X_REQUEST_ID_LIMIT` | `-x-request-id-limit` | X-Request-Id 长度限制 | `0` |
| `HEADER` | `-header` | 响应头（格式：`name=value`） | - |
| `PID_FILE` | `-pidfile` | PID 文件路径 | - |
| `UID` | `-setuid` | 用户 ID | `0` |
| `GID` | `-setgid` | 组 ID | `0` |
| `LANGUAGE` | `-lang` | 语言代码 | `en-US` |
| `LANG_DIR` | `-lang-dir` | 国际化文件目录 | `./locales` |

### 环境变量使用示例

```bash
# 设置监听端口
export PORT=8080

# 设置钩子文件路径
export HOOKS=/path/to/hooks.json

# 启用详细输出
export VERBOSE=true

# 启用热重载
export HOT_RELOAD=true

# 设置语言为中文
export LANGUAGE=zh-CN

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

当同时使用命令行参数和环境变量时，命令行参数的优先级更高。程序会先读取环境变量，然后用命令行参数覆盖相应的值。
