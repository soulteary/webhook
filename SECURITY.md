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

## Reporting a Vulnerability

If you find or encounter security-related issues, you are welcome to raise them in [Issues](https://github.com/soulteary/webhook/issues).
