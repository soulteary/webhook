# Integrating WebHook with OwlMail

[OwlMail](https://github.com/soulteary/owlmail) can forward accepted email
events to WebHook. WebHook authenticates the exact request body, maps selected
JSON fields to environment variables, and runs a controlled local command.

```text
SMTP client -> OwlMail -> signed HTTP POST -> WebHook -> command/script
```

A runnable example is available in [`example/owlmail`](../../example/owlmail/).
OwlMail also maintains the sender-side example in
[`examples/webhooks/soulteary-webhook`](https://github.com/soulteary/owlmail/tree/main/examples/webhooks/soulteary-webhook).

## Requirements

- WebHook 7.0.0 or later.
- An OwlMail build that includes webhook forwarding. The published OwlMail
  `v0.4.0` image does not include it; build current `main` or use a later release.
- A random shared secret of at least 32 bytes.
- Network reachability from OwlMail to the WebHook endpoint.

## 1. Configure WebHook

Start from [`hooks.json.tmpl`](../../example/owlmail/hooks.json.tmpl). It defines
`POST /hooks/owlmail`, requires `application/json`, validates
`X-OwlMail-Signature`, and maps the payload to these variables:

| JSON field | Command environment variable |
|---|---|
| `event` | `OWLMAIL_EVENT` |
| `emailId` | `OWLMAIL_EMAIL_ID` |
| `title` | `OWLMAIL_TITLE` |
| `message` | `OWLMAIL_MESSAGE` |
| `from` | `OWLMAIL_FROM` |
| `to` | `OWLMAIL_TO` |
| `receivedAt` | `OWLMAIL_RECEIVED_AT` |

The configuration is a Go template so the secret and command paths stay out of
source control:

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

WebHook waits for the command because
`include-command-output-in-response` is enabled. A command execution failure
therefore returns a non-2xx response and allows OwlMail to retry. Command output
is returned to OwlMail on success; do not print secrets or full message bodies
unless that behavior is intentional.

## 2. Configure OwlMail

In an OwlMail build with the configurator, open `/webhooks`, create a target,
and download `webhooks.json`. You can also start from
[`owlmail.json`](../../example/owlmail/owlmail.json).

Use these values:

| Setting | Value |
|---|---|
| URL | `http://webhook:9000/hooks/owlmail` in Compose, or the reachable WebHook URL |
| Method | `POST` |
| Content type | `application/json` |
| Secret | The same value as `OWLMAIL_WEBHOOK_SECRET` |
| Timeout | Longer than the expected command runtime; the example uses `10s` |
| Retries | A small bounded value; the example uses `2` |

The example body template creates a stable integration payload rather than
forwarding the entire internal email object:

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

Mount the downloaded configuration and enable it with either the flag or the
environment variable:

```bash
export OWLMAIL_WEBHOOK_URL=http://127.0.0.1:9000/hooks/owlmail
export OWLMAIL_WEBHOOK_SECRET='replace-with-the-same-random-secret'
owlmail -webhook-config ./owlmail.json
```

or:

```bash
export OWLMAIL_WEBHOOK_CONFIG=/app/config/owlmail.json
```

OwlMail expands environment variables before validating the configuration, so
missing or invalid values fail startup instead of silently disabling delivery.

## HMAC contract

For each request, OwlMail computes HMAC-SHA256 over the exact HTTP request body
and sends the hexadecimal digest as:

```text
X-OwlMail-Signature: sha256=<hex digest>
```

WebHook's `payload-hmac-sha256` rule verifies the same raw bytes using the shared
secret. Do not place a proxy in the path that rewrites or decompresses the body.
A signature mismatch returns a non-2xx response and the command is not run.

## Production checklist

- Keep WebHook on a private network or behind an authenticated reverse proxy.
- Use HMAC even on a private container network; never commit the shared secret.
- Set `-allowed-command-paths` to the exact script whenever possible.
- Enable `-strict-mode`, bounded concurrency, execution timeouts, and rate limits.
- Keep `-debug` and `-log-request-body` disabled for email payloads.
- Run the command as an unprivileged user and mount scripts/configuration read-only.
- Make handlers idempotent: OwlMail retries failed deliveries and the same email
  event can therefore be processed more than once.
- Keep OwlMail's timeout above the normal command duration but below the maximum
  duration acceptable to your SMTP/event pipeline.

## Filtering and multiple workflows

OwlMail can filter targets by sender, recipient, subject, and text patterns. Use
filters on the sender side when a workflow should only receive selected mail.
Multiple OwlMail targets can call different WebHook hook IDs, for example:

- `/hooks/owlmail-archive` for all messages;
- `/hooks/owlmail-alert` for subjects matching `Critical*`;
- `/hooks/owlmail-ticket` for messages sent to `support@example.com`.

Give every endpoint its own command and, where practical, its own secret. This
keeps permissions and failure handling isolated.

## Troubleshooting

| Symptom | Check |
|---|---|
| OwlMail fails at startup | The config path, expanded URL/secret, duration, template, and JSON syntax. |
| WebHook returns 401/403 | Both services use the same secret and no proxy changes the body or signature header. |
| WebHook returns 404 | The target URL ends in `/hooks/owlmail` and the hook ID is `owlmail`. |
| OwlMail retries after 5xx | Inspect command exit status and WebHook timeout/concurrency logs. |
| No command output in logs | Container logs do not automatically show captured stdout; use an explicit logger or a side effect appropriate to the workflow. |
| Duplicate processing | Make the command idempotent using `OWLMAIL_EMAIL_ID` as the deduplication key. |

[中文文档](../zh-CN/OwlMail-Integration.md)
