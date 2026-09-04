# OwlMail 联动示例

该示例将 [OwlMail](https://github.com/soulteary/owlmail) `v0.9.0` 与
`soulteary/webhook` 连接起来。每封接收的邮件都会被渲染为 JSON，由 OwlMail
使用 HMAC-SHA256 签名；WebHook 验证签名后，再通过一组固定环境变量将字段传给
演示命令。

OwlMail `v0.9.0` 为 Webhook 链路增加了持久本地 outbox、可选 Redis 恢复队列与
防重放签名元数据。Compose Demo 固定使用
`ghcr.io/soulteary/owlmail:0.9.0` 和 `soulteary/webhook:extend-7.1.0`，并显式
保留 OwlMail 的安全默认并发值 `8`。

## 运行

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

Demo 发布的所有端口均绑定到 `127.0.0.1`，因此 SMTP 监听器、收件箱界面和
WebHook 端点不会暴露给网络中的其他主机。

在另一个终端发送测试邮件：

```bash
printf 'From: monitor@example.test\r\nTo: ops@example.test\r\nSubject: Demo alert\r\n\r\nThe integration works.\r\n' \
  | curl --url smtp://127.0.0.1:1025 \
      --mail-from monitor@example.test \
      --mail-rcpt ops@example.test \
      --upload-file -
```

`webhook` 容器会输出 `print-email.sh` 生成的摘要，其中包含邮件 ID 与 OwlMail
稳定的投递 ID。打开
`http://127.0.0.1:1080` 可以查看捕获的邮件；打开
`http://127.0.0.1:1080/webhooks` 可以查看 OwlMail 的 Webhook 配置生成器，并与
本目录的 `owlmail.json` 示例进行对照。

```bash
docker compose down
```

Webhook 投递相对于 SMTP 接收是异步的，并采用“至少一次”语义。涉及副作用时应使用
`OWLMAIL_DELIVERY_ID` 作为幂等键；重试会保持同一个投递 ID。示例中的内置规则验证
旧版、仅覆盖正文的 `X-OwlMail-Signature`。OwlMail 0.9.0 还会发送带时间戳和 nonce
的 `X-OwlMail-Signature-V2`，但本示例不校验这组 V2 数据。

示例未设置持久化邮件数据卷或 Redis 队列。为了让演示命令输出可见，它开启了 WebHook 调试
日志，但仍关闭原始请求体日志。生产环境必须关闭 `DEBUG`，否则命令输出的邮件
字段仍会进入日志。

配置说明、生产建议、字段映射和故障排查见
[OwlMail 联动指南](../../docs/zh-CN/OwlMail-Integration.md)。

[English](./README.md)
