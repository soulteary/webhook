# Templates in Webhook

[`webhook`][w] can parse a hooks configuration file as a Go template when given the `-template` [CLI parameter](Webhook-Parameters.md).

In addition to the [built-in Go template functions and features][tt], `webhook` provides:

- `getenv` for inserting environment variables.
- `getenvRequired` for rejecting an unset or empty environment variable.
- `getSecret` for reading one named file from a dedicated secret directory.
- `trimSpace` for explicitly removing surrounding whitespace, including a trailing newline, from a template value.

Use `getenvRequired` or `getSecret` for authentication secrets so the configuration fails closed.

## Secret files

`getSecret` accepts a single file name, not an arbitrary path. It reads from
`/run/secrets` by default, which matches Docker and Docker Swarm secret mounts.
Set `WEBHOOK_SECRET_DIR` to an absolute path when the secrets are mounted
elsewhere.

The function rejects absolute paths, directory separators, path traversal,
files that resolve outside the secret directory, non-regular files, and files
larger than 1 MiB. It preserves the file contents exactly; use `trimSpace`
explicitly when the stored secret contains a trailing newline.

```json
{
  "source": "string",
  "envname": "AUTHORIZATION",
  "name": "Bearer {{ getSecret "webhook-token" | trimSpace | js }}"
}
```

Do not use or expect a general-purpose `cat` function. Restricting reads to a
dedicated operator-controlled directory prevents a hook template from becoming
an arbitrary file reader.

## Example Usage

In the example JSON template file below (YAML is also supported), the `payload-hmac-sha1` matching rule looks up the HMAC secret from the environment using the `getenvRequired` template function.
Additionally, the result is piped through the built-in Go template function `js` to ensure that the result is a well-formed Javascript/JSON string.

```
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
    "response-message": "I got the payload!",
    "response-headers":
    [
      {
        "name": "Access-Control-Allow-Origin",
        "value": "*"
      }
    ],
    "pass-arguments-to-command":
    [
      {
        "source": "payload",
        "name": "head_commit.id"
      },
      {
        "source": "payload",
        "name": "pusher.name"
      },
      {
        "source": "payload",
        "name": "pusher.email"
      }
    ],
    "trigger-rule":
    {
      "and":
      [
        {
          "match":
          {
            "type": "payload-hmac-sha1",
            "secret": "{{ getenvRequired "XXXTEST_SECRET" | js }}",
            "parameter":
            {
              "source": "header",
              "name": "X-Hub-Signature"
            }
          }
        },
        {
          "match":
          {
            "type": "value",
            "value": "refs/heads/master",
            "parameter":
            {
              "source": "payload",
              "name": "ref"
            }
          }
        }
      ]
    }
  }
]

```

[w]: https://github.com/adnanh/webhook
[tt]: https://golang.org/pkg/text/template/
