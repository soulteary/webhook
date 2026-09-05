# 什么是 WebHook (歪脖虎克)?

[![Release](https://github.com/soulteary/webhook/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/soulteary/webhook/actions/workflows/build.yml) [![CodeQL](https://github.com/soulteary/webhook/actions/workflows/codeql.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/codeql.yml) [![Security Scan](https://github.com/soulteary/webhook/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/scan.yml) [![Benchmarks](https://github.com/soulteary/webhook/actions/workflows/benchmark.yml/badge.svg?branch=main)](https://github.com/soulteary/webhook/actions/workflows/benchmark.yml) [![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)


 <img src="./docs/logo/logo-600x600.jpg" alt="Webhook" align="left" width="180" />
 
 **WebHook（歪脖虎克）** 是面向自托管、边缘节点和内网环境的安全、可观测 webhook-to-command runner。它以兼容 [adnanh/webhook](https://github.com/adnanh/webhook) Hook 配置为目标，并补充命令执行、流量、审计与链路追踪方面的生产控制能力。

## ✨ 核心特性

- 🔒 **安全优先**：命令路径白名单、参数验证、严格模式和安全日志记录
- ⚡ **高性能**：可配置并发、速率限制（含基于 Redis 的分布式限流）与优化的请求处理
- 🎯 **灵活配置**：支持 JSON 和 YAML 配置文件，支持 Go 模板
- 🔐 **高级认证**：多种触发规则类型，包括 HMAC 签名验证、IP 白名单和自定义规则
- 📊 **可观测性**：内置 Prometheus 指标、健康检查端点、可选 OpenAPI 规范（便于 Swagger/客户端生成）、OpenTelemetry 追踪、审计日志与全面日志
- 🐳 **容器就绪**：官方 Docker 镜像，提供多个变体
- 🌍 **国际化**：完整的中英文文档支持
- 🔄 **热重载**：无需重启服务器即可更新钩子配置

## 🚀 使用场景

- **CI/CD 自动化**：当代码推送到特定分支时自动部署应用
- **服务集成**：连接 GitHub、GitLab、Gitea 等服务到你的基础设施
- **ChatOps**：与 Slack、飞书、钉钉等聊天平台集成，通过聊天运行命令
- **监控告警**：触发对系统事件和告警的自动化响应
- **自定义工作流**：构建适合你需求的自定义自动化工作流

## 🎯 工作原理

WebHook 遵循简单、专注的方法：

1. **接收** HTTP 请求（GET、POST 等）
2. **解析** 请求头、请求体和参数
3. **验证** 触发规则和条件
4. **执行** 配置的命令，将请求数据作为参数或环境变量传递

你执行的命令完全由你决定——从简单脚本到复杂的自动化工作流。

## 兼容性与产品边界

| 范围 | 兼容性 / 行为 |
|---|---|
| Hook 配置 | 以兼容 adnanh/webhook 的 JSON/YAML 配置为目标，通常可直接加载；每次升级前仍应执行校验。 |
| 历史默认行为 | 默认 `compat` Profile 保持现有的宽松 HTTP 方法与安全能力按需开启的行为。 |
| 生产安全默认值 | `-profile secure` 默认启用仅 POST、严格参数检查、限流、请求 ID 和审计日志，并强制要求 `-allowed-command-paths`。 |
| 明确边界 | WebHook 负责受控执行本地命令，不承诺持久队列、重试、DLQ 或进程重启后的任务恢复。 |

# 🚀 快速开始

几分钟内即可上手使用 WebHook。

## 安装

### 方式一：Homebrew 或 Go install

```bash
brew install soulteary/tap/webhook

# 或通过 Go 工具链直接安装
go install github.com/soulteary/webhook@latest
```

### 方式二：预编译二进制文件

[![](.github/release.png)](https://github.com/soulteary/webhook/releases)

从 [发布页面](https://github.com/soulteary/webhook/releases) 下载适用于 Linux 和 macOS 的预编译二进制文件。

### 方式三：Docker

![](.github/dockerhub.png)

```bash
# 最新稳定版本
docker pull soulteary/webhook:latest

# 特定版本
docker pull soulteary/webhook:7.1.0

# 包含调试工具的扩展版本
docker pull soulteary/webhook:extend-7.1.0
```

两种 Release 镜像都以非 root 用户从 `/var/lib/webhook` 运行。默认镜像是基于 `scratch` 的 Core 镜像，适合挂载静态可执行文件且不依赖 Shell 的配置；`extend-*` 镜像包含 Alpine、Bash、curl、wget、jq 和 yq，适合执行 Shell 脚本。请确保挂载的 Hook、命令和审计目录可由 UID/GID `65532` 读取或写入。

需要立即发送签名请求时，可使用 [60 秒 Docker Compose 快速体验](example/quickstart/)。

### 方式四：从源码构建

```bash
git clone https://github.com/soulteary/webhook.git
cd webhook
go build
```

## 配置

**📚 完整文档请查看[版本化文档站](https://soulteary.github.io/webhook/)、[中文文档](./docs/zh-CN/)或[英文文档](./docs/en-US/)。**

启动服务前可以生成、严格校验并诊断配置：

```bash
webhook init
WEBHOOK_SECRET='替换为随机密钥' webhook validate --strict -template -hooks hooks/hooks.yaml
WEBHOOK_SECRET='替换为随机密钥' webhook doctor --strict -template -hooks hooks/hooks.yaml
```

[Hook JSON Schema](schema/hooks.schema.json) 可用于编辑器自动补全和未知字段检测；命令与 VS Code 配置详见[配置工具](docs/en-US/Configuration-Tools.md)。

### 基础示例

默认模式下，程序会使用 `./hooks` 目录扫描配置文件。你可以创建 `./hooks/hooks.yaml`（或 `./hooks/hooks.json`）来定义 webhook：

**示例：简单的部署钩子**

```json
[
  {
    "id": "redeploy-webhook",
    "execute-command": "/var/scripts/redeploy.sh",
    "command-working-directory": "/var/webhook"
  }
]
```

如果你更喜欢使用 YAML，相应的 hooks.yaml 文件内容为：

```yaml
- id: redeploy-webhook
  execute-command: "/var/scripts/redeploy.sh"
  command-working-directory: "/var/webhook"
```

### 运行 WebHook（默认目录模式）

```bash
./webhook -verbose
```

服务器将在默认的 9000 端口启动。你的钩子将在以下地址可用：

```
http://yourserver:9000/hooks/redeploy-webhook
```

如需继续使用单文件模式，仍可显式指定：

```bash
./webhook -hooks hooks.json -verbose
```

### 保护你的钩子

**重要提示**：上面的示例没有身份验证。在生产环境中请始终使用触发规则！

**示例：使用 HMAC 请求头的安全钩子**

```json
[
  {
    "id": "secure-deploy",
    "execute-command": "/var/scripts/deploy.sh",
    "http-methods": ["POST"],
    "trigger-rule": {
      "match": {
        "type": "payload-hmac-sha256",
        "secret": "replace-with-a-long-random-secret",
        "parameter": {
          "source": "header",
          "name": "X-Webhook-Signature"
        }
      }
    }
  }
]
```

使用 Secure Profile 启动（该 Profile 强制要求命令白名单）：

```bash
./webhook -profile secure \
  -allowed-command-paths=/var/scripts \
  -hooks hooks.json
```

发送原始请求体时提供 `X-Webhook-Signature: sha256=<HMAC-SHA256 十六进制值>`。建议通过[配置模板](docs/zh-CN/Templates.md)从环境变量读取密钥，避免将其提交到仓库。与 URL 查询参数 Token 不同，签名不会被复制到 URL、访问日志或浏览器历史中。

更多安全选项，请查看：
- [安全最佳实践](docs/zh-CN/Security-Best-Practices.md) - 全面的安全指南
- [钩子匹配规则](docs/zh-CN/Hook-Rules.md) - 所有可用的触发规则
- [安全策略](SECURITY.md) - 内置安全功能

## 其他功能

- **表单数据支持**：解析 multipart 表单数据和文件上传 - 查看 [表单数据](docs/zh-CN/Referencing-Request-Values.md)
- **模板支持**：使用 `-template` 标志在配置文件中使用 Go 模板 - 查看 [配置模版](docs/zh-CN/Templates.md)
- **Config UI**：同一二进制，按参数切换。使用 `-config-ui` 启用配置生成 Web UI（建议仅在调试或内网使用）；与主服务共用端口（默认 `9000`），可用 `-config-ui-path` 修改路径（尾斜杠会归一化）。目录模式（默认 `./hooks` 或显式 `-hooks-dir`）下，UI 可将生成的配置直接保存到目录，保存后可立即调用生成的 URL 验证；显式 `-hooks` 单文件模式下，仍可生成/下载但不会提供目录保存。`-urlprefix` 会影响 UI 中展示的调用 URL。详见 [配置参数](docs/zh-CN/Webhook-Parameters.md) 与 [Config UI 说明](cmd/README.md)。
- **OwlMail 联动**：接收 [OwlMail](https://github.com/soulteary/owlmail) 的签名邮件事件，使用 HMAC-SHA256 验证请求体、映射投递元数据用于链路关联，并执行受控命令。参见[联动指南](docs/zh-CN/OwlMail-Integration.md)和[可运行示例](example/owlmail/)。
- **Provider 配方**：提供由 CI 端到端验证的 GitHub、GitLab、Gitea、Harbor 和 Alertmanager 配置，见 [example/providers](example/providers/)。
- **HTTPS**：使用反向代理（nginx、Traefik、Caddy）提供 HTTPS 支持
- **CORS**：使用 `-header name=value` 设置自定义响应头，包括 CORS 响应头
- **热重载**：使用 `-hotreload` 或 `kill -USR1` 无需重启即可更新配置

更多示例和用例，请查看 [钩子示例](docs/zh-CN/Hook-Examples.md)。示例配置与用法（hooks、飞书、多实例和 OwlMail）见 [example/](example/) 目录。

## 文档

### 核心文档
- [钩子定义](docs/zh-CN/Hook-Definition.md) - 完整的钩子配置参考
- [Config UI](cmd/README.md) - 配置生成器（运行 `go run . -config-ui` 启用）
- [OwlMail 联动](docs/zh-CN/OwlMail-Integration.md) - 带签名的邮件事件转发与可运行 Compose 示例
- [钩子匹配规则](docs/zh-CN/Hook-Rules.md) - 触发规则和条件
- [配置参数](docs/zh-CN/Webhook-Parameters.md) - 命令行参数和配置
- [配置模版](docs/zh-CN/Templates.md) - 在配置中使用 Go 模板
- [请求值引用](docs/zh-CN/Referencing-Request-Values.md) - 访问请求数据
- [钩子示例](docs/zh-CN/Hook-Examples.md) - 实用示例和用例

### 高级主题
- [API 参考](docs/zh-CN/API-Reference.md) - 完整的 API 文档，包含所有端点
- [安全最佳实践](docs/zh-CN/Security-Best-Practices.md) - 全面的安全指南
- [性能调优](docs/zh-CN/Performance-Tuning.md) - 性能优化指南
- [测试指南](docs/zh-CN/Testing-Guide.md) - 如何运行测试、生成覆盖率报告以及关键测试场景
- [故障排查](docs/zh-CN/Troubleshooting.md) - 常见问题和解决方案
- [迁移指南](docs/zh-CN/Migration-Guide.md) - 从先前版本升级

### 安全
- [安全策略](SECURITY.md) - 安全功能和漏洞报告

### Release 完整性

Tag Release 会发布 SPDX SBOM、Checksum 文件的 Keyless Sigstore Bundle、已签名的多架构容器 Manifest，以及 GitHub 构建来源证明。验证示例：

```bash
gh attestation verify webhook_7.1.0_linux_amd64.tar.gz -R soulteary/webhook

cosign verify-blob \
  --bundle webhook_7.1.0_checksums.txt.sigstore.json \
  --certificate-identity-regexp='^https://github.com/soulteary/webhook/.github/workflows/build.yml@refs/tags/.+$' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  webhook_7.1.0_checksums.txt
```

## 关于此 Fork

本项目是原始 [webhook](https://github.com/adnanh/webhook) 项目的维护分支。当前支持版本为 7.x，版本支持情况见 [SECURITY.md](SECURITY.md)。

该分支专注于：

- **安全性**：定期安全更新、漏洞修复和增强的安全功能
- **维护性**：积极开发、依赖更新和错误修复
- **功能**：社区驱动的改进和新功能
- **文档**：完整的中英文文档

我们的目标是为社区提供一个可靠、安全且维护良好的 webhook 服务器。

几年前，我曾经提交过一个[改进版本的 PR](https://github.com/adnanh/webhook/pull/570)，但是因为种种原因被作者忽略，目前原始项目的版本和维护也一直停留在 2024 年，**与其继续使用明知道不可靠的程序，不如将它变的可靠。**

除了更容易从社区合并未被原始仓库作者合并的社区功能外，还可以快速对有安全风险的依赖作更新，并且文档友好、利于调试，能够快速上手。

[w]: https://github.com/soulteary/webhook
