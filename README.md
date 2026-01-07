# Welcome to WebHook! [中文文档](./README-zhCN.md)

[![Release](https://github.com/soulteary/webhook/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/soulteary/webhook/actions/workflows/build.yml) [![CodeQL](https://github.com/soulteary/webhook/actions/workflows/codeql.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/codeql.yml) [![Security Scan](https://github.com/soulteary/webhook/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/scan.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/webhook)](https://goreportcard.com/report/github.com/soulteary/webhook)

 <img src="./docs/logo/logo-600x600.jpg" alt="Webhook" align="left" width="180" />
 
 **WebHook** is a lightweight, secure, and highly configurable HTTP webhook server written in Go. It enables you to create HTTP endpoints that trigger custom commands or scripts based on incoming requests, making it perfect for automating deployments, CI/CD pipelines, and integrating with various services.

## ✨ Key Features

- 🔒 **Security First**: Command path whitelisting, argument validation, strict mode, and secure logging
- ⚡ **High Performance**: Configurable concurrency, rate limiting, and optimized request handling
- 🎯 **Flexible Configuration**: Support for JSON and YAML configuration files with Go template support
- 🔐 **Advanced Authentication**: Multiple trigger rule types including HMAC signature validation, IP whitelisting, and custom rules
- 📊 **Observability**: Built-in Prometheus metrics, health check endpoint, and comprehensive logging
- 🐳 **Container Ready**: Official Docker images with multiple variants
- 🌍 **Internationalization**: Full support for English and Chinese documentation
- 🔄 **Hot Reload**: Update hook configurations without restarting the server

## 🚀 Use Cases

- **CI/CD Automation**: Automatically deploy applications when code is pushed to specific branches
- **Service Integration**: Connect GitHub, GitLab, Gitea, and other services to your infrastructure
- **ChatOps**: Integrate with Slack, Discord, or other messaging platforms to run commands via chat
- **Monitoring & Alerts**: Trigger automated responses to system events and alerts
- **Custom Workflows**: Build custom automation workflows tailored to your needs

## 🎯 How It Works

WebHook follows a simple, focused approach:

1. **Receive** HTTP requests (GET, POST, etc.)
2. **Parse** request headers, body, and parameters
3. **Validate** trigger rules and conditions
4. **Execute** configured commands with request data passed as arguments or environment variables

The commands you execute are entirely up to you - from simple scripts to complex automation workflows.

# 🚀 Quick Start

Get up and running with WebHook in minutes.

## Installation

### Option 1: Pre-built Binaries

[![](.github/release.png)](https://github.com/soulteary/webhook/releases)

Download pre-built binaries for Linux, macOS, and Windows from the [Releases page](https://github.com/soulteary/webhook/releases).

### Option 2: Docker

![](.github/dockerhub.png)

```bash
# Latest stable version
docker pull soulteary/webhook:latest

# Specific version
docker pull soulteary/webhook:3.6.3

# Extended version with debugging tools
docker pull soulteary/webhook:extend-3.6.3
```

### Option 3: Build from Source

```bash
git clone https://github.com/soulteary/webhook.git
cd webhook
go build
```

## Configuration

**📚 For complete documentation, see [English Documentation](./docs/en-US/) or [Chinese Documentation](./docs/zh-CN/)**

### Basic Example

Create a `hooks.json` file (or `hooks.yaml` for YAML format) to define your webhooks:

**Example: Simple Deployment Hook**

```json
[
  {
    "id": "redeploy-webhook",
    "execute-command": "/var/scripts/redeploy.sh",
    "command-working-directory": "/var/webhook"
  }
]
```

If you prefer YAML, the equivalent `hooks.yaml` file would look like this:

```yaml
- id: redeploy-webhook
  execute-command: "/var/scripts/redeploy.sh"
  command-working-directory: "/var/webhook"
```

### Running WebHook

```bash
./webhook -hooks hooks.json -verbose
```

The server will start on port `9000` by default. Your hook will be available at:

```
http://yourserver:9000/hooks/redeploy-webhook
```

### Securing Your Hooks

**Important**: The example above has no authentication. Always use trigger rules in production!

**Example: Secure Hook with Secret Token**

```json
[
  {
    "id": "secure-deploy",
    "execute-command": "/var/scripts/deploy.sh",
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
]
```

Now the hook can only be triggered with: `http://yourserver:9000/hooks/secure-deploy?token=your-secret-token`

For more security options, see:
- [Security Best Practices](docs/en-US/Security-Best-Practices.md) - Comprehensive security guide
- [Hook Rules](docs/en-US/Hook-Rules.md) - All available trigger rules
- [Security Policy](SECURITY.md) - Built-in security features

## Additional Features

- **Form Data Support**: Parse multipart form data and file uploads - see [Form Data](docs/en-US/Form-Data.md)
- **Template Support**: Use Go templates in configuration files with `-template` flag - see [Templates](docs/en-US/Templates.md)
- **HTTPS**: Use a reverse proxy (nginx, Traefik, Caddy) for HTTPS support
- **CORS**: Set custom headers including CORS headers with `-header name=value`
- **Hot Reload**: Update configurations without restarting using `-hotreload` or `kill -USR1`

For more examples and use cases, check out [Hook Examples](docs/en-US/Hook-Examples.md).

## Documentation

### Core Documentation
- [Hook Definition](docs/en-US/Hook-Definition.md) - Complete hook configuration reference
- [Hook Rules](docs/en-US/Hook-Rules.md) - Trigger rules and conditions
- [Webhook Parameters](docs/en-US/Webhook-Parameters.md) - Command-line arguments and configuration
- [Templates](docs/en-US/Templates.md) - Using Go templates in configurations
- [Referencing Request Values](docs/en-US/Referencing-Request-Values.md) - Accessing request data
- [Hook Examples](docs/en-US/Hook-Examples.md) - Practical examples and use cases

### Advanced Topics
- [API Reference](docs/en-US/API-Reference.md) - Complete API documentation with all endpoints
- [Security Best Practices](docs/en-US/Security-Best-Practices.md) - Comprehensive security guide
- [Performance Tuning](docs/en-US/Performance-Tuning.md) - Performance optimization guide
- [Troubleshooting](docs/en-US/Troubleshooting.md) - Common issues and solutions
- [Migration Guide](docs/en-US/Migration-Guide.md) - Upgrading from previous versions

### Security
- [Security Policy](SECURITY.md) - Security features and vulnerability reporting

## About This Fork

This project is a maintained fork of the original [webhook](https://github.com/adnanh/webhook) project, focused on:

- **Security**: Regular security updates, vulnerability fixes, and enhanced security features
- **Maintenance**: Active development, dependency updates, and bug fixes
- **Features**: Community-driven improvements and new features
- **Documentation**: Comprehensive documentation in both English and Chinese

We aim to provide a reliable, secure, and well-maintained webhook server for the community.

[w]: https://github.com/soulteary/webhook
