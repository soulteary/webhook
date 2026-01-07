# Performance Tuning Guide

This guide provides recommendations for optimizing Webhook performance in various deployment scenarios.

## Table of Contents

1. [Concurrency Configuration](#concurrency-configuration)
2. [Timeout Configuration](#timeout-configuration)
3. [Memory Optimization](#memory-optimization)
4. [Network Optimization](#network-optimization)
5. [Monitoring and Metrics](#monitoring-and-metrics)
6. [System-Level Optimizations](#system-level-optimizations)
7. [Performance Testing](#performance-testing)

---

## Concurrency Configuration

### Max Concurrent Hooks

Control the maximum number of hooks that can execute simultaneously.

```bash
webhook -max-concurrent-hooks=20
```

**Default:** 10

**Recommendations:**
- **Low traffic (< 10 req/sec):** 5-10
- **Medium traffic (10-50 req/sec):** 10-20
- **High traffic (50+ req/sec):** 20-50
- **CPU-bound hooks:** Lower values (5-10)
- **I/O-bound hooks:** Higher values (20-50)

**Considerations:**
- Each concurrent hook consumes memory and CPU
- Too high values can lead to resource exhaustion
- Too low values can cause request queuing
- Monitor system resources (CPU, memory) to find optimal value

### Hook Execution Timeout

Set the maximum time a hook can execute before being terminated.

```bash
webhook -hook-timeout-seconds=60
```

**Default:** 30 seconds

**Recommendations:**
- **Quick operations (< 5s):** 10-30 seconds
- **Medium operations (5-30s):** 30-60 seconds
- **Long operations (30s+):** 60-300 seconds

**Considerations:**
- Prevents runaway processes from consuming resources
- Should be set based on your longest-running hook
- Consider using streaming output for long-running hooks

### Hook Execution Slot Timeout

Maximum time to wait for an execution slot when max concurrent hooks is reached.

```bash
webhook -hook-execution-timeout=10
```

**Default:** 5 seconds

**Recommendations:**
- **Low latency requirements:** 2-5 seconds
- **Normal operations:** 5-10 seconds
- **High load scenarios:** 10-30 seconds

---

## Timeout Configuration

### HTTP Server Timeouts

Configure timeouts for different phases of HTTP request handling.

```bash
webhook \
  -read-header-timeout-seconds=5 \
  -read-timeout-seconds=10 \
  -write-timeout-seconds=30 \
  -idle-timeout-seconds=90
```

**Defaults:**
- `read-header-timeout-seconds`: 5 seconds
- `read-timeout-seconds`: 10 seconds
- `write-timeout-seconds`: 30 seconds
- `idle-timeout-seconds`: 90 seconds

**Recommendations:**

**Read Header Timeout:**
- **Fast networks:** 3-5 seconds
- **Slow networks:** 5-10 seconds
- **High latency:** 10-15 seconds

**Read Timeout:**
- **Small payloads (< 1MB):** 5-10 seconds
- **Medium payloads (1-10MB):** 10-30 seconds
- **Large payloads (10MB+):** 30-60 seconds

**Write Timeout:**
- **Quick responses:** 10-30 seconds
- **Normal responses:** 30-60 seconds
- **Streaming responses:** 60-300 seconds (or disable for streaming)

**Idle Timeout:**
- **High connection reuse:** 60-90 seconds
- **Low connection reuse:** 30-60 seconds
- **Keep-alive optimization:** 90-120 seconds

---

## Memory Optimization

### Request Body Size Limits

Limit the maximum size of request bodies to prevent memory exhaustion.

```bash
webhook -max-request-body-size=10485760  # 10MB
```

**Default:** 10MB

**Recommendations:**
- **Small payloads:** 1-5MB
- **Medium payloads:** 5-10MB
- **Large payloads:** 10-50MB (adjust based on available memory)

**Considerations:**
- Larger limits consume more memory per request
- Set based on your actual payload sizes
- Monitor memory usage to find optimal value

### Multipart Form Memory

Control memory usage for multipart form parsing.

```bash
webhook -max-multipart-mem=2097152  # 2MB
```

**Default:** 1MB

**Recommendations:**
- **Small files:** 1-2MB
- **Medium files:** 2-5MB
- **Large files:** 5-10MB (larger files are cached to disk)

**Considerations:**
- Data exceeding this limit is written to disk
- Higher values use more memory but reduce disk I/O
- Balance based on available memory and disk speed

### Header Size Limits

Limit the maximum size of HTTP headers.

```bash
webhook -max-header-bytes=1048576  # 1MB
```

**Default:** 1MB

**Recommendations:**
- **Normal headers:** 64KB-256KB
- **Extended headers:** 256KB-1MB
- **Very large headers:** 1MB-2MB

---

## Network Optimization

### Rate Limiting

Control request rate to prevent overload and ensure fair resource usage.

```bash
webhook \
  -rate-limit-enabled \
  -rate-limit-rps=100 \
  -rate-limit-burst=20
```

**Defaults:**
- `rate-limit-enabled`: false
- `rate-limit-rps`: 100 requests/second
- `rate-limit-burst`: 10 requests

**Recommendations:**

**Low Traffic:**
```bash
-rate-limit-rps=10 -rate-limit-burst=5
```

**Medium Traffic:**
```bash
-rate-limit-rps=50 -rate-limit-burst=10
```

**High Traffic:**
```bash
-rate-limit-rps=200 -rate-limit-burst=50
```

**Considerations:**
- Burst should be 10-20% of RPS for smooth operation
- Too low can cause legitimate requests to be rejected
- Too high can allow abuse
- Monitor rejection rates and adjust accordingly

### Connection Keep-Alive

Optimize connection reuse by adjusting idle timeout.

```bash
webhook -idle-timeout-seconds=120
```

**Benefits:**
- Reduces connection overhead
- Improves latency for repeated requests
- Reduces TCP handshake overhead

---

## Monitoring and Metrics

### Prometheus Metrics

Webhook exposes Prometheus metrics at `/metrics` endpoint.

**Key Metrics:**

1. **HTTP Request Metrics:**
   - `webhook_http_requests_total`: Total HTTP requests
   - `webhook_http_request_duration_seconds`: Request duration histogram

2. **Hook Execution Metrics:**
   - `webhook_hook_executions_total`: Total hook executions
   - `webhook_hook_execution_duration_seconds`: Execution duration histogram

3. **System Metrics:**
   - `webhook_system_memory_bytes`: Memory usage
   - `webhook_system_cpu_percent`: CPU usage percentage

**Monitoring Recommendations:**

1. **Set up alerts for:**
   - High error rates (> 5%)
   - Slow response times (p95 > 1s)
   - High memory usage (> 80%)
   - High CPU usage (> 80%)
   - Hook execution failures

2. **Track trends:**
   - Request rate over time
   - Average response time
   - Concurrent hook executions
   - Resource utilization

3. **Performance dashboards:**
   - Request rate and latency
   - Hook execution success/failure rates
   - System resource usage
   - Queue depth (when max concurrent hooks reached)

---

## System-Level Optimizations

### Operating System Tuning

**File Descriptor Limits:**

```bash
# Increase file descriptor limit
ulimit -n 65536

# Or in /etc/security/limits.conf
webhook soft nofile 65536
webhook hard nofile 65536
```

**TCP Tuning (Linux):**

```bash
# Increase TCP connection queue
echo 'net.core.somaxconn = 1024' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 2048' >> /etc/sysctl.conf

# Enable TCP fast open
echo 'net.ipv4.tcp_fastopen = 3' >> /etc/sysctl.conf

# Apply changes
sysctl -p
```

**Memory Management:**

```bash
# Disable swap for better performance (if sufficient RAM)
swapoff -a

# Or set swappiness to 0
echo 'vm.swappiness = 0' >> /etc/sysctl.conf
sysctl -p
```

### Process Priority

Run webhook with appropriate priority:

```bash
# Set nice value (lower = higher priority)
nice -n -10 webhook -hooks hooks.json
```

### Resource Limits (systemd)

If using systemd, configure resource limits:

```ini
[Service]
LimitNOFILE=65536
LimitNPROC=4096
MemoryLimit=2G
CPUQuota=200%
```

---

## Performance Testing

### Load Testing

Use tools like `ab`, `wrk`, or `hey` to test performance:

```bash
# Apache Bench
ab -n 10000 -c 100 http://localhost:9000/hooks/test-hook

# wrk
wrk -t12 -c400 -d30s http://localhost:9000/hooks/test-hook

# hey
hey -n 10000 -c 100 http://localhost:9000/hooks/test-hook
```

### Benchmarking Scenarios

1. **Baseline Test:**
   - Single hook, no concurrency
   - Measure: latency, throughput

2. **Concurrency Test:**
   - Multiple concurrent requests
   - Measure: response time under load, resource usage

3. **Sustained Load Test:**
   - Continuous load for extended period
   - Measure: memory leaks, resource stability

4. **Stress Test:**
   - Load beyond normal capacity
   - Measure: degradation, failure points

### Performance Targets

**Response Time:**
- p50 (median): < 100ms
- p95: < 500ms
- p99: < 1s

**Throughput:**
- Small payloads: 1000+ req/sec
- Medium payloads: 100+ req/sec
- Large payloads: 10+ req/sec

**Resource Usage:**
- CPU: < 70% average
- Memory: < 80% of allocated
- No memory leaks over 24 hours

---

## Performance Checklist

Before deploying to production:

- [ ] Configure appropriate `max-concurrent-hooks` based on workload
- [ ] Set `hook-timeout-seconds` based on longest-running hook
- [ ] Configure HTTP timeouts appropriately
- [ ] Set request body size limits
- [ ] Enable and configure rate limiting
- [ ] Set up monitoring and alerting
- [ ] Tune system-level parameters (file descriptors, TCP)
- [ ] Perform load testing
- [ ] Document performance baselines
- [ ] Set up performance dashboards

---

## Troubleshooting Performance Issues

### High Latency

1. **Check hook execution time:**
   - Review hook scripts for inefficiencies
   - Optimize long-running operations
   - Consider async processing for heavy tasks

2. **Check system resources:**
   - Monitor CPU and memory usage
   - Check for resource contention
   - Verify network latency

3. **Review timeout settings:**
   - Ensure timeouts are appropriate
   - Check for timeout-related errors

### High Memory Usage

1. **Review request body size limits:**
   - Reduce if too high
   - Monitor actual payload sizes

2. **Check for memory leaks:**
   - Monitor memory over time
   - Review hook scripts for leaks

3. **Adjust concurrency:**
   - Reduce `max-concurrent-hooks` if needed
   - Monitor memory per concurrent hook

### Low Throughput

1. **Check rate limiting:**
   - Ensure limits aren't too restrictive
   - Monitor rejection rates

2. **Review concurrency settings:**
   - Increase `max-concurrent-hooks` if CPU/memory allow
   - Check for bottlenecks in hook execution

3. **Optimize hooks:**
   - Review hook scripts for performance
   - Consider parallelizing operations

---

## Additional Resources

- [API Reference](API-Reference.md) - API performance considerations
- [Configuration Parameters](Webhook-Parameters.md) - All configuration options
- [Security Best Practices](Security-Best-Practices.md) - Security vs. performance trade-offs

