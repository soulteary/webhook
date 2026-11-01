# Changelog

## 7.3.4 - 2026-11-02

### Changed

- Refresh first-party kit dependencies and their compatible transitive requirements; the Go toolchain remains 1.27.3.
- Verify distributed rate limiting against a real password-protected Redis server, including shared quotas, independent keys and expiration. Existing in-memory and Redis configuration paths are retained.
- Synchronize container, verification and migration examples with this release.

### Upgrade from 7.3.3

No hook-definition or configuration migration is required. Retain your hook files, configuration, credentials and mounted data. Keep the same runtime variant (`core`, `runner` or `extended`) and change only its version suffix to `7.3.4`. The unprefixed tag continues to select the core variant, which does not contain a shell.

Validate hooks and test a representative signed request before switching production traffic. Redis-backed deployments should verify shared rate-limit behavior after replacement. See the [migration guide](docs/en-US/Migration-Guide.md) and [container guide](docs/en-US/Container-Images.md) for earlier-version compatibility requirements.

[Compare 7.3.3...7.3.4](https://github.com/soulteary/webhook/compare/7.3.3...7.3.4)
