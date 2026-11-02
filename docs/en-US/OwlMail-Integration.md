# Integrating WebHook with OwlMail

[OwlMail](https://github.com/soulteary/owlmail) can forward accepted email events
to WebHook. WebHook authenticates the exact request body, maps selected JSON
fields to environment variables, and runs a controlled local command.

```text
SMTP client -> OwlMail v0.13.0 -> signed HTTP POST -> WebHook -> command/script
```

A runnable example is available in [`example/owlmail`](../../example/owlmail/).
OwlMail also maintains the sender-side example in
[`examples/webhooks/soulteary-webhook`](https://github.com/soulteary/owlmail/tree/v0.13.0/examples/webhooks/soulteary-webhook).

## Requirements

- WebHook 7.3.4 (the Compose example uses its shell-capable `runner` image).
- OwlMail v0.13.0 (the version pinned by this example) or later.
- A random shared secret of at least 32 bytes.
- Network reachability from OwlMail to the WebHook endpoint.

## Configure WebHook

Start from [`hooks.json.tmpl`](https://github.com/soulteary/webhook/blob/main/example/owlmail/hooks.json.tmpl). It defines
`POST /hooks/owlmail`, requires `application/json`, validates
`X-OwlMail-Signature`, and maps `event`, `emailId`, `title`, `message`, `from`,
`to`, and `receivedAt` to fixed `OWLMAIL_*` command environment variables. It
also maps the `X-OwlMail-Delivery-ID` request header to
`OWLMAIL_DELIVERY_ID` for delivery correlation.

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

## Configure OwlMail v0.13.0

Open `/webhooks` to create, import, validate, copy, and download a version 1
configuration. Editing happens locally in the browser. Downloading does not
activate the configuration: mount the JSON file and select it with
`-webhook-config` or `OWLMAIL_WEBHOOK_CONFIG`.

The runnable demo uses the released image directly:

```yaml
owlmail:
  image: ghcr.io/soulteary/owlmail:0.13.0
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

## Signatures and delivery identity

OwlMail 0.13.0 sends both signature formats when a target has a `secret`:

| Header | Meaning |
|---|---|
| `X-OwlMail-Signature` | Legacy HMAC-SHA256 over the exact request body: `sha256=<hex digest>`. |
| `X-OwlMail-Signature-V2` | Replay-aware HMAC over `timestamp + "." + nonce + "." + body`: `v2=<hex digest>`. |
| `X-OwlMail-Timestamp` | UTC RFC 3339 signing time. |
| `X-OwlMail-Nonce` | A new random value for each HTTP attempt. |
| `X-OwlMail-Delivery-ID` | Stable identifier retained across retries of one queued delivery; neither signature covers this header. |

The bundled WebHook `payload-hmac-sha256` rule validates the legacy
body-only header. It prevents body tampering, but it does not validate the
timestamp or nonce and therefore does not itself provide replay protection. If
replay resistance is required, validate the V2 tuple in an authenticated
reverse proxy or purpose-built handler, enforce a short timestamp window, and
reject reused nonces.

Neither the legacy nor V2 signature covers `X-OwlMail-Delivery-ID`: V2 signs
only the timestamp, nonce, and exact body. Treat `OWLMAIL_DELIVERY_ID` as
correlation metadata, not as an authenticated authorization value or sole
idempotency key across an untrusted path. For this fixed `email.received`
payload, derive the idempotency key from the signed `event` and `emailId`
fields. If a future integration needs a trusted per-delivery identifier, place
an authenticated copy in the signed body through a trusted adapter.

## Delivery timing and durability

OwlMail accepts and stores SMTP mail independently from the eventual HTTP
result. In 0.13.0, each event is first synced to
`.owlmail-webhook-outbox` under the mail directory; HTTP delivery then happens
asynchronously and with at-least-once semantics. A failed WebHook command causes
a non-2xx delivery result and retry, but it does not reject or delete the
already accepted email.

For restart-safe delivery, use a persistent OwlMail mail directory and Redis
6.2 or newer:

```bash
export OWLMAIL_MAIL_DIR=/app/mail
export OWLMAIL_WEBHOOK_REDIS_URL=redis://redis:6379/0
export OWLMAIL_WEBHOOK_REDIS_PREFIX=owlmail:webhooks
export OWLMAIL_WEBHOOK_SHUTDOWN_TIMEOUT=15s
```

Without Redis, the local outbox protects work until the in-memory queue accepts
it; accepted in-memory jobs and exhausted deliveries are not replayed after a
restart. With Redis, pending jobs can be reclaimed and exhausted deliveries are
moved to a dead-letter Stream. Use one active OwlMail instance per Redis prefix.

The default receiver treats an invalid HMAC as a trigger-rule evaluation error
(currently HTTP 500) and does not execute the command. This legacy status is
not an acceptance result: OwlMail treats it as a failure and applies its bounded
retry policy. Fix mismatched secrets rather than enabling signature soft failures.

## Upgrading an existing deployment

The version 1 webhook JSON, exact-body HMAC, environment-variable expansion,
field mappings and concurrency limit remain compatible. Keep the same shared
secret and the `runner` image variant; the core image cannot run `print-email.sh`.
The example explicitly probes `/app/owlmail healthcheck --container`; do not
assume the published image supplies a Docker HEALTHCHECK or puts `/app` in PATH.

Stop incoming SMTP traffic and the old process before backing up the complete
mail directory, webhook outbox and any configured Redis/S3 state. Rehearse
0.13.0 against an isolated copy with real outbound destinations disabled, then
check readiness, retained mail/attachments, a new SMTP receipt and its signed
webhook command before reopening traffic. Recovery may resume pending work,
and delivery remains at least once. If Relay is configured, review its
[0.13.0 recovery changes](https://github.com/soulteary/owlmail/blob/v0.13.0/docs/en/Release-0.13.0.md).
Rollback requires stopping the new process and restoring the pre-upgrade backup
with the old image and configuration; do not share writable state between
versions. Reconcile downstream receipts before resuming pending deliveries.
The demo itself remains ephemeral and does not configure Redis or persistent mail.

## One-command demo

```bash
cd example/owlmail
export OWLMAIL_WEBHOOK_SECRET="$(openssl rand -hex 32)"
docker compose up
```

Send a test message to `smtp://127.0.0.1:1025`. Open
`http://127.0.0.1:1080` for the inbox and `http://127.0.0.1:1080/webhooks` for
the v0.13.0 webhook configurator. The WebHook container logs the verified and
mapped demo event.

## Production checklist

- Keep WebHook private or behind an authenticated TLS reverse proxy and retain HMAC.
- Treat the bundled legacy HMAC rule as body authentication, not replay protection; validate OwlMail's V2 signature when replay resistance is required.
- Restrict `-allowed-command-paths`; enable strict mode, bounded concurrency, execution timeouts, and rate limits.
- Keep OwlMail at the default concurrency of `8` or size it to downstream capacity; do not accidentally set it to `0`.
- Keep raw request-body logging disabled for email payloads and run commands unprivileged.
- For this fixed payload, deduplicate with the signed `(event, emailId)` tuple; use the unsigned delivery ID only for correlation.
- Persist OwlMail's mail directory and configure Redis when delivery must survive process restarts.
- Keep OwlMail's per-attempt timeout above normal command duration but bounded; set a finite shutdown drain timeout.

## Filtering and multiple workflows

OwlMail v0.13.0 supports case-insensitive wildcard filters for sender, recipient,
subject, and plain-text body. Multiple targets can call separate hook IDs such as
`/hooks/owlmail-archive`, `/hooks/owlmail-alert`, and `/hooks/owlmail-ticket`.
Use separate commands and, where practical, separate secrets to isolate permissions.

## Troubleshooting

| Symptom | Check |
|---|---|
| OwlMail fails at startup | Config path, expanded URL/secret, duration, template, and JSON syntax. |
| WebHook returns 401/403 | Both services use the same secret and no proxy changes the body or signature header. |
| WebHook returns 404 | The target URL ends in `/hooks/owlmail` and the hook ID is `owlmail`. |
| OwlMail retries after 5xx | Inspect command exit status and WebHook timeout/concurrency logs. |
| Duplicate processing | Delivery is at least once; deduplicate this fixed payload with the signed `(event, emailId)` tuple and use the unsigned delivery ID only for correlation. |
| Events disappear after restart | Persist the mail directory and configure Redis; the demo intentionally uses neither. |
| Replay protection is required | The bundled rule checks the legacy body HMAC only; validate the V2 timestamp, nonce, signature, and replay window separately. |

[中文文档](../zh-CN/OwlMail-Integration.md)
