# OwlMail 联动示例

该示例将 [OwlMail](https://github.com/soulteary/owlmail) 与
`soulteary/webhook` 连接起来。每封接收的邮件都会被渲染为 JSON，由 OwlMail
使用 HMAC-SHA256 签名；WebHook 验证签名后，再通过一组固定环境变量将字段传给
演示命令。

当前已发布的 OwlMail `v0.4.0` 镜像早于 Webhook 转发功能，因此 Compose 示例
会从 OwlMail 当前 `main` 分支构建。后续正式版本包含该功能后，可将 `build`
替换为对应的 OwlMail 发布镜像。

## 运行

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up --build
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
`http://127.0.0.1:1080` 可以查看捕获的邮件。

```bash
docker compose down
```

示例未设置持久化邮件数据卷。为了让演示命令输出可见，它开启了 WebHook 调试
日志，但仍关闭原始请求体日志。生产环境必须关闭 `DEBUG`，否则命令输出的邮件
字段仍会进入日志。

配置说明、生产建议、字段映射和故障排查见
[OwlMail 联动指南](../../docs/zh-CN/OwlMail-Integration.md)。

[English](./README.md)
