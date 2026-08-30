# WebHook 与 OwlMail 联动

[OwlMail](https://github.com/soulteary/owlmail) 可以把接收邮件事件转发给
WebHook。WebHook 对完整请求体进行鉴权，将指定 JSON 字段映射为环境变量，再执行
受控的本地命令。

```text
SMTP 客户端 -> OwlMail -> 带签名的 HTTP POST -> WebHook -> 命令/脚本
```

可运行示例位于 [`example/owlmail`](../../example/owlmail/)。OwlMail 仓库也维护了
发送端示例：
[`examples/webhooks/soulteary-webhook`](https://github.com/soulteary/owlmail/tree/main/examples/webhooks/soulteary-webhook)。

## 使用要求

- WebHook 7.0.0 或更高版本。
- 包含 Webhook 转发功能的 OwlMail。目前已发布的 OwlMail `v0.4.0` 镜像尚不
  包含该功能，需要构建当前 `main`，或使用后续版本。
- 至少 32 字节的随机共享密钥。
- OwlMail 能够访问 WebHook 端点。

## 1. 配置 WebHook

可以从 [`hooks.json.tmpl`](../../example/owlmail/hooks.json.tmpl) 开始。它定义了
`POST /hooks/owlmail`，要求 `application/json`，验证
`X-OwlMail-Signature`，并进行以下字段映射：

| JSON 字段 | 命令环境变量 |
|---|---|
| `event` | `OWLMAIL_EVENT` |
| `emailId` | `OWLMAIL_EMAIL_ID` |
| `title` | `OWLMAIL_TITLE` |
| `message` | `OWLMAIL_MESSAGE` |
| `from` | `OWLMAIL_FROM` |
| `to` | `OWLMAIL_TO` |
| `receivedAt` | `OWLMAIL_RECEIVED_AT` |

该配置使用 Go 模板，使密钥和命令路径不进入版本库：

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

示例启用了 `include-command-output-in-response`，WebHook 会等待命令结束；命令
执行失败会返回非 2xx，让 OwlMail 按策略重试。成功时命令输出会返回给 OwlMail，
因此脚本不应输出密钥或完整邮件内容，除非这是明确需要的行为。

## 2. 配置 OwlMail

在包含配置页面的 OwlMail 构建中，打开 `/webhooks`，创建目标并下载
`webhooks.json`；也可以直接从
[`owlmail.json`](../../example/owlmail/owlmail.json) 修改。

建议配置如下：

| 配置项 | 数值 |
|---|---|
| URL | Compose 中使用 `http://webhook:9000/hooks/owlmail`，其他环境填写 WebHook 可达地址 |
| 方法 | `POST` |
| Content-Type | `application/json` |
| 密钥 | 与 `OWLMAIL_WEBHOOK_SECRET` 完全相同 |
| 超时 | 大于命令正常耗时；示例为 `10s` |
| 重试 | 使用较小的有界值；示例为 `2` |

示例请求体模板只发送联动所需字段，不会直接转发完整的内部邮件对象：

```json
{
  "event": "email.received",
  "emailId": "...",
  "title": "...",
  "message": "...",
  "from": "...",
  "to": "...",
  "receivedAt": "..."
}
```

挂载下载的配置文件，并通过参数或环境变量启用：

```bash
export OWLMAIL_WEBHOOK_URL=http://127.0.0.1:9000/hooks/owlmail
export OWLMAIL_WEBHOOK_SECRET='替换为两边一致的随机密钥'
owlmail -webhook-config ./owlmail.json
```

或者：

```bash
export OWLMAIL_WEBHOOK_CONFIG=/app/config/owlmail.json
```

OwlMail 会先展开环境变量，再验证配置；缺少变量或数值无效时会启动失败，不会
静默关闭消息投递。

## HMAC 约定

OwlMail 对每次 HTTP 请求的原始请求体计算 HMAC-SHA256，并发送十六进制摘要：

```text
X-OwlMail-Signature: sha256=<十六进制摘要>
```

WebHook 的 `payload-hmac-sha256` 规则使用同一密钥校验完全相同的原始字节。链路
中的代理不能改写或重新压缩请求体。签名不匹配时，WebHook 返回非 2xx 且不执行
命令。

## 生产环境检查清单

- 将 WebHook 放在私有网络，或置于带身份认证的反向代理之后。
- 即使使用容器私有网络也保留 HMAC；不要把共享密钥提交到仓库。
- 尽量让 `-allowed-command-paths` 只包含明确的脚本路径。
- 启用 `-strict-mode`、有界并发、执行超时与速率限制。
- 邮件请求不要启用 `-debug` 和 `-log-request-body`。
- 使用非特权用户运行命令，并以只读方式挂载脚本和配置。
- 处理脚本必须幂等：OwlMail 会重试失败投递，同一邮件事件可能被处理多次。
- OwlMail 超时应高于命令正常耗时，但不能无限延长 SMTP/事件链路等待时间。

## 过滤与多工作流

OwlMail 可以按发件人、收件人、主题和正文模式过滤目标。只需要处理部分邮件时，
优先在发送端过滤。多个 OwlMail 目标可以调用不同 Hook ID，例如：

- `/hooks/owlmail-archive`：处理全部邮件；
- `/hooks/owlmail-alert`：仅处理主题匹配 `Critical*` 的邮件；
- `/hooks/owlmail-ticket`：仅处理发送到 `support@example.com` 的邮件。

每个端点应使用独立命令；条件允许时也使用独立密钥，以隔离权限和失败处理。

## 故障排查

| 现象 | 检查内容 |
|---|---|
| OwlMail 启动失败 | 配置路径、展开后的 URL/密钥、时长、模板和 JSON 语法。 |
| WebHook 返回 401/403 | 两边密钥是否一致，代理是否修改请求体或签名头。 |
| WebHook 返回 404 | 目标 URL 是否以 `/hooks/owlmail` 结尾，Hook ID 是否为 `owlmail`。 |
| OwlMail 遇到 5xx 后重试 | 检查命令退出状态、WebHook 超时及并发日志。 |
| 日志里没有命令输出 | 捕获的 stdout 不一定自动进入容器日志；根据工作流显式记录日志或执行副作用。 |
| 同一事件重复处理 | 使用 `OWLMAIL_EMAIL_ID` 作为去重键，让命令保持幂等。 |

[English](../en-US/OwlMail-Integration.md)
