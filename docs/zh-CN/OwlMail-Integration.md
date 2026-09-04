# WebHook 与 OwlMail 联动

[OwlMail](https://github.com/soulteary/owlmail) 可以把接收邮件事件转发给
WebHook。WebHook 对完整请求体进行鉴权，将指定 JSON 字段映射为环境变量，再执行
受控的本地命令。

```text
SMTP 客户端 -> OwlMail v0.9.0 -> 带签名的 HTTP POST -> WebHook -> 命令/脚本
```

可运行示例位于 [`example/owlmail`](../../example/owlmail/)。OwlMail 仓库也维护了
发送端示例：[`examples/webhooks/soulteary-webhook`](https://github.com/soulteary/owlmail/tree/v0.9.0/examples/webhooks/soulteary-webhook)。

## 使用要求

- WebHook 7.1.0 或更高版本。
- OwlMail v0.9.0（本示例固定使用的版本）或更高版本。
- 至少 32 字节的随机共享密钥。
- OwlMail 能够访问 WebHook 端点。

## 1. 配置 WebHook

可以从 [`hooks.json.tmpl`](../../example/owlmail/hooks.json.tmpl) 开始。它定义了
`POST /hooks/owlmail`，要求 `application/json`，验证 `X-OwlMail-Signature`，并将
`event`、`emailId`、`title`、`message`、`from`、`to`、`receivedAt` 映射到固定的
`OWLMAIL_*` 环境变量；同时把 `X-OwlMail-Delivery-ID` 请求头映射为用于幂等处理的
`OWLMAIL_DELIVERY_ID`。

```bash
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
export OWLMAIL_WEBHOOK_COMMAND=/opt/owlmail/handle-email.sh
export OWLMAIL_WEBHOOK_WORKDIR=/opt/owlmail

webhook \
  -hooks hooks.json.tmpl \
  -template \
  -allowed-command-paths=/opt/owlmail/handle-email.sh \
  -strict-mode
```

示例启用了 `include-command-output-in-response`，命令失败会返回非 2xx，使 OwlMail
按策略重试。处理脚本应保持幂等，并避免输出密钥或不必要的完整邮件正文。

## 2. 配置 OwlMail v0.9.0

打开 OwlMail 的 `/webhooks` 页面，可以创建、导入、校验、复制和下载 version 1
配置。编辑过程发生在浏览器本地；下载配置并不会自动激活它，需要把 JSON 文件挂载
到 OwlMail 并通过 `-webhook-config` 或 `OWLMAIL_WEBHOOK_CONFIG` 指定。

Compose 示例直接使用发布镜像：

```yaml
owlmail:
  image: ghcr.io/soulteary/owlmail:0.9.0
  environment:
    OWLMAIL_WEBHOOK_CONFIG: /app/config/owlmail.json
    OWLMAIL_WEBHOOK_MAX_CONCURRENCY: "8"
```

`OWLMAIL_WEBHOOK_MAX_CONCURRENCY` 默认值为 `8`，用于限制进程级 Webhook 并发投递；
只有明确希望不限制并发时才设置为 `0`。

建议目标配置：URL 使用 `http://webhook:9000/hooks/owlmail`，方法 `POST`，内容类型
`application/json`，共享密钥与 WebHook 一致，超时略高于正常命令耗时，重试次数保持
较小且有界。示例使用 `10s` 超时和 `2` 次重试。

运行时可以这样启用：

```bash
export OWLMAIL_WEBHOOK_URL=http://127.0.0.1:9000/hooks/owlmail
export OWLMAIL_WEBHOOK_SECRET='替换为两边一致的随机密钥'
owlmail -webhook-config ./owlmail.json
```

OwlMail 会先展开环境变量，再验证配置；缺少变量或数值无效时启动失败，不会静默关闭
消息投递。

## 签名与投递标识

目标配置了 `secret` 时，OwlMail 0.9.0 会同时发送两种签名：

| 请求头 | 含义 |
|---|---|
| `X-OwlMail-Signature` | 对原始请求体计算的旧版 HMAC-SHA256：`sha256=<十六进制摘要>`。 |
| `X-OwlMail-Signature-V2` | 对 `时间戳 + "." + nonce + "." + 请求体` 计算的防重放 HMAC：`v2=<十六进制摘要>`。 |
| `X-OwlMail-Timestamp` | UTC RFC 3339 签名时间。 |
| `X-OwlMail-Nonce` | 每次 HTTP 尝试都会生成的新随机值。 |
| `X-OwlMail-Delivery-ID` | 同一个队列投递在重试期间保持稳定的标识。 |

本示例使用 WebHook 内置的 `payload-hmac-sha256` 规则，只验证旧版的正文签名。
它可以防止请求体被修改，但不会验证时间戳、nonce 或投递 ID，因此本身不具备
防重放能力。需要防重放时，应在经过认证的反向代理或专用处理器中验证 V2 签名，
限制时间窗口，并拒绝重复 nonce。

执行非幂等副作用前，应使用 `OWLMAIL_DELIVERY_ID` 去重。邮件 ID 标识一封已保存
邮件，而同一邮件未来可能产生多个事件或投递。仅使用旧版签名规则时，不能把
投递 ID 请求头视为已经被密码学绑定。

## 投递时序与持久性

OwlMail 的 SMTP 接收与最终 HTTP 结果相互独立。0.9.0 会先把每个事件同步写入
邮件目录下的 `.owlmail-webhook-outbox`，随后异步执行 HTTP 投递，并采用
“至少一次”语义。WebHook 命令失败会让本次投递返回非 2xx 并触发重试，但不会
拒收或删除已经保存的邮件。

需要跨重启恢复投递时，应使用持久化邮件目录，并配置 Redis 6.2 或更高版本：

```bash
export OWLMAIL_MAIL_DIR=/app/mail
export OWLMAIL_WEBHOOK_REDIS_URL=redis://redis:6379/0
export OWLMAIL_WEBHOOK_REDIS_PREFIX=owlmail:webhooks
export OWLMAIL_WEBHOOK_SHUTDOWN_TIMEOUT=15s
```

未配置 Redis 时，本地 outbox 只保护内存队列接收之前的任务；已经进入内存的任务
以及重试耗尽的投递不会在重启后重放。配置 Redis 后，可以重新认领 pending 任务，
重试耗尽的投递会进入死信 Stream。每个 Redis 前缀应只运行一个活动 OwlMail 实例。

## 一键 Demo

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

随后通过 `curl smtp://127.0.0.1:1025` 发送测试邮件。访问 `http://127.0.0.1:1080`
查看邮件，访问 `http://127.0.0.1:1080/webhooks` 查看 v0.9.0 的 Webhook 配置生成器，
WebHook 容器日志中可以看到经过签名校验和字段映射后的 Demo 输出。

## 生产环境检查清单

- WebHook 放在私有网络或带认证的 TLS 反向代理之后，并保留 HMAC。
- 内置旧版 HMAC 规则只能验证正文，不等同于防重放；有防重放要求时应验证 OwlMail V2 签名。
- `-allowed-command-paths` 只允许明确脚本，启用 `-strict-mode`、并发限制、执行超时和限流。
- OwlMail 保持默认 `8` 或按下游容量设置 `OWLMAIL_WEBHOOK_MAX_CONCURRENCY`，不要无意设置为 `0`。
- 邮件链路不要开启原始请求体日志；命令使用非特权用户执行。
- 使用 `OWLMAIL_DELIVERY_ID` 对重试去重，而不是只依赖邮件 ID。
- 需要跨重启恢复时，持久化 OwlMail 邮件目录并配置 Redis。
- OwlMail 单次投递超时应略高于命令正常耗时但保持有界，同时设置有限的退出排空时间。

## 过滤与多工作流

OwlMail v0.9.0 支持按发件人、收件人、主题和纯文本正文进行大小写不敏感的通配符过滤。多个目标可
分别调用 `/hooks/owlmail-archive`、`/hooks/owlmail-alert`、`/hooks/owlmail-ticket`
等 Hook，并使用独立命令和密钥隔离权限与失败处理。

## 故障排查

| 现象 | 检查内容 |
|---|---|
| OwlMail 启动失败 | 配置路径、展开后的 URL/密钥、时长、模板和 JSON 语法。 |
| WebHook 返回 401/403 | 两边密钥是否一致，代理是否修改请求体或签名头。 |
| WebHook 返回 404 | 目标 URL 是否以 `/hooks/owlmail` 结尾，Hook ID 是否为 `owlmail`。 |
| OwlMail 遇到 5xx 后重试 | 检查命令退出状态、WebHook 超时及并发日志。 |
| 同一事件重复处理 | 使用 `OWLMAIL_DELIVERY_ID` 作为幂等键；投递语义是“至少一次”。 |
| 重启后事件丢失 | 持久化邮件目录并配置 Redis；本 Demo 有意没有配置这两项。 |
| 需要防重放 | 内置规则只校验旧版正文 HMAC；需单独验证 V2 时间戳、nonce、签名与重放窗口。 |

[English](../en-US/OwlMail-Integration.md)
