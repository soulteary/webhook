# Examples

This directory contains example configurations and setups for webhook.

| Directory | Description |
|-----------|-------------|
| **configs/** | General hook configuration examples: `hooks.yaml`, `hooks.json`, and Go template variants (`.tmpl`) for dynamic config. Use these as a starting point for your own `hooks.yaml` or `hooks.json`. |
| **lark/** | [Lark (Feishu)](https://www.lark.com/) integration: `hook-lark.yaml` plus a sample script `send-lark-message.sh` and `docker-compose.yml` to run webhook and send messages to Lark. |
| **multi-webhook/** | Multiple hooks and/or multiple webhook instances: separate hook files under `hooks/`, a `trigger.sh` script to call them, and `docker-compose.yml` to run a multi-instance setup. |

For full documentation, see the [English docs](../docs/en-US/) or [中文文档](../docs/zh-CN/).
