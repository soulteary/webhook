# What's WebHook? [中文文档](./README-zhCN.md)

[![Release](https://github.com/soulteary/webhook/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/soulteary/webhook/actions/workflows/build.yml) [![CodeQL](https://github.com/soulteary/webhook/actions/workflows/codeql.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/codeql.yml) [![Security Scan](https://github.com/soulteary/webhook/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/scan.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/webhook)](https://goreportcard.com/report/github.com/soulteary/webhook)

 <img src="./docs/logo/logo-600x600.jpg" alt="Webhook" align="left" width="180" />
 
 [WebHook][w], it's a lightweight and configurable utility written in the Golang that allows you to easily and quickly create HTTP WebHook services. You can use it to execute pre-configured commands. It also enables you to flexibly pass data from the HTTP request (such as request headers, request body, and request parameters) to your pre-configured commands or programs. Of course, it also allows triggering hooks based on specific condition rules.

Here's an example: If you are using GitHub or Gitea, you can set up a hook using Webhook, so that every time you push changes to a specific branch of your project, this hook will run a script on the device where your service is running to "update the program deployment content."

If you are using Discord, Slack, or others IM, you can also set up an "Outgoing Webhook Integration" or "Slash Command" to run various commands on your server. We can handle the response content of the interface through the "Incoming Webhook Integration" feature of the chat tool, and directly report the execution results to you or your IM conversation or channel.

[WebHook][w] project goal is very simple: **only do what it is supposed to do.**

- Receive requests,
- Parse request headers, request body, and request parameters,
- Check if the execution rules specified by the hook are satisfied,
- Finally, pass the specified parameters to the designated command through command-line arguments or environment variables.

As for the specific commands, from processing data, storing data, to turning on the air conditioner or shutting down the computer using remote commands, it's all up to you. You can implement anything you want. Webhook is only responsible for accepting execution instructions at the appropriate time point.

# Getting Started

This section explains how to download and obtain the executable program, and how to quickly start the program and begin connecting various applications.

## Software Installation: Downloading Pre-built Programs

[![](.github/release.png)](https://github.com/soulteary/webhook/releases)

Webhook provides pre-built executable programs for different operating systems and architectures. You can directly download the appropriate version for your platform from the [Releases page of the project on GitHub](https://github.com/soulteary/webhook/releases).

## Software Installation: Docker

![](.github/dockerhub.png)

You can use any of the following commands to download the automatically built executable program image from this repository:

```bash
docker pull soulteary/webhook:latest
docker pull soulteary/webhook:3.6.0
```

If you wish to have some debugging tools included in the image for your convenience, you can use the following command to obtain the extended version of the image:

```bash
docker pull soulteary/webhook:extend-3.6.0
```

We can then build and refine the runtime environment required for our commands based on this image.

## Program Configuration

**It is recommended to read the complete documentation to understand the specific capabilities of the program. [English Documentation](./docs/en-US/), [Chinese Documentation](./docs/zh-CN/)**

---

We can define some hooks that you want [webhook][w] to provide HTTP services for.

[webhook][w] supports JSON or YAML configuration files. Let's first look at how to implement JSON configuration.

First, create an empty file named `hooks.json`. This file will contain an array of hooks that [webhook][w] will start as HTTP services. You can check the [Hook Definition page](docs/en-US/Hook-Definition.md) to see what properties hooks can contain and detailed descriptions of how to use them.

Let's define a simple hook named `redeploy-webhook`, which will run the redeployment script located at `/var/scripts/redeploy.sh`. Make sure your bash script has `#!/bin/sh` at the top.

Our `hooks.json` file will look like this:

```json
[
  {
    "id": "redeploy-webhook",
    "execute-command": "/var/scripts/redeploy.sh",
    "command-working-directory": "/var/webhook"
  }
]
```

If you prefer to use YAML, the corresponding content of the `hooks.yaml` file is:

```yaml
- id: redeploy-webhook
  execute-command: "/var/scripts/redeploy.sh"
  command-working-directory: "/var/webhook"
```

Next, you can execute the [webhook][w] using the following command:

```bash
$ /path/to/webhook -hooks hooks.json -verbose
```

The program will start on the default port `9000` and provide a publicly accessible HTTP service address:

```bash
http://yourserver:9000/hooks/redeploy-webhook
```

Check the [webhook parameters](docs/en-US/Webhook-Parameters.md) to learn how to set the IP, port, and other settings when starting the [webhook][w], such as hook hot-reloading, verbose output, etc.

Once any HTTP `GET` or `POST` request accesses the service address, the redeploy script you set will be executed.

However, hooks defined like this may pose a security threat to your system, as anyone who knows your endpoint can send requests and execute commands. To prevent this, you can use the "trigger-rule" property of the hook to specify the exact conditions that trigger the hook.

For example, you can use them to add a secret parameter that must be provided to successfully trigger the hook. Please refer to [Hook Rules](docs/en-US/Hook-Rules.md) for a detailed list of available rules and how to use them.

## Form Data

[webhook][w] provides limited parsing support for form data.

Form data can typically contain two types of parts: values and files.

All form _values_ are automatically added within the `payload` scope.

The `parse-parameters-as-json` setting is used to parse the given values as JSON.

All files are ignored unless one of the following criteria is met:

1. The `Content-Type` header is `application/json`.
2. The part is named in the `parse-parameters-as-json` setting.

In either case, the given file part will be parsed as JSON and added to the payload map.

## Templates

When using the `-template` [command line argument](docs/en-US/Webhook-Parameters.md), [webhook][w] can parse the hook configuration file as a Go template. For more details on template usage, please refer to [Templates](docs/en-US/Templates.md).

## Using HTTPS

[webhook][w] serves using http by default. If you want [webhook][w] to serve HTTPS using https, a simpler approach is to use a reverse proxy or a service like traefik to provide HTTPS service.

## Cross-Origin CORS Request Headers

If you want to set CORS headers, you can use the `-header name=value` flag when starting [webhook][w] to set the appropriate CORS headers that will be returned with each response.

## Usage Examples

Check out [Hook Examples](docs/en-US/Hook-Examples.md) to learn various fresh usages.

# Why Fork an Open-Source Software

There are two main reasons:

1. The `webhook` program version maintained by the original author has been slowly upgraded from a relatively outdated Go program version.
   - It includes a lot of content that is no longer needed, as well as numerous security issues that urgently need to be fixed.

2. A few years ago, I submitted an [improved version of PR](https://github.com/adnanh/webhook/pull/570), but for various reasons, it was ignored by the author. **Rather than continuing to use a program that is known to be unreliable, it's better to make it reliable.**
   - This way, in addition to making it easier to merge community features that have not been merged by the original repository author, it also allows for quick updates to dependencies with security risks. Moreover, I hope that this program will be more Chinese-friendly in the future, including the documentation.

[w]: https://github.com/soulteary/webhook
