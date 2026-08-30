# OwlMail integration example

This example connects [OwlMail](https://github.com/soulteary/owlmail) `v0.5.0` to
`soulteary/webhook`. Every accepted email is rendered as JSON, signed with
HMAC-SHA256, verified by WebHook, and passed to a demo command through a fixed
set of environment variables.

OwlMail `v0.5.0` includes webhook forwarding, the `/webhooks` browser
configurator, and bounded webhook delivery concurrency. The Compose demo uses
the released `soulteary/owlmail:0.5.0` image directly and keeps the default
OwlMail delivery concurrency of `8` explicit so the example documents the
runtime behavior.

## Run

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
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
`http://127.0.0.1:1080` to inspect the captured message. Open
`http://127.0.0.1:1080/webhooks` to inspect OwlMail's webhook configurator and
compare its generated configuration with `owlmail.json` in this directory.

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
