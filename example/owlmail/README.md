# OwlMail integration example

This example connects [OwlMail](https://github.com/soulteary/owlmail) `v0.15.0` to
`soulteary/webhook`. Every accepted email is rendered as JSON, signed with
HMAC-SHA256, verified by WebHook, and passed to a demo command through a fixed
set of environment variables.

OwlMail `v0.15.0` retains the durable local outbox, optional Redis-backed recovery,
and replay-aware signature metadata to the webhook pipeline. The Compose demo
pins `ghcr.io/soulteary/owlmail:0.15.0` and
`soulteary/webhook:runner-7.3.4`, and keeps OwlMail's safe default delivery
concurrency of `8` explicit.

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
0.15.0 against an isolated copy with real outbound destinations disabled, then
check readiness, retained mail/attachments, a new SMTP receipt and its signed
webhook command before reopening traffic. Recovery may resume pending work,
and delivery remains at least once. If Relay is configured, review its
[0.15.0 upgrade notes](https://github.com/soulteary/owlmail/blob/v0.15.0/docs/en/Release-0.15.0.md).
Rollback requires stopping the new process and restoring the pre-upgrade backup
with the old image and configuration; do not share writable state between
versions. Reconcile downstream receipts before resuming pending deliveries.
The demo itself remains ephemeral and does not configure Redis or persistent mail.

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
the email ID and OwlMail's stable delivery ID for correlation. Open
`http://127.0.0.1:1080` to inspect the captured message. Open
`http://127.0.0.1:1080/webhooks` to inspect OwlMail's webhook configurator and
compare its generated configuration with `owlmail.json` in this directory.

```bash
docker compose down
```

Webhook delivery is asynchronous relative to SMTP acceptance and has
at-least-once semantics. Retries keep the same `OWLMAIL_DELIVERY_ID`, which is
useful for log correlation, but neither OwlMail signature covers that header.
For idempotency in this fixed `email.received` contract, derive the key from
the signed body fields `OWLMAIL_EVENT` and `OWLMAIL_EMAIL_ID`; do not use the
unsigned delivery ID as the sole key across an untrusted path. The bundled rule
validates the legacy body-only `X-OwlMail-Signature`. OwlMail 0.15.0 also sends
the replay-aware `X-OwlMail-Signature-V2`, timestamp, and nonce, but this
example does not validate that V2 tuple.

The example stores no persistent mail volume or Redis queue. It enables WebHook debug logging
so the demo command output is visible, but keeps raw request-body logging
disabled. Disable `DEBUG` in production because the mapped email fields printed
by the command will otherwise be written to logs.

For configuration details, production guidance, field mappings, and
troubleshooting, read the [OwlMail integration guide](../../docs/en-US/OwlMail-Integration.md).

[中文说明](./README.zh-CN.md)
