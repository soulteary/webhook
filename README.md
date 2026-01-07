# Welcome to WebHook! [中文文档](./README-zhCN.md)

[![Release](https://github.com/soulteary/webhook/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/soulteary/webhook/actions/workflows/build.yml) [![CodeQL](https://github.com/soulteary/webhook/actions/workflows/codeql.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/codeql.yml) [![Security Scan](https://github.com/soulteary/webhook/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/scan.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/webhook)](https://goreportcard.com/report/github.com/soulteary/webhook)

 <img src="./docs/logo/logo-600x600.jpg" alt="Webhook" align="left" width="180" />
 
 [WebHook][w] is a lightweight and customizable tool written in Go that enables you to effortlessly create HTTP WebHook services. With WebHook, you can execute predefined commands and flexibly pass data from HTTP requests (including headers, body, and parameters) to your configured commands or programs. It also supports triggering hooks based on specific conditions.

For example, if you're using GitHub or Gitea, you can set up a hook with WebHook to automatically update your deployed program whenever you push changes to a specific branch of your project.

If you use Discord, Slack, or other messaging platforms, you can create an "Outgoing Webhook Integration" or "Slash Command" to run various commands on your server. You can then use the "Incoming Webhook Integration" feature of your messaging tool to report the execution results directly to you or your conversation channel.

The [WebHook][w] project has a straightforward goal: **to do exactly what it's designed for.**

- Receive requests
- Parse request headers, body, and parameters
- Verify if the hook's execution rules are met
- Pass specified parameters to the designated command via command-line arguments or environment variables

The specific commands - whether processing data, storing information, or controlling devices - are entirely up to you. WebHook's role is to accept and execute instructions at the appropriate time.

# Getting Started

Let's explore how to download the executable program and quickly set it up to connect various applications.

## Software Installation: Downloading Pre-built Programs

[![](.github/release.png)](https://github.com/soulteary/webhook/releases)

WebHook offers pre-built executable programs for various operating systems and architectures. You can download the version suitable for your platform from the [Releases page on GitHub](https://github.com/soulteary/webhook/releases).

## Software Installation: Docker

![](.github/dockerhub.png)

You can use any of the following commands to download the automatically built executable program image:

```bash
docker pull soulteary/webhook:latest
docker pull soulteary/webhook:3.6.3
```

For an extended version of the image that includes debugging tools, use:

```bash
docker pull soulteary/webhook:extend-3.6.3
```

You can then build and refine the runtime environment required for your commands based on this image.

## Program Configuration

**We recommend reading the complete documentation to fully understand the program's capabilities. [English Documentation](./docs/en-US/), [Chinese Documentation](./docs/zh-CN/)**

---

Let's define some hooks for [webhook][w] to provide HTTP services.

[webhook][w] supports both JSON and YAML configuration files. We'll start with JSON configuration.

Create an empty file named `hooks.json`. This file will contain an array of hooks that [webhook][w] will start as HTTP services. For detailed information on hook properties and usage, please refer to the [Hook Definition page](docs/en-US/Hook-Definition.md).

Here's a simple hook named `redeploy-webhook` that runs a redeployment script located at `/var/scripts/redeploy.sh`:

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

To run [webhook][w], use the following command:

```bash
$ /path/to/webhook -hooks hooks.json -verbose
```

The program will start on the default port `9000` and provide a publicly accessible HTTP service address:

```bash
http://yourserver:9000/hooks/redeploy-webhook
```

To learn how to customize IP, port, and other settings when starting [webhook][w], check out the [webhook parameters](docs/en-US/Webhook-Parameters.md) documentation.

Any HTTP `GET` or `POST` request to the service address will trigger the redeploy script.

To enhance security and prevent unauthorized access, you can use the "trigger-rule" property to specify exact conditions for hook triggering. For a detailed list of available rules and their usage, please refer to [Hook Rules](docs/en-US/Hook-Rules.md).

For additional security, WebHook includes command injection protection features such as command path whitelisting, argument validation, and strict mode. See the [Security Policy](SECURITY.md) and [Configuration Parameters](docs/en-US/Webhook-Parameters.md) for more details.

## Form Data

[webhook][w] offers limited parsing support for form data, including both values and files. For more details on how form data is handled, please refer to the [Form Data](docs/en-US/Form-Data.md) documentation.

## Templates

[webhook][w] supports parsing the hook configuration file as a Go template when using the `-template` [command line argument](docs/en-US/Webhook-Parameters.md). For more information on template usage, see [Templates](docs/en-US/Templates.md).

## Using HTTPS

While [webhook][w] serves using HTTP by default, we recommend using a reverse proxy or a service like Traefik to provide HTTPS service for enhanced security.

## Cross-Origin CORS Request Headers

To set CORS headers, use the `-header name=value` flag when starting [webhook][w]. This will ensure the appropriate CORS headers are returned with each response.

## Usage Examples

Explore various creative uses of WebHook in our [Hook Examples](docs/en-US/Hook-Examples.md) documentation.

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

# Our Motivation

We decided to fork this open-source software for two main reasons:

1. To address security issues and outdated dependencies in the original version.
2. To incorporate community-contributed features and improvements that were not merged into the original repository.

Our goal is to make WebHook more reliable, secure, and user-friendly, including improved documentation for our Chinese users.

[w]: https://github.com/soulteary/webhook
