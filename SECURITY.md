# Security Policy

## Supported Versions

Current support status of each version.

| Version | Supported          |
| ------- | ------------------ |
| 3.4.x   | :white_check_mark: |
| < 3.4.1   | :x:                |

## Security Features

### Command Injection Protection

Webhook includes several security features to help prevent command injection attacks:

1. **Command Path Whitelist**: Use `--allowed-command-paths` (or `ALLOWED_COMMAND_PATHS` environment variable) to restrict which commands can be executed. Only commands from the whitelist will be allowed to run.

2. **Argument Validation**: 
   - `--max-arg-length`: Limit the maximum length of a single argument (default: 1MB)
   - `--max-total-args-length`: Limit the total length of all arguments (default: 10MB)
   - `--max-args-count`: Limit the maximum number of arguments (default: 1000)

3. **Strict Mode**: Enable `--strict-mode` to reject arguments containing potentially dangerous characters (shell special characters like `;`, `|`, `&`, `` ` ``, `$`, etc.)

4. **Secure Logging**: All command executions are logged with sensitive information (passwords, tokens, keys) automatically sanitized.

**Best Practices**:
- Always use command path whitelist in production environments
- Enable strict mode for enhanced security
- Set appropriate limits for argument length and count
- Regularly review and update your whitelist
- Never enable `--allow-auto-chmod` in production (it's a security risk)

For more details, see the [Configuration Parameters documentation](docs/zh-CN/CLI-ENV.md) or [Webhook Parameters](docs/en-US/Webhook-Parameters.md).

## Comprehensive Security Guide

For a complete guide on securing your Webhook deployment, including network security, authentication, configuration security, and deployment best practices, see:

- **[Security Best Practices (English)](docs/en-US/Security-Best-Practices.md)** - Comprehensive security guide
- **[安全最佳实践 (中文)](docs/zh-CN/Security-Best-Practices.md)** - 全面的安全指南

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Report security concerns through one of the following methods:
   - Open a [Security Advisory](https://github.com/soulteary/webhook/security/advisories/new) on GitHub
   - Contact the maintainers through GitHub (if available)
3. Provide detailed information about the vulnerability, including:
   - Description of the vulnerability
   - Steps to reproduce (if applicable)
   - Potential impact
   - Suggested fix (if you have one)
4. Allow time for the issue to be addressed before public disclosure

For more comprehensive security guidance, see the [Security Best Practices documentation](docs/en-US/Security-Best-Practices.md).
