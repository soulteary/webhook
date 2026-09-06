# Container images

WebHook 7.2.0 publishes three multi-architecture runtime variants to Docker Hub
and GHCR. All variants run as UID/GID `65532`, use `/var/lib/webhook` as their
working directory, and contain the same statically linked WebHook binary.

| Variant | Versioned tag | Runtime contents | Use it when |
|---|---|---|---|
| Core | `core-7.2.0` | `scratch`, CA certificates, WebHook | Every executed command is a mounted static binary |
| Runner | `runner-7.2.0` | Alpine, BusyBox utilities, Bash, CA certificates | Hooks execute ordinary POSIX shell or Bash scripts |
| Extended | `extended-7.2.0` | Runner plus curl, jq, and yq | Hooks explicitly require network or JSON/YAML command-line tools |

The unprefixed `7.2.0` and `latest` tags are aliases for `core-7.2.0`. Pin a
versioned tag, or preferably an image digest, in production. Both registries use
the same tag contract:

```text
soulteary/webhook:<tag>
ghcr.io/soulteary/webhook:<tag>
```

## Security boundary

`core` has no shell, package manager, or general-purpose userland. It is the
smallest attack surface, but it cannot execute a shell script by itself.

`runner` is the normal choice for script hooks. It supplies `/bin/sh`, Bash,
BusyBox applets, and the public CA bundle, without adding curl, jq, or yq.

`extended` intentionally includes tools that can make outbound requests and
transform structured data. This is useful for debugging and some integrations,
but it also increases the number of installed packages and the capabilities
available to a compromised hook. Prefer `runner` when the extra tools are not
part of the hook contract.

For every variant, combine the non-root default with a read-only root filesystem,
read-only configuration mounts, a writable tmpfs only where needed, and the
secure profile's command allowlist. Mounted files must be accessible to UID/GID
`65532`.

## Migrating from 7.1

The old `extend-7.2.0` tag remains as a compatibility alias for
`extended-7.2.0`. Existing deployments can move without an immediate rename,
but new configuration should use `extended-*`.

The old extended image was commonly used for every script hook. In 7.2, choose
the smallest matching runtime:

- Move shell-only hooks to `runner-7.2.0`.
- Keep hooks that require curl, jq, or yq on `extended-7.2.0`.
- Use `core-7.2.0` only when the configured command is a static executable.

The default image remains shell-free, so changing an existing script-based
deployment from `extend-*` to the unprefixed version tag will not work.
