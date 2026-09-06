# 容器镜像

WebHook 7.2.0 会向 Docker Hub 和 GHCR 发布三种多架构运行时镜像。所有变体
都以 UID/GID `65532` 运行，工作目录均为 `/var/lib/webhook`，并包含同一个
静态链接的 WebHook 可执行文件。

| 变体 | 版本标签 | 运行时内容 | 适用场景 |
|---|---|---|---|
| Core | `core-7.2.0` | `scratch`、CA 证书、WebHook | 所有被执行命令都是挂载进来的静态可执行文件 |
| Runner | `runner-7.2.0` | Alpine、BusyBox 工具、Bash、CA 证书 | Hook 执行普通 POSIX Shell 或 Bash 脚本 |
| Extended | `extended-7.2.0` | Runner 加 curl、jq 和 yq | Hook 明确依赖网络或 JSON/YAML 命令行工具 |

无前缀的 `7.2.0` 和 `latest` 都是 `core-7.2.0` 的别名。生产环境应固定
版本标签，最好进一步固定镜像 Digest。两个镜像仓库使用相同的标签规则：

```text
soulteary/webhook:<tag>
ghcr.io/soulteary/webhook:<tag>
```

## 安全边界

`core` 不包含 Shell、包管理器和通用用户空间，攻击面最小，但它本身无法执行
Shell 脚本。

`runner` 是脚本型 Hook 的常规选择。它提供 `/bin/sh`、Bash、BusyBox Applet
和公共 CA 证书，但不额外提供 curl、jq 或 yq。

`extended` 有意加入能够发起网络请求和处理结构化数据的工具，适合排障和部分
集成，但同时增加了软件包数量，也扩大了 Hook 被利用后可使用的能力。不需要
额外工具时应优先选择 `runner`。

无论选择哪一种变体，都建议同时使用只读根文件系统、只读配置挂载、仅在必要
位置提供可写 tmpfs，并配合 Secure Profile 的命令白名单。挂载文件必须允许
UID/GID `65532` 访问。

## 从 7.1 迁移

旧名称 `extend-7.2.0` 会继续作为 `extended-7.2.0` 的兼容别名。现有部署
不必立即改名，但新配置应使用 `extended-*`。

旧版 Extended 镜像经常承担所有脚本执行场景。7.2 应按实际依赖选择最小运行时：

- 只有 Shell 依赖的 Hook 迁移到 `runner-7.2.0`。
- 需要 curl、jq 或 yq 的 Hook 使用 `extended-7.2.0`。
- 只有命令本身为静态可执行文件时才使用 `core-7.2.0`。

默认镜像仍不包含 Shell，因此不能把现有的脚本型部署从 `extend-*` 直接改成无
前缀的版本标签。
