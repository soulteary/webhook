# 什么是 WebHook (歪脖虎克)?

[![Release](https://github.com/soulteary/webhook/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/soulteary/webhook/actions/workflows/build.yml) [![CodeQL](https://github.com/soulteary/webhook/actions/workflows/codeql.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/codeql.yml) [![Security Scan](https://github.com/soulteary/webhook/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/webhook/actions/workflows/scan.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/webhook)](https://goreportcard.com/report/github.com/soulteary/webhook) [![codecov](https://codecov.io/gh/soulteary/webhook/branch/main/graph/badge.svg?token=V9E9VZR967)](https://codecov.io/gh/soulteary/webhook)


 <img src="./docs/logo/logo-600x600.jpg" alt="Webhook" align="left" width="180" />
 
 **WebHook（歪脖虎克）** 是一个用 Go 语言编写的轻量、安全、高度可配置的 HTTP Webhook 服务器。它允许你创建 HTTP 端点来触发自定义命令或脚本，非常适合自动化部署、CI/CD 流水线以及各种服务集成。

## ✨ 核心特性

- 🔒 **安全优先**：命令路径白名单、参数验证、严格模式和安全日志记录
- ⚡ **高性能**：可配置并发、速率限制和优化的请求处理
- 🎯 **灵活配置**：支持 JSON 和 YAML 配置文件，支持 Go 模板
- 🔐 **高级认证**：多种触发规则类型，包括 HMAC 签名验证、IP 白名单和自定义规则
- 📊 **可观测性**：内置 Prometheus 指标、健康检查端点和全面的日志记录
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

# 🚀 快速开始

几分钟内即可上手使用 WebHook。

## 安装

### 方式一：预编译二进制文件

[![](.github/release.png)](https://github.com/soulteary/webhook/releases)

从 [发布页面](https://github.com/soulteary/webhook/releases) 下载适用于 Linux、macOS 和 Windows 的预编译二进制文件。

### 方式二：Docker

![](.github/dockerhub.png)

```bash
# 最新稳定版本
docker pull soulteary/webhook:latest

# 特定版本
docker pull soulteary/webhook:5.0.0

# 包含调试工具的扩展版本
docker pull soulteary/webhook:extend-5.0.0
```

### 方式三：从源码构建

```bash
git clone https://github.com/soulteary/webhook.git
cd webhook
go build
```

## 配置

**📚 完整文档请查看 [中文文档](./docs/zh-CN/) 或 [英文文档](./docs/en-US/)**

### 基础示例

创建一个 `hooks.json` 文件（或使用 `hooks.yaml` 作为 YAML 格式）来定义你的 webhook：

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

### 运行 WebHook

```bash
./webhook -hooks hooks.json -verbose
```

服务器将在默认的 9000 端口启动。你的钩子将在以下地址可用：

```
http://yourserver:9000/hooks/redeploy-webhook
```

### 保护你的钩子

**重要提示**：上面的示例没有身份验证。在生产环境中请始终使用触发规则！

**示例：带密钥令牌的安全钩子**

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

现在钩子只能通过以下方式触发：`http://yourserver:9000/hooks/secure-deploy?token=your-secret-token`

更多安全选项，请查看：
- [安全最佳实践](docs/zh-CN/Security-Best-Practices.md) - 全面的安全指南
- [钩子匹配规则](docs/zh-CN/Hook-Rules.md) - 所有可用的触发规则
- [安全策略](SECURITY.md) - 内置安全功能

## 其他功能

- **表单数据支持**：解析 multipart 表单数据和文件上传 - 查看 [表单数据](docs/zh-CN/Request-Values.md)
- **模板支持**：使用 `-template` 标志在配置文件中使用 Go 模板 - 查看 [配置模版](docs/zh-CN/Templates.md)
- **HTTPS**：使用反向代理（nginx、Traefik、Caddy）提供 HTTPS 支持
- **CORS**：使用 `-header name=value` 设置自定义响应头，包括 CORS 响应头
- **热重载**：使用 `-hotreload` 或 `kill -USR1` 无需重启即可更新配置

更多示例和用例，请查看 [钩子示例](docs/zh-CN/Hook-Examples.md)。

## 文档

### 核心文档
- [钩子定义](docs/zh-CN/Hook-Definition.md) - 完整的钩子配置参考
- [钩子匹配规则](docs/zh-CN/Hook-Rules.md) - 触发规则和条件
- [配置参数](docs/zh-CN/CLI-ENV.md) - 命令行参数和配置
- [配置模版](docs/zh-CN/Templates.md) - 在配置中使用 Go 模板
- [请求值引用](docs/zh-CN/Request-Values.md) - 访问请求数据
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

## 关于此 Fork

本项目是原始 [webhook](https://github.com/adnanh/webhook) 项目的维护分支，专注于：

- **安全性**：定期安全更新、漏洞修复和增强的安全功能
- **维护性**：积极开发、依赖更新和错误修复
- **功能**：社区驱动的改进和新功能
- **文档**：完整的中英文文档

我们的目标是为社区提供一个可靠、安全且维护良好的 webhook 服务器。

几年前，我曾经提交过一个[改进版本的 PR](https://github.com/adnanh/webhook/pull/570)，但是因为种种原因被作者忽略，目前原始项目的版本和维护也一直停留在 2024 年，**与其继续使用明知道不可靠的程序，不如将它变的可靠。**

除了更容易从社区合并未被原始仓库作者合并的社区功能外，还可以快速对有安全风险的依赖作更新，并且文档友好、利于调试，能够快速上手。

[w]: https://github.com/soulteary/webhook
