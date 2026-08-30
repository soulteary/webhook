# OwlMail 联动示例

该示例将 [OwlMail](https://github.com/soulteary/owlmail) `v0.5.0` 与
`soulteary/webhook` 连接起来。每封接收的邮件都会被渲染为 JSON，由 OwlMail
使用 HMAC-SHA256 签名；WebHook 验证签名后，再通过一组固定环境变量将字段传给
演示命令。

OwlMail `v0.5.0` 已正式包含 Webhook 转发、`/webhooks` 浏览器配置生成器，以及
Webhook 投递并发上限。Compose Demo 直接使用发布镜像
`soulteary/owlmail:0.5.0`，并显式设置默认并发值 `8`，让示例同时能够说明生产
运行时行为。

## 运行

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

在另一个终端发送测试邮件：

```bash
printf 'From: monitor@example.test\r\nTo: ops@example.test\r\nSubject: Demo alert\r\n\r\nThe integration works.\r\n' \
  | curl --url smtp://127.0.0.1:1025 \
      --mail-from monitor@example.test \
      --mail-rcpt ops@example.test \
      --upload-file -
```

`webhook` 容器会输出 `print-email.sh` 生成的摘要。打开
`http://127.0.0.1:1080` 可以查看捕获的邮件；打开
`http://127.0.0.1:1080/webhooks` 可以查看 OwlMail 的 Webhook 配置生成器，并与
本目录的 `owlmail.json` 示例进行对照。

```bash
docker compose down
```

示例未设置持久化邮件数据卷。为了让演示命令输出可见，它开启了 WebHook 调试
日志，但仍关闭原始请求体日志。生产环境必须关闭 `DEBUG`，否则命令输出的邮件
字段仍会进入日志。

配置说明、生产建议、字段映射和故障排查见
[OwlMail 联动指南](../../docs/zh-CN/OwlMail-Integration.md)。

[English](./README.md)
