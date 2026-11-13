# Changelog

## Unreleased

## 7.4.0 - 2026-11-13

### Security

- Config UI: the save-to-directory API (`POST {config-ui-path}/api/save`) is now disabled unless `-config-ui-token` / `CONFIG_UI_TOKEN` is set, and requires the token in the `X-Config-UI-Token` header, a JSON content type and a same-origin request. Previously anyone who could reach the port could write a hook file and trigger it.
- Rate limiting no longer trusts client-supplied `X-Forwarded-For` / `X-Real-IP`. Clients are identified by the TCP peer address unless the request comes from an address listed in the new `-trusted-proxies` / `TRUSTED_PROXIES` option.
- Signature rules (`payload-hmac-*`) now cover the raw body of `multipart/form-data` requests; previously the signature was computed over an empty body.
- `-allowed-command-paths` directory entries are now matched by path component and after resolving symlinks. Previously `/opt/hooks` also allowed `/opt/hooks-evil/run.sh`, and a symlink inside an allowed directory could point anywhere.
- A client-supplied `X-Request-Id` is discarded unless `-x-request-id` is enabled, and `-x-request-id-limit` is now enforced.
- Command arguments in execution error logs are sanitized the same way as in the execution log.
- A `value` match rule with an empty configured value (for example a template `getenv` of an unset variable) no longer matches an empty parameter.

### Added

- `-rate-limit-per-ip-rps` and `-rate-limit-per-ip-burst` set a per-client quota separate from the global one (default `0`: same as the global values). The per-client quota is checked first, so requests rejected by it do not consume the global quota.

### Changed

- Asynchronous hooks reserve an execution slot before responding. When all slots stay busy for the acquire timeout the request gets `503` with `Retry-After` instead of a success response followed by a silently dropped execution.
- Hook files that load in the default (compatible) mode but contain unknown keys now log a startup/reload warning naming the ignored key.
- The Redis health check sends a real `PING` instead of always reporting healthy.
- `-max-header-bytes` takes effect when set to a non-default value; `-read-header-timeout-seconds` logs a warning because it cannot be applied.
- On shutdown the HTTP server and running async hooks finish before audit logging and tracing are closed.
- Require Go 1.27.4 and update Fiber to v3.7.0. Refresh first-party kit dependencies (audit-kit v2.1.6, cli-kit v1.9.4, health-kit v4.0.7, i18n-kit v4.0.6, logger-kit v3.0.6, metrics-kit v3.0.6, middleware-kit v3.0.6, redis-kit v1.8.2, secure-kit v2.1.5, tracing-kit v2.0.4, version-kit v4.0.5) and their compatible transitive requirements.
- Synchronize container, verification and migration examples with this release.

### Fixed

- `ip-whitelist` lists mixing IPv4 and IPv6 single addresses no longer return an error for clients of the other family.
- `parse-parameters-as-json` works for nested keys and no longer panics when the path crosses a non-object value.
- Signature helpers no longer panic on non-string header values, the Scalr check no longer appends to the request body buffer, and `pass-file-to-command` no longer writes to the shared hook definition under concurrent requests.
- Temp files created for `pass-file-to-command` are now removed when `command-working-directory` is not set.
- When a hook times out, the whole process group is killed (Unix) and the executor slot is released even if leftover processes keep the output pipes open.

### Upgrade from 7.3.4

- Config UI "Save to directory" stops working until `-config-ui-token` is configured.
- Senders of asynchronous hooks may now receive `503` under saturation and should retry.
- Deployments behind a reverse proxy that use rate limiting must set `-trusted-proxies`, otherwise all requests share the proxy's per-IP quota.

Hook files keep their format. Retain your hook files, configuration, credentials and mounted data. Keep the same runtime variant (`core`, `runner` or `extended`) and change only its version suffix to `7.4.0`. The unprefixed tag continues to select the core variant, which does not contain a shell.

Validate hooks and test a representative signed request before switching production traffic. See the [migration guide](docs/en-US/Migration-Guide.md) and [container guide](docs/en-US/Container-Images.md) for earlier-version compatibility requirements.

[Compare 7.3.4...7.4.0](https://github.com/soulteary/webhook/compare/7.3.4...7.4.0)

## 7.3.4 - 2026-11-02

### Changed

- Refresh first-party kit dependencies and their compatible transitive requirements; the Go toolchain remains 1.27.3.
- Verify distributed rate limiting against a real password-protected Redis server, including shared quotas, independent keys and expiration. Existing in-memory and Redis configuration paths are retained.
- Synchronize container, verification and migration examples with this release.

### Upgrade from 7.3.3

No hook-definition or configuration migration is required. Retain your hook files, configuration, credentials and mounted data. Keep the same runtime variant (`core`, `runner` or `extended`) and change only its version suffix to `7.3.4`. The unprefixed tag continues to select the core variant, which does not contain a shell.

Validate hooks and test a representative signed request before switching production traffic. Redis-backed deployments should verify shared rate-limit behavior after replacement. See the [migration guide](docs/en-US/Migration-Guide.md) and [container guide](docs/en-US/Container-Images.md) for earlier-version compatibility requirements.

[Compare 7.3.3...7.3.4](https://github.com/soulteary/webhook/compare/7.3.3...7.3.4)
