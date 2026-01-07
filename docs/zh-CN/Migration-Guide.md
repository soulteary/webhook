# 迁移指南

本指南帮助您从 Webhook 的先前版本或原始 webhook 项目迁移到此 fork。

## 目录

1. [从原始 Webhook 迁移](#从原始-webhook-迁移)
2. [版本间升级](#版本间升级)
3. [破坏性更改](#破坏性更改)
4. [配置迁移](#配置迁移)
5. [功能添加](#功能添加)
6. [迁移检查清单](#迁移检查清单)

---

## 从原始 Webhook 迁移

此 fork (soulteary/webhook) 基于原始 webhook 项目 (adnanh/webhook)，在安全性、性能和功能方面有显著改进。

### 主要差异

1. **安全增强:**
   - 命令路径白名单
   - 参数验证和限制
   - 严格模式防止命令注入
   - 改进的日志记录和脱敏

2. **性能改进:**
   - 更好的并发控制
   - 可配置的超时
   - 速率限制支持
   - 优化的请求处理

3. **新功能:**
   - Prometheus 指标端点
   - 健康检查端点
   - 增强的错误处理
   - 更好的日志记录和调试

4. **配置兼容性:**
   - 与原始 hook 配置完全兼容
   - 额外的安全和性能参数
   - 与现有设置向后兼容

### 迁移步骤

1. **备份当前配置:**
   ```bash
   cp hooks.json hooks.json.backup
   cp /path/to/webhook /path/to/webhook.backup
   ```

2. **下载新版本:**
   ```bash
   # 从发布页面下载
   wget https://github.com/soulteary/webhook/releases/latest/download/webhook-linux-amd64
   
   # 或使用 Docker
   docker pull soulteary/webhook:latest
   ```

3. **测试配置:**
   ```bash
   # 验证配置
   ./webhook -validate-config -hooks hooks.json
   ```

4. **逐步迁移:**
   - 从测试环境开始
   - 如果可能，并行运行两个版本
   - 监控问题
   - 逐步迁移生产环境

5. **更新配置（可选）:**
   - 添加安全参数（推荐）
   - 配置速率限制
   - 设置监控端点

### 推荐的安全更新

迁移后，增强安全性：

```bash
# 添加命令路径白名单
webhook \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -hooks hooks.json

# 或通过环境变量
export ALLOWED_COMMAND_PATHS="/usr/bin,/opt/scripts"
export STRICT_MODE=true
webhook -hooks hooks.json
```

---

## 版本间升级

### 一般升级流程

1. **审查发布说明:**
   - 检查破坏性更改
   - 审查新功能
   - 注意已弃用的功能

2. **备份:**
   ```bash
   # 备份配置
   cp hooks.json hooks.json.backup
   
   # 备份二进制文件
   cp /usr/local/bin/webhook /usr/local/bin/webhook.backup
   ```

3. **在预发布环境测试:**
   - 将新版本部署到预发布环境
   - 运行完整测试套件
   - 验证所有 hook 正常工作

4. **更新:**
   ```bash
   # 下载新版本
   wget https://github.com/soulteary/webhook/releases/download/vX.X.X/webhook-linux-amd64
   
   # 替换二进制文件
   sudo mv webhook-linux-amd64 /usr/local/bin/webhook
   sudo chmod +x /usr/local/bin/webhook
   ```

5. **重启服务:**
   ```bash
   # systemd
   sudo systemctl restart webhook
   
   # Docker
   docker restart webhook-container
   ```

6. **验证:**
   ```bash
   # 检查版本
   webhook -version
   
   # 检查健康状态
   curl http://localhost:9000/health
   
   # 测试 hook
   curl http://localhost:9000/hooks/test-hook
   ```

### 特定版本升级

#### 升级到 3.6.x

**新功能:**
- Prometheus 指标端点
- 健康检查端点
- 增强的错误响应
- 改进的日志记录

**配置更改:**
- 无破坏性更改
- 可选：添加安全参数
- 可选：配置速率限制

**迁移:**
```bash
# 无需配置更改
# 可选：添加安全设置
webhook \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -hooks hooks.json
```

#### 升级到 3.5.x

**新功能:**
- 速率限制支持
- 增强的超时配置
- 改进的并发控制

**配置更改:**
- 速率限制的新可选参数
- 新的超时参数

**迁移:**
```bash
# 现有配置按原样工作
# 可选：启用速率限制
webhook \
  -rate-limit-enabled \
  -rate-limit-rps=100 \
  -rate-limit-burst=20 \
  -hooks hooks.json
```

---

## 破坏性更改

### 配置文件格式

**状态:** Hook 配置格式无破坏性更改。

Hook 配置保持完全兼容。所有现有配置无需修改即可工作。

### 命令行参数

**状态:** 向后兼容，有新增。

所有原始命令行参数都受支持。新参数是可选的，具有合理的默认值。

### API 端点

**状态:** 完全向后兼容。

所有原始端点按以前的方式工作。新端点（`/health`、`/metrics`）是新增的，不影响现有功能。

### 行为更改

1. **错误响应:**
   - 错误响应现在包括带请求 ID 的 JSON 格式
   - 仍支持纯文本格式以保持向后兼容
   - 更详细的错误消息

2. **日志记录:**
   - 增强的日志格式
   - 请求 ID 跟踪
   - 更好的错误上下文

3. **安全性:**
   - 更严格的默认行为（可配置）
   - 增强的验证
   - 更好的错误处理

---

## 配置迁移

### 添加安全参数

**之前:**
```bash
webhook -hooks hooks.json
```

**之后（推荐）:**
```bash
webhook \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -max-arg-length=1048576 \
  -max-total-args-length=10485760 \
  -max-args-count=1000 \
  -hooks hooks.json
```

### 添加性能参数

**之前:**
```bash
webhook -hooks hooks.json
```

**之后（可选）:**
```bash
webhook \
  -max-concurrent-hooks=20 \
  -hook-timeout-seconds=60 \
  -rate-limit-enabled \
  -rate-limit-rps=100 \
  -hooks hooks.json
```

### 环境变量

您可以从命令行参数迁移到环境变量：

**之前:**
```bash
webhook -hooks hooks.json -verbose -port 9000
```

**之后:**
```bash
export HOOKS=/path/to/hooks.json
export VERBOSE=true
export PORT=9000
webhook
```

### Docker 迁移

**之前（原始）:**
```bash
docker run -d \
  -p 9000:9000 \
  -v /path/to/hooks.json:/etc/webhook/hooks.json \
  adnanh/webhook:latest
```

**之后（此 Fork）:**
```bash
docker run -d \
  -p 9000:9000 \
  -v /path/to/hooks.json:/etc/webhook/hooks.json \
  -e ALLOWED_COMMAND_PATHS="/usr/bin,/opt/scripts" \
  -e STRICT_MODE=true \
  soulteary/webhook:latest
```

---

## 功能添加

### 新端点

1. **健康检查:**
   ```bash
   curl http://localhost:9000/health
   # 响应: {"status":"ok"}
   ```

2. **指标:**
   ```bash
   curl http://localhost:9000/metrics
   # Prometheus 格式指标
   ```

### 新配置选项

1. **安全:**
   - `--allowed-command-paths`
   - `--strict-mode`
   - `--max-arg-length`
   - `--max-total-args-length`
   - `--max-args-count`

2. **性能:**
   - `--rate-limit-enabled`
   - `--rate-limit-rps`
   - `--rate-limit-burst`
   - `--max-concurrent-hooks`
   - `--hook-timeout-seconds`
   - `--hook-execution-timeout`

3. **HTTP 服务器:**
   - `--read-header-timeout-seconds`
   - `--read-timeout-seconds`
   - `--write-timeout-seconds`
   - `--idle-timeout-seconds`
   - `--max-header-bytes`

### 增强功能

1. **错误处理:**
   - JSON 错误响应
   - 请求 ID 跟踪
   - 更好的错误上下文

2. **日志记录:**
   - 结构化日志
   - 请求 ID 关联
   - 增强的调试输出

3. **监控:**
   - Prometheus 指标
   - 健康检查端点
   - 系统指标

---

## 迁移检查清单

### 迁移前

- [ ] 审查发布说明和变更日志
- [ ] 备份当前配置
- [ ] 备份当前二进制文件/容器
- [ ] 在预发布环境测试
- [ ] 记录当前设置

### 迁移步骤

- [ ] 下载新版本
- [ ] 验证配置
- [ ] 更新配置（如需要）
- [ ] 部署到预发布环境
- [ ] 运行测试套件
- [ ] 监控问题
- [ ] 部署到生产环境
- [ ] 验证功能

### 迁移后

- [ ] 验证所有 hook 工作
- [ ] 检查监控端点
- [ ] 审查日志中的错误
- [ ] 更新文档
- [ ] 培训团队新功能
- [ ] 设置告警（如果使用指标）

### 安全增强（推荐）

- [ ] 添加命令路径白名单
- [ ] 启用严格模式
- [ ] 设置参数限制
- [ ] 配置速率限制
- [ ] 审查和更新触发规则
- [ ] 设置监控

### 性能调优（可选）

- [ ] 配置并发限制
- [ ] 设置适当的超时
- [ ] 启用速率限制
- [ ] 设置指标收集
- [ ] 配置告警

---

## 回滚程序

如果需要回滚：

1. **停止当前服务:**
   ```bash
   sudo systemctl stop webhook
   # 或
   docker stop webhook-container
   ```

2. **恢复备份:**
   ```bash
   # 恢复二进制文件
   sudo cp /usr/local/bin/webhook.backup /usr/local/bin/webhook
   
   # 恢复配置（如果已更改）
   cp hooks.json.backup hooks.json
   ```

3. **重启服务:**
   ```bash
   sudo systemctl start webhook
   # 或
   docker start webhook-container
   ```

4. **验证:**
   ```bash
   curl http://localhost:9000/health
   ```

---

## 获取帮助

如果在迁移过程中遇到问题：

1. **查看文档:**
   - [故障排查指南](Troubleshooting.md)
   - [API 参考](API-Reference.md)
   - [配置参数](CLI-ENV.md)

2. **审查日志:**
   ```bash
   # 启用详细日志
   webhook -hooks hooks.json -verbose
   
   # 检查错误
   grep -i error /var/log/webhook.log
   ```

3. **测试配置:**
   ```bash
   webhook -validate-config -hooks hooks.json
   ```

4. **报告问题:**
   - 包含版本信息
   - 提供配置（已脱敏）
   - 包含相关日志
   - 描述采取的迁移步骤

---

## 其他资源

- [安全最佳实践](Security-Best-Practices.md) - 安全建议
- [性能调优](Performance-Tuning.md) - 性能优化
- [故障排查指南](Troubleshooting.md) - 常见问题和解决方案
- [API 参考](API-Reference.md) - API 文档

