# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Webhook.

## Table of Contents

1. [Common Issues](#common-issues)
2. [Configuration Problems](#configuration-problems)
3. [Execution Problems](#execution-problems)
4. [Network Issues](#network-issues)
5. [Performance Issues](#performance-issues)
6. [Debugging Techniques](#debugging-techniques)
7. [Log Analysis](#log-analysis)

---

## Common Issues

### Hook Not Found (404)

**Symptoms:**
- HTTP 404 response when triggering a hook
- Error message: "Hook not found."

**Possible Causes:**
1. Hook ID mismatch between URL and configuration
2. Hook configuration file not loaded
3. Hook ID contains invalid characters

**Solutions:**

1. **Verify hook ID:**
   ```bash
   # Check the hook ID in your configuration
   cat hooks.json | jq '.[] | .id'
   
   # Ensure URL matches exactly (case-sensitive)
   curl http://localhost:9000/hooks/your-hook-id
   ```

2. **Check configuration loading:**
   ```bash
   # Run with verbose mode to see loaded hooks
   webhook -hooks hooks.json -verbose
   
   # Look for "loaded X hooks" message
   ```

3. **Validate hook ID format:**
   - Hook IDs are case-sensitive
   - Avoid special characters
   - Use alphanumeric characters and hyphens

### Hook Not Triggering

**Symptoms:**
- Hook exists but doesn't execute
- Returns "Hook rules were not satisfied"

**Possible Causes:**
1. Trigger rules not met
2. HTTP method not allowed
3. Missing required parameters

**Solutions:**

1. **Check trigger rules:**
   ```bash
   # Enable debug mode to see rule evaluation
   webhook -hooks hooks.json -debug
   
   # Review logs for rule evaluation results
   ```

2. **Verify HTTP method:**
   ```json
   {
     "id": "my-hook",
     "http-methods": ["POST"],
     "execute-command": "/path/to/script.sh"
   }
   ```

3. **Check required parameters:**
   - Review trigger rule requirements
   - Verify request includes all required data
   - Check parameter sources (header, url, payload)

### Command Execution Fails

**Symptoms:**
- Hook triggers but command fails
- HTTP 500 error
- Command not found errors

**Possible Causes:**
1. Command path not in whitelist
2. Insufficient permissions
3. Command doesn't exist
4. Working directory issues

**Solutions:**

1. **Check command path whitelist:**
   ```bash
   # If using -allowed-command-paths, verify command is included
   webhook -allowed-command-paths="/usr/bin,/opt/scripts" -hooks hooks.json
   
   # Check if command path matches whitelist
   ```

2. **Verify file permissions:**
   ```bash
   # Check script permissions
   ls -l /path/to/script.sh
   
   # Make executable if needed
   chmod +x /path/to/script.sh
   
   # Note: Don't use -allow-auto-chmod in production
   ```

3. **Verify command exists:**
   ```bash
   # Test command manually
   /path/to/script.sh
   
   # Check if command is in PATH
   which command-name
   ```

4. **Check working directory:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "script.sh",
     "command-working-directory": "/opt/scripts"
   }
   ```

### Permission Denied Errors

**Symptoms:**
- "permission denied" errors in logs
- Commands fail to execute

**Solutions:**

1. **Set correct permissions:**
   ```bash
   # Make script executable
   chmod +x /path/to/script.sh
   
   # Set ownership
   chown webhook:webhook /path/to/script.sh
   ```

2. **Check user permissions:**
   ```bash
   # Verify webhook user can access files
   sudo -u webhook ls -l /path/to/script.sh
   sudo -u webhook /path/to/script.sh
   ```

3. **Use setuid/setgid:**
   ```bash
   # Run with specific user/group
   webhook -setuid $(id -u webhook) -setgid $(id -g webhook)
   ```

---

## Configuration Problems

### Configuration File Not Loading

**Symptoms:**
- Webhook starts but no hooks available
- Configuration errors in logs

**Solutions:**

1. **Validate JSON/YAML syntax:**
   ```bash
   # Validate JSON
   jq . hooks.json
   
   # Validate YAML
   yamllint hooks.yaml
   ```

2. **Check file path:**
   ```bash
   # Use absolute path
   webhook -hooks /absolute/path/to/hooks.json
   
   # Verify file exists
   ls -l /path/to/hooks.json
   ```

3. **Check file permissions:**
   ```bash
   # Ensure file is readable
   chmod 644 hooks.json
   ```

### Template Parsing Errors

**Symptoms:**
- Template syntax errors
- Environment variables not substituted

**Solutions:**

1. **Enable template mode:**
   ```bash
   webhook -template -hooks hooks.json.tmpl
   ```

2. **Check template syntax:**
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

3. **Verify environment variables:**
   ```bash
   # Check if variable is set
   echo $SECRET_TOKEN
   
   # Export if needed
   export SECRET_TOKEN="your-secret"
   ```

### Hot Reload Not Working

**Symptoms:**
- Configuration changes not applied
- Need to restart webhook

**Solutions:**

1. **Enable hot reload:**
   ```bash
   webhook -hotreload -hooks hooks.json
   ```

2. **Check file watching:**
   - Ensure file system supports inotify (Linux)
   - Verify file permissions allow reading
   - Check for file system events in logs

3. **Manual reload:**
   ```bash
   # Send USR1 signal
   kill -USR1 $(pgrep webhook)
   
   # Or HUP signal
   kill -HUP $(pgrep webhook)
   ```

---

## Execution Problems

### Hook Timeout

**Symptoms:**
- Hook execution times out
- "context deadline exceeded" errors

**Solutions:**

1. **Increase timeout:**
   ```bash
   webhook -hook-timeout-seconds=60
   ```

2. **Use streaming output:**
   ```json
   {
     "id": "long-running",
     "execute-command": "/path/to/long-script.sh",
     "stream-command-output": true
   }
   ```

3. **Optimize hook script:**
   - Review script for inefficiencies
   - Consider breaking into smaller steps
   - Use background processing if appropriate

### Concurrent Hook Limit Reached

**Symptoms:**
- Requests queued
- "execution slot timeout" errors

**Solutions:**

1. **Increase concurrency:**
   ```bash
   webhook -max-concurrent-hooks=20
   ```

2. **Increase slot timeout:**
   ```bash
   webhook -hook-execution-timeout=10
   ```

3. **Optimize hook execution:**
   - Reduce execution time
   - Use async processing
   - Review resource usage

### Command Output Not Captured

**Symptoms:**
- Hook executes but no output in response
- Output missing from logs

**Solutions:**

1. **Enable output capture:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "/path/to/script.sh",
     "include-command-output-in-response": true
   }
   ```

2. **Check script output:**
   ```bash
   # Test script manually
   /path/to/script.sh
   
   # Ensure script writes to stdout/stderr
   ```

3. **Use streaming:**
   ```json
   {
     "id": "my-hook",
     "execute-command": "/path/to/script.sh",
     "stream-command-output": true
   }
   ```

---

## Network Issues

### Connection Refused

**Symptoms:**
- Cannot connect to webhook server
- "Connection refused" errors

**Solutions:**

1. **Check if server is running:**
   ```bash
   # Check process
   ps aux | grep webhook
   
   # Check port
   netstat -tlnp | grep 9000
   # or
   ss -tlnp | grep 9000
   ```

2. **Verify bind address:**
   ```bash
   # Check configuration
   webhook -ip 0.0.0.0 -port 9000
   
   # Test local connection
   curl http://localhost:9000/health
   ```

3. **Check firewall:**
   ```bash
   # Check iptables
   iptables -L -n | grep 9000
   
   # Check firewalld
   firewall-cmd --list-ports
   ```

### Rate Limiting Issues

**Symptoms:**
- Legitimate requests rejected
- 429 Too Many Requests errors

**Solutions:**

1. **Adjust rate limits:**
   ```bash
   webhook \
     -rate-limit-enabled \
     -rate-limit-rps=100 \
     -rate-limit-burst=20
   ```

2. **Disable if not needed:**
   ```bash
   # Don't set -rate-limit-enabled
   webhook -hooks hooks.json
   ```

3. **Monitor rejection rate:**
   - Check metrics endpoint
   - Review logs for rate limit hits
   - Adjust based on actual traffic

### SSL/TLS Issues (Reverse Proxy)

**Symptoms:**
- HTTPS not working
- Certificate errors

**Solutions:**

1. **Verify reverse proxy configuration:**
   ```nginx
   # nginx example
   server {
       listen 443 ssl;
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
       
       location / {
           proxy_pass http://localhost:9000;
       }
   }
   ```

2. **Check certificate validity:**
   ```bash
   openssl x509 -in cert.pem -text -noout
   ```

3. **Test HTTPS connection:**
   ```bash
   curl -v https://your-domain.com/health
   ```

---

## Performance Issues

### High Memory Usage

**Symptoms:**
- Memory consumption growing
- Out of memory errors

**Solutions:**

1. **Reduce request body size:**
   ```bash
   webhook -max-request-body-size=5242880  # 5MB
   ```

2. **Reduce concurrency:**
   ```bash
   webhook -max-concurrent-hooks=5
   ```

3. **Check for memory leaks:**
   - Monitor memory over time
   - Review hook scripts
   - Use memory profiling tools

### Slow Response Times

**Symptoms:**
- High latency
- Slow hook execution

**Solutions:**

1. **Optimize hook scripts:**
   - Review for inefficiencies
   - Use faster commands
   - Parallelize operations

2. **Check system resources:**
   ```bash
   # Monitor CPU
   top
   
   # Monitor I/O
   iostat
   
   # Monitor network
   iftop
   ```

3. **Review timeout settings:**
   ```bash
   webhook \
     -read-timeout-seconds=5 \
     -write-timeout-seconds=10
   ```

### High CPU Usage

**Symptoms:**
- CPU usage near 100%
- System slowdown

**Solutions:**

1. **Reduce concurrency:**
   ```bash
   webhook -max-concurrent-hooks=5
   ```

2. **Optimize hook scripts:**
   - Reduce CPU-intensive operations
   - Use efficient algorithms
   - Consider async processing

3. **Check for runaway processes:**
   ```bash
   # Find high CPU processes
   ps aux --sort=-%cpu | head
   ```

---

## Debugging Techniques

### Enable Verbose Logging

```bash
webhook -hooks hooks.json -verbose
```

**What it shows:**
- Hook loading information
- Request details
- Rule evaluation results
- Command execution details

### Enable Debug Mode

```bash
webhook -hooks hooks.json -debug
```

**What it shows:**
- Detailed request/response dumps
- Full payload contents
- All middleware processing
- Internal state information

**Security Note:** Debug mode may log sensitive information. Use only in development.

### Log to File

```bash
webhook -hooks hooks.json -logfile=/var/log/webhook.log -verbose
```

**Benefits:**
- Persistent logs
- Easier analysis
- Can be rotated

### Use Request ID

Enable request ID tracking:

```bash
webhook -x-request-id -hooks hooks.json
```

**Benefits:**
- Trace requests through logs
- Correlate errors with requests
- Better debugging

---

## Log Analysis

### Common Log Patterns

**Successful execution:**
```
[request-id] hook-id got matched
[request-id] hook-id hook triggered successfully
[request-id] finished handling hook-id
```

**Rule mismatch:**
```
[request-id] hook-id got matched, but didn't get triggered because the trigger rules were not satisfied
```

**Execution error:**
```
[request-id] error executing command for hook hook-id: exit status 1
```

**Timeout:**
```
[request-id] command execution timeout for hook hook-id
```

### Log Analysis Tools

1. **grep for errors:**
   ```bash
   grep -i error /var/log/webhook.log
   ```

2. **Count by hook:**
   ```bash
   grep "got matched" /var/log/webhook.log | awk '{print $2}' | sort | uniq -c
   ```

3. **Find slow requests:**
   ```bash
   grep "finished handling" /var/log/webhook.log | awk '{print $NF}' | sort -n
   ```

4. **Monitor in real-time:**
   ```bash
   tail -f /var/log/webhook.log | grep -i error
   ```

---

## Getting Help

If you're still experiencing issues:

1. **Check documentation:**
   - [API Reference](API-Reference.md)
   - [Configuration Parameters](Webhook-Parameters.md)
   - [Hook Rules](Hook-Rules.md)

2. **Review logs:**
   - Enable verbose/debug mode
   - Check for error patterns
   - Look for request IDs

3. **Test configuration:**
   ```bash
   webhook -validate-config -hooks hooks.json
   ```

4. **Report issues:**
   - Include error messages
   - Provide configuration (sanitized)
   - Include relevant logs
   - Describe steps to reproduce

---

## Quick Reference

### Common Commands

```bash
# Start with verbose logging
webhook -hooks hooks.json -verbose

# Start with debug mode
webhook -hooks hooks.json -debug

# Validate configuration
webhook -validate-config -hooks hooks.json

# Check version
webhook -version

# Reload configuration
kill -USR1 $(pgrep webhook)
```

### Health Check

```bash
# Check if server is running
curl http://localhost:9000/health

# Check metrics
curl http://localhost:9000/metrics
```

### Useful Log Locations

- Default: stdout/stderr
- With `-logfile`: specified file path
- systemd: `journalctl -u webhook`
- Docker: `docker logs container-name`

