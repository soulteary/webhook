# 安全最佳实践

本文档提供了在生产环境中部署和使用 Webhook 的全面安全最佳实践。

## 目录

1. [命令执行安全](#命令执行安全)
2. [网络安全](#网络安全)
3. [身份验证和授权](#身份验证和授权)
4. [配置安全](#配置安全)
5. [文件系统安全](#文件系统安全)
6. [日志和监控](#日志和监控)
7. [部署安全](#部署安全)
8. [常见安全陷阱](#常见安全陷阱)

---

## 命令执行安全

### 1. 使用命令路径白名单

在生产环境中**始终**使用 `--allowed-command-paths` 标志来限制可以执行的命令。

```bash
webhook -allowed-command-paths="/usr/bin,/opt/scripts"
```

这可以防止未经授权的命令执行，即使攻击者获得了触发 hook 的访问权限。

**最佳实践:**
- 尽可能使用特定文件路径而不是目录
- 定期审查和更新白名单
- 为不同环境（开发、预发布、生产）使用不同的白名单

**示例:**
```bash
# 好：特定文件
-allowed-command-paths="/usr/bin/git,/opt/scripts/deploy.sh"

# 更好：具有有限访问权限的特定目录
-allowed-command-paths="/opt/scripts"
```

### 2. 启用严格模式

严格模式拒绝包含潜在危险 shell 字符的参数。

```bash
webhook -strict-mode
```

**它阻止的内容:**
- Shell 特殊字符：`;`, `|`, `&`, `` ` ``, `$`, `()`, `{}` 等
- 命令链接尝试
- 变量扩展尝试

**何时使用:**
- 始终在生产环境中使用
- 当接受可能传递给命令的用户输入时
- 当您不需要 shell 功能时

### 3. 设置参数限制

限制命令参数的大小和数量，以防止资源耗尽攻击。

```bash
webhook \
  -max-arg-length=1048576 \
  -max-total-args-length=10485760 \
  -max-args-count=1000
```

**推荐值:**
- `max-arg-length`: 1MB (1048576 字节) - 足以满足大多数用例
- `max-total-args-length`: 10MB (10485760 字节) - 防止内存耗尽
- `max-args-count`: 1000 - 防止参数泛滥

### 4. 永远不要启用自动 Chmod

在生产环境中**永远不要**使用 `--allow-auto-chmod`。这是一个安全风险，可能导致权限提升。

```bash
# ❌ 错误 - 永远不要在生产环境中这样做
webhook -allow-auto-chmod

# ✅ 正确 - 手动设置权限
chmod +x /path/to/script.sh
webhook -hooks hooks.json
```

---

## 网络安全

### 1. 使用 HTTPS

Webhook 不直接提供 HTTPS。始终使用反向代理（nginx、Traefik、Caddy）来提供 HTTPS。

**nginx 配置示例:**
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

### 2. 限制网络访问

- 使用防火墙规则限制对 webhook 服务器的访问
- 仅允许来自受信任源的连接
- 尽可能使用 VPN 或专用网络

**iptables 规则示例:**
```bash
# 仅允许来自特定 IP 的连接
iptables -A INPUT -p tcp --dport 9000 -s 192.168.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 9000 -j DROP
```

### 3. 使用速率限制

启用速率限制以防止滥用和 DoS 攻击。

```bash
webhook \
  -rate-limit-enabled \
  -rate-limit-rps=10 \
  -rate-limit-burst=20
```

**推荐值:**
- `rate-limit-rps`: 每秒 10-50 个请求（根据您的需求调整）
- `rate-limit-burst`: RPS 值的 2 倍

---

## 身份验证和授权

### 1. 使用触发规则

始终使用触发规则来限制谁可以触发 hook。

**示例：密钥参数**
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

**示例：HMAC 签名验证**
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

### 2. IP 白名单

将 hook 访问限制为特定 IP 地址或范围。

```json
{
  "id": "internal-deploy",
  "execute-command": "/opt/scripts/deploy.sh",
  "trigger-rule": {
    "match": {
      "type": "ip-whitelist",
      "ip-range": "192.168.1.0/24"
    }
  }
}
```

### 3. 使用强密钥

- 使用长随机密钥（至少 32 个字符）
- 安全存储密钥（环境变量、密钥管理系统）
- 定期轮换密钥
- 永远不要将密钥提交到版本控制

---

## 配置安全

### 1. 保护配置文件

- 设置适当的文件权限（对 webhook 用户只读）
- 对敏感值使用环境变量
- 永远不要将密钥提交到配置文件
- 使用带有环境变量替换的配置模板

**示例:**
```bash
# 设置限制性权限
chmod 600 hooks.json
chown webhook:webhook hooks.json
```

### 2. 验证配置

- 在部署前始终验证 hook 配置
- 在测试期间使用 `-verbose` 模式以捕获配置错误
- 定期审查 hook 配置

### 3. 使用最小权限

- 使用专用的非特权用户运行 webhook
- 使用 `--setuid` 和 `--setgid` 在绑定端口后降低权限

```bash
# 创建专用用户
useradd -r -s /bin/false webhook

# 使用降低的权限运行
webhook -setuid $(id -u webhook) -setgid $(id -g webhook)
```

---

## 文件系统安全

### 1. 安全脚本权限

- 仅对必要的脚本设置可执行权限
- 使用限制性权限（例如，`750` 或 `700`）
- 定期审计脚本权限

```bash
# 良好的权限
chmod 750 /opt/scripts/deploy.sh
chown webhook:webhook /opt/scripts/deploy.sh
```

### 2. 工作目录安全

- 为 hook 执行使用专用目录
- 设置适当的目录权限
- 避免使用系统目录（例如，`/tmp`、`/var/tmp`）

```json
{
  "id": "deploy",
  "execute-command": "/opt/scripts/deploy.sh",
  "command-working-directory": "/opt/webhook/workspace"
}
```

### 3. 临时文件处理

- 确保脚本清理临时文件
- 使用安全的临时目录
- 设置适当的 TMPDIR 环境变量

---

## 日志和监控

### 1. 启用安全日志

- 将日志记录到具有适当权限的文件
- 定期轮换日志
- 监控日志中的可疑活动

```bash
webhook -logfile=/var/log/webhook/webhook.log
```

### 2. 监控指标

- 使用 `/metrics` 端点进行 Prometheus 监控
- 设置告警：
  - 高错误率
  - 异常请求模式
  - 资源耗尽
  - 失败的 hook 执行

### 3. 审计日志

- 记录所有 hook 执行
- 包含请求 ID 以便追踪
- 安全存储日志
- 根据合规要求保留日志

---

## 部署安全

### 1. 容器安全（Docker）

如果使用 Docker：

- 在容器中使用非 root 用户
- 限制容器功能
- 尽可能使用只读文件系统
- 扫描镜像以查找漏洞

**Dockerfile 示例:**
```dockerfile
FROM soulteary/webhook:latest

# 创建非 root 用户
RUN adduser -D -s /bin/sh webhook

# 切换到非 root 用户
USER webhook

# 运行 webhook
CMD ["webhook", "-hooks", "/etc/webhook/hooks.json"]
```

### 2. 系统更新

- 保持 webhook 二进制文件更新
- 定期更新操作系统
- 监控安全公告
- 使用自动化安全扫描

### 3. 备份和恢复

- 定期备份 hook 配置
- 测试恢复程序
- 记录事件响应程序

---

## 常见安全陷阱

### 1. ❌ 在没有身份验证的情况下暴露 Hook

```json
// 错误 - 任何人都可以触发这个
{
  "id": "deploy",
  "execute-command": "/opt/scripts/deploy.sh"
}
```

```json
// 正确 - 需要密钥令牌
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

### 2. ❌ 在命令中直接使用用户输入

```json
// 错误 - 容易受到命令注入攻击
{
  "id": "run-command",
  "execute-command": "/bin/sh",
  "pass-arguments-to-command": [
    {"source": "payload", "name": "command"}
  ]
}
```

```json
// 正确 - 验证和清理输入
{
  "id": "run-command",
  "execute-command": "/opt/scripts/safe-runner.sh",
  "pass-arguments-to-command": [
    {"source": "payload", "name": "command"}
  ]
}
```

### 3. ❌ 以 Root 身份运行

```bash
# 错误
sudo webhook -hooks hooks.json

# 正确
webhook -setuid $(id -u webhook) -setgid $(id -g webhook) -hooks hooks.json
```

### 4. ❌ 在配置文件中存储密钥

```json
// 错误 - 配置文件中的密钥
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
# 正确 - 使用环境变量
export WEBHOOK_SECRET="my-secret-key-12345"
```

然后使用模板：
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

## 安全检查清单

在部署到生产环境之前，确保：

- [ ] 已配置命令路径白名单
- [ ] 已启用严格模式
- [ ] 已设置参数限制
- [ ] 已配置 HTTPS（通过反向代理）
- [ ] 已启用速率限制
- [ ] 所有 hook 都有触发规则
- [ ] Webhook 以非 root 用户身份运行
- [ ] 配置文件具有限制性权限
- [ ] 已启用日志记录和监控
- [ ] 密钥安全存储（不在配置文件中）
- [ ] 网络访问受到限制
- [ ] 定期应用安全更新
- [ ] 已制定备份和恢复程序

---

## 报告安全问题

如果您发现安全漏洞，请负责任地报告：

1. **不要**打开公开问题
2. 通过以下方式之一报告安全问题：
   - 在 GitHub 上创建[安全公告](https://github.com/soulteary/webhook/security/advisories/new)
   - 通过 GitHub 联系维护者（如果可用）
3. 提供有关漏洞的详细信息，包括：
   - 漏洞描述
   - 重现步骤（如适用）
   - 潜在影响
   - 建议的修复方案（如果您有）
4. 在公开披露之前留出时间解决问题

有关更多信息，请参阅[安全策略](../SECURITY.md)。

---

## 其他资源

- [安全策略](../SECURITY.md)
- [Hook 规则](Hook-Rules.md) - 身份验证和授权规则
- [配置参数](CLI-ENV.md) - 安全相关参数
- [API 参考](API-Reference.md) - API 安全注意事项

