# 故障排查指南

本指南帮助您诊断和解决 Webhook 的常见问题。

## 目录

1. [常见问题](#常见问题)
2. [配置问题](#配置问题)
3. [执行问题](#执行问题)
4. [网络问题](#网络问题)
5. [性能问题](#性能问题)
6. [调试技术](#调试技术)
7. [日志分析](#日志分析)

---

## 常见问题

### Hook 未找到 (404)

**症状:**
- 触发 hook 时返回 HTTP 404
- 错误消息："Hook not found."

**可能原因:**
1. URL 和配置中的 Hook ID 不匹配
2. Hook 配置文件未加载
3. Hook ID 包含无效字符

**解决方案:**

1. **验证 Hook ID:**
   ```bash
   # 检查配置中的 Hook ID
   cat hooks.json | jq '.[] | .id'
   
   # 确保 URL 完全匹配（区分大小写）
   curl http://localhost:9000/hooks/your-hook-id
   ```

2. **检查配置加载:**
   ```bash
   # 使用详细模式运行以查看加载的 hook
   webhook -hooks hooks.json -verbose
   
   # 查找 "loaded X hooks" 消息
   ```

3. **验证 Hook ID 格式:**
   - Hook ID 区分大小写
   - 避免特殊字符
   - 使用字母数字字符和连字符

### Hook 未触发

**症状:**
- Hook 存在但不执行
- 返回 "Hook rules were not satisfied"

**可能原因:**
1. 触发规则未满足
2. HTTP 方法不允许
3. 缺少必需参数

**解决方案:**

1. **检查触发规则:**
   ```bash
   # 启用调试模式以查看规则评估
   webhook -hooks hooks.json -debug
   
   # 查看日志中的规则评估结果
   ```

2. **验证 HTTP 方法:**
   ```json
   {
     "id": "my-hook",
     "http-methods": ["POST"],
     "execute-command": "/path/to/script.sh"
   }
   ```

3. **检查必需参数:**
   - 审查触发规则要求
   - 验证请求包含所有必需数据
   - 检查参数来源（请求头、URL、负载）

### 命令执行失败

**症状:**
- Hook 触发但命令失败
- HTTP 500 错误
- 命令未找到错误

**可能原因:**
1. 命令路径不在白名单中
2. 权限不足
3. 命令不存在
4. 工作目录问题

**解决方案:**

1. **检查命令路径白名单:**
   ```bash
   # 如果使用 -allowed-command-paths，验证命令是否包含在内
   webhook -allowed-command-paths="/usr/bin,/opt/scripts" -hooks hooks.json
   
   # 检查命令路径是否匹配白名单
   ```

2. **验证文件权限:**
   ```bash
   # 检查脚本权限
   ls -l /path/to/script.sh
   
   # 如果需要，使其可执行
   chmod +x /path/to/script.sh
   
   # 注意：不要在生产环境中使用 -allow-auto-chmod
   ```

3. **验证命令存在:**
   ```bash
   # 手动测试命令
   /path/to/script.sh
   
   # 检查命令是否在 PATH 中
   which command-name
   ```

4. **检查工作目录:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "script.sh",
     "command-working-directory": "/opt/scripts"
   }
   ```

### 权限被拒绝错误

**症状:**
- 日志中出现 "permission denied" 错误
- 命令执行失败

**解决方案:**

1. **设置正确的权限:**
   ```bash
   # 使脚本可执行
   chmod +x /path/to/script.sh
   
   # 设置所有权
   chown webhook:webhook /path/to/script.sh
   ```

2. **检查用户权限:**
   ```bash
   # 验证 webhook 用户可以访问文件
   sudo -u webhook ls -l /path/to/script.sh
   sudo -u webhook /path/to/script.sh
   ```

3. **使用 setuid/setgid:**
   ```bash
   # 使用特定用户/组运行
   webhook -setuid $(id -u webhook) -setgid $(id -g webhook)
   ```

---

## 配置问题

### 配置文件未加载

**症状:**
- Webhook 启动但没有可用的 hook
- 日志中出现配置错误

**解决方案:**

1. **验证 JSON/YAML 语法:**
   ```bash
   # 验证 JSON
   jq . hooks.json
   
   # 验证 YAML
   yamllint hooks.yaml
   ```

2. **检查文件路径:**
   ```bash
   # 使用绝对路径
   webhook -hooks /absolute/path/to/hooks.json
   
   # 验证文件存在
   ls -l /path/to/hooks.json
   ```

3. **检查文件权限:**
   ```bash
   # 确保文件可读
   chmod 644 hooks.json
   ```

### 模板解析错误

**症状:**
- 模板语法错误
- 环境变量未替换

**解决方案:**

1. **启用模板模式:**
   ```bash
   webhook -template -hooks hooks.json.tmpl
   ```

2. **检查模板语法:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "/path/to/script.sh",
     "trigger-rule": {
       "match": {
         "type": "value",
         "value": "{{.Env.SECRET_TOKEN}}"
       }
     }
   }
   ```

3. **验证环境变量:**
   ```bash
   # 检查变量是否设置
   echo $SECRET_TOKEN
   
   # 如果需要，导出
   export SECRET_TOKEN="your-secret"
   ```

### 热重载不工作

**症状:**
- 配置更改未应用
- 需要重启 webhook

**解决方案:**

1. **启用热重载:**
   ```bash
   webhook -hotreload -hooks hooks.json
   ```

2. **检查文件监视:**
   - 确保文件系统支持 inotify (Linux)
   - 验证文件权限允许读取
   - 检查日志中的文件系统事件

3. **手动重载:**
   ```bash
   # 发送 USR1 信号
   kill -USR1 $(pgrep webhook)
   
   # 或 HUP 信号
   kill -HUP $(pgrep webhook)
   ```

---

## 执行问题

### Hook 超时

**症状:**
- Hook 执行超时
- "context deadline exceeded" 错误

**解决方案:**

1. **增加超时:**
   ```bash
   webhook -hook-timeout-seconds=60
   ```

2. **使用流式输出:**
   ```json
   {
     "id": "long-running",
     "execute-command": "/path/to/long-script.sh",
     "stream-command-output": true
   }
   ```

3. **优化 hook 脚本:**
   - 审查脚本的低效之处
   - 考虑分解为更小的步骤
   - 如果适当，使用后台处理

### 达到并发 Hook 限制

**症状:**
- 请求排队
- "execution slot timeout" 错误

**解决方案:**

1. **增加并发:**
   ```bash
   webhook -max-concurrent-hooks=20
   ```

2. **增加槽位超时:**
   ```bash
   webhook -hook-execution-timeout=10
   ```

3. **优化 hook 执行:**
   - 减少执行时间
   - 使用异步处理
   - 审查资源使用

### 命令输出未捕获

**症状:**
- Hook 执行但响应中没有输出
- 日志中缺少输出

**解决方案:**

1. **启用输出捕获:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "/path/to/script.sh",
     "include-command-output-in-response": true
   }
   ```

2. **检查脚本输出:**
   ```bash
   # 手动测试脚本
   /path/to/script.sh
   
   # 确保脚本写入 stdout/stderr
   ```

3. **使用流式:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "/path/to/script.sh",
     "stream-command-output": true
   }
   ```

---

## 网络问题

### 连接被拒绝

**症状:**
- 无法连接到 webhook 服务器
- "Connection refused" 错误

**解决方案:**

1. **检查服务器是否运行:**
   ```bash
   # 检查进程
   ps aux | grep webhook
   
   # 检查端口
   netstat -tlnp | grep 9000
   # 或
   ss -tlnp | grep 9000
   ```

2. **验证绑定地址:**
   ```bash
   # 检查配置
   webhook -ip 0.0.0.0 -port 9000
   
   # 测试本地连接
   curl http://localhost:9000/health
   ```

3. **检查防火墙:**
   ```bash
   # 检查 iptables
   iptables -L -n | grep 9000
   
   # 检查 firewalld
   firewall-cmd --list-ports
   ```

### 速率限制问题

**症状:**
- 合法请求被拒绝
- 429 Too Many Requests 错误

**解决方案:**

1. **调整速率限制:**
   ```bash
   webhook \
     -rate-limit-enabled \
     -rate-limit-rps=100 \
     -rate-limit-burst=20
   ```

2. **如果不需要则禁用:**
   ```bash
   # 不要设置 -rate-limit-enabled
   webhook -hooks hooks.json
   ```

3. **监控拒绝率:**
   - 检查指标端点
   - 查看日志中的速率限制命中
   - 根据实际流量调整

### SSL/TLS 问题（反向代理）

**症状:**
- HTTPS 不工作
- 证书错误

**解决方案:**

1. **验证反向代理配置:**
   ```nginx
   # nginx 示例
   server {
       listen 443 ssl;
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
       
       location / {
           proxy_pass http://localhost:9000;
       }
   }
   ```

2. **检查证书有效性:**
   ```bash
   openssl x509 -in cert.pem -text -noout
   ```

3. **测试 HTTPS 连接:**
   ```bash
   curl -v https://your-domain.com/health
   ```

---

## 性能问题

### 高内存使用

**症状:**
- 内存消耗增长
- 内存不足错误

**解决方案:**

1. **减少请求体大小:**
   ```bash
   webhook -max-request-body-size=5242880  # 5MB
   ```

2. **减少并发:**
   ```bash
   webhook -max-concurrent-hooks=5
   ```

3. **检查内存泄漏:**
   - 监控内存随时间变化
   - 审查 hook 脚本
   - 使用内存分析工具

### 响应时间慢

**症状:**
- 高延迟
- Hook 执行慢

**解决方案:**

1. **优化 hook 脚本:**
   - 审查低效之处
   - 使用更快的命令
   - 并行化操作

2. **检查系统资源:**
   ```bash
   # 监控 CPU
   top
   
   # 监控 I/O
   iostat
   
   # 监控网络
   iftop
   ```

3. **审查超时设置:**
   ```bash
   webhook \
     -read-timeout-seconds=5 \
     -write-timeout-seconds=10
   ```

### 高 CPU 使用

**症状:**
- CPU 使用率接近 100%
- 系统变慢

**解决方案:**

1. **减少并发:**
   ```bash
   webhook -max-concurrent-hooks=5
   ```

2. **优化 hook 脚本:**
   - 减少 CPU 密集型操作
   - 使用高效算法
   - 考虑异步处理

3. **检查失控进程:**
   ```bash
   # 查找高 CPU 进程
   ps aux --sort=-%cpu | head
   ```

---

## 调试技术

### 启用详细日志

```bash
webhook -hooks hooks.json -verbose
```

**显示内容:**
- Hook 加载信息
- 请求详情
- 规则评估结果
- 命令执行详情

### 启用调试模式

```bash
webhook -hooks hooks.json -debug
```

**显示内容:**
- 详细的请求/响应转储
- 完整负载内容
- 所有中间件处理
- 内部状态信息

**安全注意:** 调试模式可能记录敏感信息。仅在开发中使用。

### 记录到文件

```bash
webhook -hooks hooks.json -logfile=/var/log/webhook.log -verbose
```

**好处:**
- 持久化日志
- 更容易分析
- 可以轮换

### 使用请求 ID

启用请求 ID 跟踪：

```bash
webhook -x-request-id -hooks hooks.json
```

**好处:**
- 通过日志追踪请求
- 将错误与请求关联
- 更好的调试

---

## 日志分析

### 常见日志模式

**成功执行:**
```
[request-id] hook-id got matched
[request-id] hook-id hook triggered successfully
[request-id] finished handling hook-id
```

**规则不匹配:**
```
[request-id] hook-id got matched, but didn't get triggered because the trigger rules were not satisfied
```

**执行错误:**
```
[request-id] error executing command for hook hook-id: exit status 1
```

**超时:**
```
[request-id] command execution timeout for hook hook-id
```

### 日志分析工具

1. **grep 查找错误:**
   ```bash
   grep -i error /var/log/webhook.log
   ```

2. **按 hook 计数:**
   ```bash
   grep "got matched" /var/log/webhook.log | awk '{print $2}' | sort | uniq -c
   ```

3. **查找慢请求:**
   ```bash
   grep "finished handling" /var/log/webhook.log | awk '{print $NF}' | sort -n
   ```

4. **实时监控:**
   ```bash
   tail -f /var/log/webhook.log | grep -i error
   ```

---

## 获取帮助

如果您仍然遇到问题：

1. **查看文档:**
   - [API 参考](API-Reference.md)
   - [配置参数](CLI-ENV.md)
   - [Hook 规则](Hook-Rules.md)

2. **审查日志:**
   - 启用详细/调试模式
   - 检查错误模式
   - 查找请求 ID

3. **测试配置:**
   ```bash
   webhook -validate-config -hooks hooks.json
   ```

4. **报告问题:**
   - 包含错误消息
   - 提供配置（已脱敏）
   - 包含相关日志
   - 描述重现步骤

---

## 快速参考

### 常用命令

```bash
# 使用详细日志启动
webhook -hooks hooks.json -verbose

# 使用调试模式启动
webhook -hooks hooks.json -debug

# 验证配置
webhook -validate-config -hooks hooks.json

# 检查版本
webhook -version

# 重载配置
kill -USR1 $(pgrep webhook)
```

### 健康检查

```bash
# 检查服务器是否运行
curl http://localhost:9000/health

# 检查指标
curl http://localhost:9000/metrics
```

### 有用的日志位置

- 默认：stdout/stderr
- 使用 `-logfile`：指定的文件路径
- systemd：`journalctl -u webhook`
- Docker：`docker logs container-name`

