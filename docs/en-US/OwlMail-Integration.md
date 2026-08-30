# Integrating WebHook with OwlMail

[OwlMail](https://github.com/soulteary/owlmail) can forward accepted email events
to WebHook. WebHook authenticates the exact request body, maps selected JSON
fields to environment variables, and runs a controlled local command.

```text
SMTP client -> OwlMail v0.5.0 -> signed HTTP POST -> WebHook -> command/script
```

A runnable example is available in [`example/owlmail`](../../example/owlmail/).
OwlMail also maintains the sender-side example in
[`examples/webhooks/soulteary-webhook`](https://github.com/soulteary/owlmail/tree/main/examples/webhooks/soulteary-webhook).

## Requirements

- WebHook 7.0.0 or later.
- OwlMail v0.5.0 or later. v0.5.0 includes webhook forwarding and the `/webhooks` configurator.
- A random shared secret of at least 32 bytes.
- Network reachability from OwlMail to the WebHook endpoint.

## Configure WebHook

Start from [`hooks.json.tmpl`](../../example/owlmail/hooks.json.tmpl). It defines
`POST /hooks/owlmail`, requires `application/json`, validates
`X-OwlMail-Signature`, and maps `event`, `emailId`, `title`, `message`, `from`,
`to`, and `receivedAt` to fixed `OWLMAIL_*` command environment variables.

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

The example waits for command completion, so failures return non-2xx responses
and OwlMail can retry. Keep handlers idempotent and avoid printing secrets or
unnecessary full message bodies.

## Configure OwlMail v0.5.0

Open `/webhooks` to create, import, validate, copy, and download a version 1
configuration. Editing happens locally in the browser. Downloading does not
activate the configuration: mount the JSON file and select it with
`-webhook-config` or `OWLMAIL_WEBHOOK_CONFIG`.

The runnable demo uses the released image directly:

```yaml
owlmail:
  image: soulteary/owlmail:0.5.0
  environment:
    OWLMAIL_WEBHOOK_CONFIG: /app/config/owlmail.json
    OWLMAIL_WEBHOOK_MAX_CONCURRENCY: "8"
```

OwlMail's process-wide webhook concurrency limit defaults to `8`. Set
`OWLMAIL_WEBHOOK_MAX_CONCURRENCY=0` only when unlimited delivery is intentional.

For the demo, use `http://webhook:9000/hooks/owlmail`, `POST`, JSON content, the
same shared secret on both services, a timeout above normal command duration,
and a small bounded retry count. The checked-in example uses `10s` and `2` retries.

```bash
export OWLMAIL_WEBHOOK_URL=http://127.0.0.1:9000/hooks/owlmail
export OWLMAIL_WEBHOOK_SECRET='replace-with-the-same-random-secret'
owlmail -webhook-config ./owlmail.json
```

OwlMail expands environment variables before validation. Missing or invalid
runtime values fail startup instead of silently disabling delivery.

## HMAC contract

OwlMail computes HMAC-SHA256 over the exact HTTP request body and sends:

```text
X-OwlMail-Signature: sha256=<hex digest>
```

WebHook's `payload-hmac-sha256` rule verifies the same raw bytes. A proxy must not
rewrite the request body; a signature mismatch prevents command execution.

## One-command demo

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

Send a test message to `smtp://127.0.0.1:1025`. Open
`http://127.0.0.1:1080` for the inbox and `http://127.0.0.1:1080/webhooks` for
the v0.5.0 webhook configurator. The WebHook container logs the verified and
mapped demo event.

## Production checklist

- Keep WebHook private or behind an authenticated reverse proxy and retain HMAC.
- Restrict `-allowed-command-paths`; enable strict mode, bounded concurrency, execution timeouts, and rate limits.
- Keep OwlMail at the default concurrency of `8` or size it to downstream capacity; do not accidentally set it to `0`.
- Keep raw request-body logging disabled for email payloads and run commands unprivileged.
- Use `OWLMAIL_EMAIL_ID` as a deduplication key so retries are safe.
- Keep OwlMail's timeout above normal command duration but bounded for the event pipeline.

## Filtering and multiple workflows

OwlMail v0.5.0 supports case-insensitive wildcard filters for sender, recipient,
and subject. Multiple targets can call separate hook IDs such as
`/hooks/owlmail-archive`, `/hooks/owlmail-alert`, and `/hooks/owlmail-ticket`.
Use separate commands and, where practical, separate secrets to isolate permissions.

## Troubleshooting

| Symptom | Check |
|---|---|
| OwlMail fails at startup | Config path, expanded URL/secret, duration, template, and JSON syntax. |
| WebHook returns 401/403 | Both services use the same secret and no proxy changes the body or signature header. |
| WebHook returns 404 | The target URL ends in `/hooks/owlmail` and the hook ID is `owlmail`. |
| OwlMail retries after 5xx | Inspect command exit status and WebHook timeout/concurrency logs. |
| Duplicate processing | Make the command idempotent using `OWLMAIL_EMAIL_ID`. |

[中文文档](../zh-CN/OwlMail-Integration.md)
