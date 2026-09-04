# OwlMail integration example

This example connects [OwlMail](https://github.com/soulteary/owlmail) `v0.9.0` to
`soulteary/webhook`. Every accepted email is rendered as JSON, signed with
HMAC-SHA256, verified by WebHook, and passed to a demo command through a fixed
set of environment variables.

OwlMail `v0.9.0` adds a durable local outbox, optional Redis-backed recovery,
and replay-aware signature metadata to the webhook pipeline. The Compose demo
pins `ghcr.io/soulteary/owlmail:0.9.0` and
`soulteary/webhook:extend-7.1.0`, and keeps OwlMail's safe default delivery
concurrency of `8` explicit.

## Run

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

All published demo ports bind to `127.0.0.1`, so the SMTP listener, inbox UI,
and WebHook endpoint are not exposed to other hosts on the network.

Send a test message from another terminal:

```bash
printf 'From: monitor@example.test\r\nTo: ops@example.test\r\nSubject: Demo alert\r\n\r\nThe integration works.\r\n' \
  | curl --url smtp://127.0.0.1:1025 \
      --mail-from monitor@example.test \
      --mail-rcpt ops@example.test \
      --upload-file -
```

The `webhook` container logs a summary from `print-email.sh`, including both
the email ID and OwlMail's stable delivery ID. Open
`http://127.0.0.1:1080` to inspect the captured message. Open
`http://127.0.0.1:1080/webhooks` to inspect OwlMail's webhook configurator and
compare its generated configuration with `owlmail.json` in this directory.

```bash
docker compose down
```

Webhook delivery is asynchronous relative to SMTP acceptance and has
at-least-once semantics. Use `OWLMAIL_DELIVERY_ID` as the idempotency key for
side effects; retries keep the same delivery ID. The bundled rule validates the
legacy body-only `X-OwlMail-Signature` header. OwlMail 0.9.0 also sends the
replay-aware `X-OwlMail-Signature-V2`, timestamp, and nonce, but this example
does not validate that V2 tuple.

The example stores no persistent mail volume or Redis queue. It enables WebHook debug logging
so the demo command output is visible, but keeps raw request-body logging
disabled. Disable `DEBUG` in production because the mapped email fields printed
by the command will otherwise be written to logs.

For configuration details, production guidance, field mappings, and
troubleshooting, read the [OwlMail integration guide](../../docs/en-US/OwlMail-Integration.md).

[中文说明](./README.zh-CN.md)
