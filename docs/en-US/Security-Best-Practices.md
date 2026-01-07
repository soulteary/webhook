# Security Best Practices

This document provides comprehensive security best practices for deploying and using Webhook in production environments.

## Table of Contents

1. [Command Execution Security](#command-execution-security)
2. [Network Security](#network-security)
3. [Authentication and Authorization](#authentication-and-authorization)
4. [Configuration Security](#configuration-security)
5. [File System Security](#file-system-security)
6. [Logging and Monitoring](#logging-and-monitoring)
7. [Deployment Security](#deployment-security)
8. [Common Security Pitfalls](#common-security-pitfalls)

---

## Command Execution Security

### 1. Use Command Path Whitelist

**Always** use the `--allowed-command-paths` flag in production to restrict which commands can be executed.

```bash
webhook -allowed-command-paths="/usr/bin,/opt/scripts"
```

This prevents unauthorized command execution even if an attacker gains access to trigger hooks.

**Best Practices:**
- Use specific file paths instead of directories when possible
- Regularly review and update the whitelist
- Use separate whitelists for different environments (dev, staging, production)

**Example:**
```bash
# Good: Specific files
-allowed-command-paths="/usr/bin/git,/opt/scripts/deploy.sh"

# Better: Specific directories with limited access
-allowed-command-paths="/opt/scripts"
```

### 2. Enable Strict Mode

Strict mode rejects arguments containing potentially dangerous shell characters.

```bash
webhook -strict-mode
```

**What it blocks:**
- Shell special characters: `;`, `|`, `&`, `` ` ``, `$`, `()`, `{}`, etc.
- Command chaining attempts
- Variable expansion attempts

**When to use:**
- Always in production
- When accepting user input that might be passed to commands
- When you don't need shell features

### 3. Set Argument Limits

Limit the size and count of command arguments to prevent resource exhaustion attacks.

```bash
webhook \
  -max-arg-length=1048576 \
  -max-total-args-length=10485760 \
  -max-args-count=1000
```

**Recommended values:**
- `max-arg-length`: 1MB (1048576 bytes) - sufficient for most use cases
- `max-total-args-length`: 10MB (10485760 bytes) - prevents memory exhaustion
- `max-args-count`: 1000 - prevents argument flooding

### 4. Never Enable Auto-Chmod

**Never** use `--allow-auto-chmod` in production. This is a security risk that can lead to privilege escalation.

```bash
# ❌ BAD - Never do this in production
webhook -allow-auto-chmod

# ✅ GOOD - Set permissions manually
chmod +x /path/to/script.sh
webhook -hooks hooks.json
```

---

## Network Security

### 1. Use HTTPS

Webhook does not provide HTTPS directly. Always use a reverse proxy (nginx, Traefik, Caddy) to provide HTTPS.

**Example nginx configuration:**
```nginx
server {
    listen 443 ssl http2;
    server_name webhook.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. Restrict Network Access

- Use firewall rules to restrict access to the webhook server
- Only allow connections from trusted sources
- Use VPN or private networks when possible

**Example iptables rules:**
```bash
# Allow only from specific IP
iptables -A INPUT -p tcp --dport 9000 -s 192.168.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 9000 -j DROP
```

### 3. Use Rate Limiting

Enable rate limiting to prevent abuse and DoS attacks.

```bash
webhook \
  -rate-limit-enabled \
  -rate-limit-rps=10 \
  -rate-limit-burst=20
```

**Recommended values:**
- `rate-limit-rps`: 10-50 requests per second (adjust based on your needs)
- `rate-limit-burst`: 2x the RPS value

---

## Authentication and Authorization

### 1. Use Trigger Rules

Always use trigger rules to restrict who can trigger hooks.

**Example: Secret parameter**
```json
{
  "id": "deploy",
  "execute-command": "/opt/scripts/deploy.sh",
  "trigger-rule": {
    "match": {
      "type": "value",
      "value": "your-secret-token",
      "parameter": {
        "source": "url",
        "name": "token"
      }
    }
  }
}
```

**Example: HMAC signature validation**
```json
{
  "id": "github-webhook",
  "execute-command": "/opt/scripts/github.sh",
  "trigger-rule": {
    "match": {
      "type": "payload-hmac-sha256",
      "secret": "your-webhook-secret",
      "parameter": {
        "source": "header",
        "name": "X-Hub-Signature-256"
      }
    }
  }
}
```

### 2. IP Whitelisting

Restrict hook access to specific IP addresses or ranges.

```json
{
  "id": "internal-deploy",
  "execute-rule": "/opt/scripts/deploy.sh",
  "trigger-rule": {
    "match": {
      "type": "ip-whitelist",
      "ip-range": "192.168.1.0/24"
    }
  }
}
```

### 3. Use Strong Secrets

- Use long, random secrets (at least 32 characters)
- Store secrets securely (environment variables, secret management systems)
- Rotate secrets regularly
- Never commit secrets to version control

---

## Configuration Security

### 1. Protect Configuration Files

- Set appropriate file permissions (read-only for webhook user)
- Use environment variables for sensitive values
- Never commit secrets to configuration files
- Use configuration templates with environment variable substitution

**Example:**
```bash
# Set restrictive permissions
chmod 600 hooks.json
chown webhook:webhook hooks.json
```

### 2. Validate Configuration

- Always validate hook configurations before deployment
- Use `-verbose` mode during testing to catch configuration errors
- Review hook configurations regularly

### 3. Use Least Privilege

- Run webhook with a dedicated, non-privileged user
- Use `--setuid` and `--setgid` to drop privileges after binding to port

```bash
# Create dedicated user
useradd -r -s /bin/false webhook

# Run with dropped privileges
webhook -setuid $(id -u webhook) -setgid $(id -g webhook)
```

---

## File System Security

### 1. Secure Script Permissions

- Set executable permissions only for necessary scripts
- Use restrictive permissions (e.g., `750` or `700`)
- Regularly audit script permissions

```bash
# Good permissions
chmod 750 /opt/scripts/deploy.sh
chown webhook:webhook /opt/scripts/deploy.sh
```

### 2. Working Directory Security

- Use dedicated directories for hook execution
- Set appropriate directory permissions
- Avoid using system directories (e.g., `/tmp`, `/var/tmp`)

```json
{
  "id": "deploy",
  "execute-command": "/opt/scripts/deploy.sh",
  "command-working-directory": "/opt/webhook/workspace"
}
```

### 3. Temporary File Handling

- Ensure scripts clean up temporary files
- Use secure temporary directories
- Set appropriate TMPDIR environment variable

---

## Logging and Monitoring

### 1. Enable Secure Logging

- Log to files with appropriate permissions
- Rotate logs regularly
- Monitor logs for suspicious activity

```bash
webhook -logfile=/var/log/webhook/webhook.log
```

### 2. Monitor Metrics

- Use the `/metrics` endpoint for Prometheus monitoring
- Set up alerts for:
  - High error rates
  - Unusual request patterns
  - Resource exhaustion
  - Failed hook executions

### 3. Audit Logging

- Log all hook executions
- Include request IDs for traceability
- Store logs securely
- Retain logs according to compliance requirements

---

## Deployment Security

### 1. Container Security (Docker)

If using Docker:

- Use non-root user in container
- Limit container capabilities
- Use read-only file systems where possible
- Scan images for vulnerabilities

**Example Dockerfile:**
```dockerfile
FROM soulteary/webhook:latest

# Create non-root user
RUN adduser -D -s /bin/sh webhook

# Switch to non-root user
USER webhook

# Run webhook
CMD ["webhook", "-hooks", "/etc/webhook/hooks.json"]
```

### 2. System Updates

- Keep the webhook binary updated
- Regularly update the operating system
- Monitor security advisories
- Use automated security scanning

### 3. Backup and Recovery

- Regularly backup hook configurations
- Test recovery procedures
- Document incident response procedures

---

## Common Security Pitfalls

### 1. ❌ Exposing Hooks Without Authentication

```json
// BAD - Anyone can trigger this
{
  "id": "deploy",
  "execute-command": "/opt/scripts/deploy.sh"
}
```

```json
// GOOD - Requires secret token
{
  "id": "deploy",
  "execute-command": "/opt/scripts/deploy.sh",
  "trigger-rule": {
    "match": {
      "type": "value",
      "value": "secret-token",
      "parameter": {"source": "url", "name": "token"}
    }
  }
}
```

### 2. ❌ Using User Input Directly in Commands

```json
// BAD - Vulnerable to command injection
{
  "id": "run-command",
  "execute-command": "/bin/sh",
  "pass-arguments-to-command": [
    {"source": "payload", "name": "command"}
  ]
}
```

```json
// GOOD - Validate and sanitize input
{
  "id": "run-command",
  "execute-command": "/opt/scripts/safe-runner.sh",
  "pass-arguments-to-command": [
    {"source": "payload", "name": "command"}
  ]
}
```

### 3. ❌ Running as Root

```bash
# BAD
sudo webhook -hooks hooks.json

# GOOD
webhook -setuid $(id -u webhook) -setgid $(id -g webhook) -hooks hooks.json
```

### 4. ❌ Storing Secrets in Configuration Files

```json
// BAD - Secret in config file
{
  "id": "webhook",
  "trigger-rule": {
    "match": {
      "type": "payload-hmac-sha256",
      "secret": "my-secret-key-12345"
    }
  }
}
```

```bash
# GOOD - Use environment variable
export WEBHOOK_SECRET="my-secret-key-12345"
```

Then use template:
```json
{
  "id": "webhook",
  "trigger-rule": {
    "match": {
      "type": "payload-hmac-sha256",
      "secret": "{{.Env.WEBHOOK_SECRET}}"
    }
  }
}
```

---

## Security Checklist

Before deploying to production, ensure:

- [ ] Command path whitelist is configured
- [ ] Strict mode is enabled
- [ ] Argument limits are set
- [ ] HTTPS is configured (via reverse proxy)
- [ ] Rate limiting is enabled
- [ ] All hooks have trigger rules
- [ ] Webhook runs as non-root user
- [ ] Configuration files have restrictive permissions
- [ ] Logging is enabled and monitored
- [ ] Secrets are stored securely (not in config files)
- [ ] Network access is restricted
- [ ] Regular security updates are applied
- [ ] Backup and recovery procedures are in place

---

## Reporting Security Issues

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

For more information, see the [Security Policy](../SECURITY.md).

---

## Additional Resources

- [Security Policy](../SECURITY.md)
- [Hook Rules](Hook-Rules.md) - Authentication and authorization rules
- [Configuration Parameters](Webhook-Parameters.md) - Security-related parameters
- [API Reference](API-Reference.md) - API security considerations

