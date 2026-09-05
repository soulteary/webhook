# Provider recipes

These recipes are loaded and exercised by `go test ./internal/server`, including
one accepted and one rejected delivery per provider. Copy a directory, replace
the command, and set the documented environment variable. Run webhook with
`-template` so secrets are read from the environment instead of source control.

| Provider | Endpoint | Authentication | Event filter | Secret variable |
| --- | --- | --- | --- | --- |
| GitHub | `/hooks/github-push` | HMAC-SHA256 (`X-Hub-Signature-256`) | `X-GitHub-Event: push` | `GITHUB_WEBHOOK_SECRET` |
| GitLab | `/hooks/gitlab-push` | Secret token (`X-Gitlab-Token`) | `X-Gitlab-Event: Push Hook` | `GITLAB_WEBHOOK_TOKEN` |
| Gitea | `/hooks/gitea-push` | HMAC-SHA256 (`X-Gitea-Signature`) | `X-Gitea-Event: push` | `GITEA_WEBHOOK_SECRET` |
| Harbor | `/hooks/harbor-push` | Configured bearer header | payload `type: PUSH_ARTIFACT` | `HARBOR_WEBHOOK_TOKEN` |
| Alertmanager | `/hooks/alertmanager-firing` | HTTP authorization bearer token | payload `status: firing` | `ALERTMANAGER_WEBHOOK_TOKEN` |

Example:

```bash
export GITHUB_WEBHOOK_SECRET='replace-with-a-random-secret'
webhook -template -hooks example/providers/github/hooks.yaml
```

The checked-in request bodies are deliberately minimal CI fixtures. Consult the
provider documentation for the full event schema and rotate every example
credential before production use.
