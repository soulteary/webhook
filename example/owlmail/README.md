# OwlMail integration example

This example connects [OwlMail](https://github.com/soulteary/owlmail) to
`soulteary/webhook`. Every accepted email is rendered as JSON, signed with
HMAC-SHA256, verified by WebHook, and passed to a demo command through a fixed
set of environment variables.

The published OwlMail `v0.4.0` image predates webhook forwarding. The Compose
example therefore builds OwlMail from its current `main` branch. Replace that
build with a released OwlMail image after webhook forwarding is included in a
release.

## Run

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up --build
```

Send a test message from another terminal:

```bash
printf 'From: monitor@example.test\r\nTo: ops@example.test\r\nSubject: Demo alert\r\n\r\nThe integration works.\r\n' \
  | curl --url smtp://127.0.0.1:1025 \
      --mail-from monitor@example.test \
      --mail-rcpt ops@example.test \
      --upload-file -
```

The `webhook` container logs a summary from `print-email.sh`. Open
`http://127.0.0.1:1080` to inspect the captured message.

```bash
docker compose down
```

The example stores no persistent mail volume. It enables WebHook debug logging
so the demo command output is visible, but keeps raw request-body logging
disabled. Disable `DEBUG` in production because the mapped email fields printed
by the command will otherwise be written to logs.

For configuration details, production guidance, field mappings, and
troubleshooting, read the [OwlMail integration guide](../../docs/en-US/OwlMail-Integration.md).

[中文说明](./README.zh-CN.md)
