# 60-second quickstart

This local-only demo starts the non-root extended image with the secure profile,
an HMAC-authenticated POST hook, a read-only root filesystem, and an explicit
command allowlist.

```bash
docker compose up -d --wait

curl --fail-with-body http://127.0.0.1:9000/hooks/hello \
  -H 'Content-Type: application/json' \
  -H 'X-Webhook-Signature: sha256=fc2f94757ec900bc605b39b984b3431fa13f3ea2eb7881f2861c1c747d9ffdbd' \
  --data-binary '{"message":"hello from webhook"}'

docker compose logs webhook
docker compose down
```

The fixed signature is valid only for the checked-in local demo secret and exact
request body. Set `DEMO_SECRET` and calculate a new HMAC before adapting this
example. The port is deliberately bound to loopback.
