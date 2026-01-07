# 性能调优指南

本指南提供了在各种部署场景中优化 Webhook 性能的建议。

## 目录

1. [并发配置](#并发配置)
2. [超时配置](#超时配置)
3. [内存优化](#内存优化)
4. [网络优化](#网络优化)
5. [监控和指标](#监控和指标)
6. [系统级优化](#系统级优化)
7. [性能测试](#性能测试)

---

## 并发配置

### 最大并发 Hook 数

控制可以同时执行的最大 hook 数量。

```bash
webhook -max-concurrent-hooks=20
```

**默认值:** 10

**建议:**
- **低流量 (< 10 请求/秒):** 5-10
- **中等流量 (10-50 请求/秒):** 10-20
- **高流量 (50+ 请求/秒):** 20-50
- **CPU 密集型 hook:** 较低值 (5-10)
- **I/O 密集型 hook:** 较高值 (20-50)

**注意事项:**
- 每个并发 hook 消耗内存和 CPU
- 值过高可能导致资源耗尽
- 值过低可能导致请求排队
- 监控系统资源（CPU、内存）以找到最佳值

### Hook 执行超时

设置 hook 在被终止前可以执行的最大时间。

```bash
webhook -hook-timeout-seconds=60
```

**默认值:** 30 秒

**建议:**
- **快速操作 (< 5 秒):** 10-30 秒
- **中等操作 (5-30 秒):** 30-60 秒
- **长时间操作 (30 秒+):** 60-300 秒

**注意事项:**
- 防止失控进程消耗资源
- 应根据最长运行的 hook 设置
- 对于长时间运行的 hook，考虑使用流式输出

### Hook 执行槽位超时

当达到最大并发 hook 数时，等待执行槽位的最大时间。

```bash
webhook -hook-execution-timeout=10
```

**默认值:** 5 秒

**建议:**
- **低延迟要求:** 2-5 秒
- **正常操作:** 5-10 秒
- **高负载场景:** 10-30 秒

---

## 超时配置

### HTTP 服务器超时

为 HTTP 请求处理的不同阶段配置超时。

```bash
webhook \
  -read-header-timeout-seconds=5 \
  -read-timeout-seconds=10 \
  -write-timeout-seconds=30 \
  -idle-timeout-seconds=90
```

**默认值:**
- `read-header-timeout-seconds`: 5 秒
- `read-timeout-seconds`: 10 秒
- `write-timeout-seconds`: 30 秒
- `idle-timeout-seconds`: 90 秒

**建议:**

**读取请求头超时:**
- **快速网络:** 3-5 秒
- **慢速网络:** 5-10 秒
- **高延迟:** 10-15 秒

**读取超时:**
- **小负载 (< 1MB):** 5-10 秒
- **中等负载 (1-10MB):** 10-30 秒
- **大负载 (10MB+):** 30-60 秒

**写入超时:**
- **快速响应:** 10-30 秒
- **正常响应:** 30-60 秒
- **流式响应:** 60-300 秒（或对流式禁用）

**空闲超时:**
- **高连接复用:** 60-90 秒
- **低连接复用:** 30-60 秒
- **Keep-alive 优化:** 90-120 秒

---

## 内存优化

### 请求体大小限制

限制请求体的最大大小以防止内存耗尽。

```bash
webhook -max-request-body-size=10485760  # 10MB
```

**默认值:** 10MB

**建议:**
- **小负载:** 1-5MB
- **中等负载:** 5-10MB
- **大负载:** 10-50MB（根据可用内存调整）

**注意事项:**
- 更大的限制会消耗更多每个请求的内存
- 根据实际负载大小设置
- 监控内存使用以找到最佳值

### Multipart 表单内存

控制 multipart 表单解析的内存使用。

```bash
webhook -max-multipart-mem=2097152  # 2MB
```

**默认值:** 1MB

**建议:**
- **小文件:** 1-2MB
- **中等文件:** 2-5MB
- **大文件:** 5-10MB（更大的文件会缓存到磁盘）

**注意事项:**
- 超过此限制的数据会写入磁盘
- 更高的值使用更多内存但减少磁盘 I/O
- 根据可用内存和磁盘速度平衡

### 请求头大小限制

限制 HTTP 请求头的最大大小。

```bash
webhook -max-header-bytes=1048576  # 1MB
```

**默认值:** 1MB

**建议:**
- **正常请求头:** 64KB-256KB
- **扩展请求头:** 256KB-1MB
- **非常大的请求头:** 1MB-2MB

---

## 网络优化

### 速率限制

控制请求速率以防止过载并确保公平的资源使用。

```bash
webhook \
  -rate-limit-enabled \
  -rate-limit-rps=100 \
  -rate-limit-burst=20
```

**默认值:**
- `rate-limit-enabled`: false
- `rate-limit-rps`: 100 请求/秒
- `rate-limit-burst`: 10 请求

**建议:**

**低流量:**
```bash
-rate-limit-rps=10 -rate-limit-burst=5
```

**中等流量:**
```bash
-rate-limit-rps=50 -rate-limit-burst=10
```

**高流量:**
```bash
-rate-limit-rps=200 -rate-limit-burst=50
```

**注意事项:**
- 突发应该是 RPS 的 10-20% 以实现平稳运行
- 太低可能导致合法请求被拒绝
- 太高可能允许滥用
- 监控拒绝率并相应调整

### 连接 Keep-Alive

通过调整空闲超时来优化连接复用。

```bash
webhook -idle-timeout-seconds=120
```

**好处:**
- 减少连接开销
- 改善重复请求的延迟
- 减少 TCP 握手开销

---

## 监控和指标

### Prometheus 指标

Webhook 在 `/metrics` 端点公开 Prometheus 指标。

**关键指标:**

1. **HTTP 请求指标:**
   - `webhook_http_requests_total`: HTTP 请求总数
   - `webhook_http_request_duration_seconds`: 请求持续时间直方图

2. **Hook 执行指标:**
   - `webhook_hook_executions_total`: Hook 执行总数
   - `webhook_hook_execution_duration_seconds`: 执行持续时间直方图

3. **系统指标:**
   - `webhook_system_memory_bytes`: 内存使用量
   - `webhook_system_cpu_percent`: CPU 使用百分比

**监控建议:**

1. **设置告警:**
   - 高错误率 (> 5%)
   - 慢响应时间 (p95 > 1 秒)
   - 高内存使用 (> 80%)
   - 高 CPU 使用 (> 80%)
   - Hook 执行失败

2. **跟踪趋势:**
   - 请求速率随时间变化
   - 平均响应时间
   - 并发 hook 执行
   - 资源利用率

3. **性能仪表板:**
   - 请求速率和延迟
   - Hook 执行成功/失败率
   - 系统资源使用
   - 队列深度（当达到最大并发 hook 时）

---

## 系统级优化

### 操作系统调优

**文件描述符限制:**

```bash
# 增加文件描述符限制
ulimit -n 65536

# 或在 /etc/security/limits.conf 中
webhook soft nofile 65536
webhook hard nofile 65536
```

**TCP 调优 (Linux):**

```bash
# 增加 TCP 连接队列
echo 'net.core.somaxconn = 1024' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 2048' >> /etc/sysctl.conf

# 启用 TCP fast open
echo 'net.ipv4.tcp_fastopen = 3' >> /etc/sysctl.conf

# 应用更改
sysctl -p
```

**内存管理:**

```bash
# 禁用交换以提高性能（如果有足够的 RAM）
swapoff -a

# 或将 swappiness 设置为 0
echo 'vm.swappiness = 0' >> /etc/sysctl.conf
sysctl -p
```

### 进程优先级

以适当的优先级运行 webhook：

```bash
# 设置 nice 值（越低 = 优先级越高）
nice -n -10 webhook -hooks hooks.json
```

### 资源限制 (systemd)

如果使用 systemd，配置资源限制：

```ini
[Service]
LimitNOFILE=65536
LimitNPROC=4096
MemoryLimit=2G
CPUQuota=200%
```

---

## 性能测试

### 负载测试

使用 `ab`、`wrk` 或 `hey` 等工具测试性能：

```bash
# Apache Bench
ab -n 10000 -c 100 http://localhost:9000/hooks/test-hook

# wrk
wrk -t12 -c400 -d30s http://localhost:9000/hooks/test-hook

# hey
hey -n 10000 -c 100 http://localhost:9000/hooks/test-hook
```

### 基准测试场景

1. **基线测试:**
   - 单个 hook，无并发
   - 测量：延迟、吞吐量

2. **并发测试:**
   - 多个并发请求
   - 测量：负载下的响应时间、资源使用

3. **持续负载测试:**
   - 长时间连续负载
   - 测量：内存泄漏、资源稳定性

4. **压力测试:**
   - 超出正常容量的负载
   - 测量：性能下降、故障点

### 性能目标

**响应时间:**
- p50 (中位数): < 100ms
- p95: < 500ms
- p99: < 1s

**吞吐量:**
- 小负载: 1000+ 请求/秒
- 中等负载: 100+ 请求/秒
- 大负载: 10+ 请求/秒

**资源使用:**
- CPU: < 70% 平均值
- 内存: < 80% 已分配
- 24 小时内无内存泄漏

---

## 性能检查清单

在部署到生产环境之前：

- [ ] 根据工作负载配置适当的 `max-concurrent-hooks`
- [ ] 根据最长运行的 hook 设置 `hook-timeout-seconds`
- [ ] 适当配置 HTTP 超时
- [ ] 设置请求体大小限制
- [ ] 启用并配置速率限制
- [ ] 设置监控和告警
- [ ] 调整系统级参数（文件描述符、TCP）
- [ ] 执行负载测试
- [ ] 记录性能基线
- [ ] 设置性能仪表板

---

## 性能问题排查

### 高延迟

1. **检查 hook 执行时间:**
   - 审查 hook 脚本的低效之处
   - 优化长时间运行的操作
   - 考虑对重任务进行异步处理

2. **检查系统资源:**
   - 监控 CPU 和内存使用
   - 检查资源争用
   - 验证网络延迟

3. **审查超时设置:**
   - 确保超时设置适当
   - 检查与超时相关的错误

### 高内存使用

1. **审查请求体大小限制:**
   - 如果太高则降低
   - 监控实际负载大小

2. **检查内存泄漏:**
   - 监控内存随时间变化
   - 审查 hook 脚本是否有泄漏

3. **调整并发:**
   - 如果需要，减少 `max-concurrent-hooks`
   - 监控每个并发 hook 的内存

### 低吞吐量

1. **检查速率限制:**
   - 确保限制不会过于严格
   - 监控拒绝率

2. **审查并发设置:**
   - 如果 CPU/内存允许，增加 `max-concurrent-hooks`
   - 检查 hook 执行中的瓶颈

3. **优化 hook:**
   - 审查 hook 脚本的性能
   - 考虑并行化操作

---

## 其他资源

- [API 参考](API-Reference.md) - API 性能注意事项
- [配置参数](CLI-ENV.md) - 所有配置选项
- [安全最佳实践](Security-Best-Practices.md) - 安全与性能权衡

