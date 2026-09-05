# Configuration tools

## Create a starter configuration

```bash
webhook init
```

This writes `hooks/hooks.yaml` with mode `0600`, refuses to overwrite an
existing file, and creates an HMAC-authenticated POST hook. Use `--output`,
`--format json`, or `--force` to change that behavior. Set `WEBHOOK_SECRET` and
run the generated configuration with `-template` before starting the server.

## Validate before deployment

```bash
webhook validate --strict -template -hooks hooks/hooks.yaml
```

`validate` checks runtime flags, parses every hook file, and validates hook IDs.
`--strict` additionally rejects unknown object fields, catching misspelled keys.
The legacy `-validate-config` flag remains supported.

## Diagnose the runtime environment

```bash
webhook doctor --strict -template -hooks hooks/hooks.yaml
```

`doctor` performs strict validation and also checks that each command exists and
is executable and that each configured working directory exists. It does not
start the HTTP server or execute hooks.

## Editor completion

The published [JSON Schema](https://raw.githubusercontent.com/soulteary/webhook/main/schema/hooks.schema.json)
describes the complete hook contract. Associate YAML files with it in VS Code:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/soulteary/webhook/main/schema/hooks.schema.json": [
      "hooks/*.yaml",
      "hooks/*.yml"
    ]
  }
}
```

For JSON, configure a file association because the hook file's top level is an
array and cannot contain a `$schema` property:

```json
{
  "json.schemas": [
    {
      "fileMatch": ["hooks/*.json"],
      "url": "https://raw.githubusercontent.com/soulteary/webhook/main/schema/hooks.schema.json"
    }
  ]
}
```

The CLI remains the source of truth for template expansion and runtime-specific
checks.
