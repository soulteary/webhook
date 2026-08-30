# WebHook 与 OwlMail 联动

[OwlMail](https://github.com/soulteary/owlmail) 可以把接收邮件事件转发给
WebHook。WebHook 对完整请求体进行鉴权，将指定 JSON 字段映射为环境变量，再执行
受控的本地命令。

```text
SMTP 客户端 -> OwlMail v0.5.0 -> 带签名的 HTTP POST -> WebHook -> 命令/脚本
```

可运行示例位于 [`example/owlmail`](../../example/owlmail/)。OwlMail 仓库也维护了
发送端示例：[`examples/webhooks/soulteary-webhook`](https://github.com/soulteary/owlmail/tree/main/examples/webhooks/soulteary-webhook)。

## 使用要求

- WebHook 7.0.0 或更高版本。
- OwlMail v0.5.0 或更高版本；v0.5.0 已正式包含 Webhook 转发与 `/webhooks` 配置生成器。
- 至少 32 字节的随机共享密钥。
- OwlMail 能够访问 WebHook 端点。

## 1. 配置 WebHook

可以从 [`hooks.json.tmpl`](../../example/owlmail/hooks.json.tmpl) 开始。它定义了
`POST /hooks/owlmail`，要求 `application/json`，验证 `X-OwlMail-Signature`，并将
`event`、`emailId`、`title`、`message`、`from`、`to`、`receivedAt` 映射到固定的
`OWLMAIL_*` 环境变量。

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

## 2. 配置 OwlMail v0.5.0

打开 OwlMail 的 `/webhooks` 页面，可以创建、导入、校验、复制和下载 version 1
配置。编辑过程发生在浏览器本地；下载配置并不会自动激活它，需要把 JSON 文件挂载
到 OwlMail 并通过 `-webhook-config` 或 `OWLMAIL_WEBHOOK_CONFIG` 指定。

Compose 示例直接使用发布镜像：

```yaml
owlmail:
  image: soulteary/owlmail:0.5.0
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

## HMAC 约定

OwlMail 对每次 HTTP 请求的原始请求体计算 HMAC-SHA256：

```text
X-OwlMail-Signature: sha256=<十六进制摘要>
```

WebHook 的 `payload-hmac-sha256` 规则使用同一密钥验证原始字节。代理不能改写请求体；
签名不匹配时不执行命令。

## 一键 Demo

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

随后通过 `curl smtp://127.0.0.1:1025` 发送测试邮件。访问 `http://127.0.0.1:1080`
查看邮件，访问 `http://127.0.0.1:1080/webhooks` 查看 v0.5.0 的 Webhook 配置生成器，
WebHook 容器日志中可以看到经过签名校验和字段映射后的 Demo 输出。

## 生产环境检查清单

- WebHook 放在私有网络或带认证的反向代理之后，并保留 HMAC。
- `-allowed-command-paths` 只允许明确脚本，启用 `-strict-mode`、并发限制、执行超时和限流。
- OwlMail 保持默认 `8` 或按下游容量设置 `OWLMAIL_WEBHOOK_MAX_CONCURRENCY`，不要无意设置为 `0`。
- 邮件链路不要开启原始请求体日志；命令使用非特权用户执行。
- 使用 `OWLMAIL_EMAIL_ID` 作为去重键，使重试安全。
- OwlMail 超时应高于命令正常耗时，但不要无限延长事件链路等待时间。

## 过滤与多工作流

OwlMail v0.5.0 支持按发件人、收件人和主题进行大小写不敏感的通配符过滤。多个目标可
分别调用 `/hooks/owlmail-archive`、`/hooks/owlmail-alert`、`/hooks/owlmail-ticket`
等 Hook，并使用独立命令和密钥隔离权限与失败处理。

## 故障排查

| 现象 | 检查内容 |
|---|---|
| OwlMail 启动失败 | 配置路径、展开后的 URL/密钥、时长、模板和 JSON 语法。 |
| WebHook 返回 401/403 | 两边密钥是否一致，代理是否修改请求体或签名头。 |
| WebHook 返回 404 | 目标 URL 是否以 `/hooks/owlmail` 结尾，Hook ID 是否为 `owlmail`。 |
| OwlMail 遇到 5xx 后重试 | 检查命令退出状态、WebHook 超时及并发日志。 |
| 同一事件重复处理 | 使用 `OWLMAIL_EMAIL_ID` 作为去重键，让命令保持幂等。 |

[English](../en-US/OwlMail-Integration.md)
