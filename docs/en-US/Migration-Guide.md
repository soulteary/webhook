# Migration Guide

This guide helps you migrate from previous versions of Webhook or from the original webhook project to this fork.

## Table of Contents

1. [Migrating from Original Webhook](#migrating-from-original-webhook)
2. [Upgrading Between Versions](#upgrading-between-versions)
3. [Breaking Changes](#breaking-changes)
4. [Configuration Migration](#configuration-migration)
5. [Feature Additions](#feature-additions)
6. [Migration Checklist](#migration-checklist)

---

## Migrating from Original Webhook

This fork (soulteary/webhook) is based on the original webhook project (adnanh/webhook) with significant improvements in security, performance, and features.

### Key Differences

1. **Security Enhancements:**
   - Command path whitelisting
   - Argument validation and limits
   - Strict mode for command injection prevention
   - Improved logging with sanitization

2. **Performance Improvements:**
   - Better concurrency control
   - Configurable timeouts
   - Rate limiting support
   - Optimized request handling

3. **New Features:**
   - Prometheus metrics endpoint
   - Health check endpoint
   - Enhanced error handling
   - Better logging and debugging

4. **Configuration Compatibility:**
   - Fully compatible with original hook configurations
   - Additional security and performance parameters
   - Backward compatible with existing setups

### Migration Steps

1. **Backup Current Configuration:**
   ```bash
   cp hooks.json hooks.json.backup
   cp /path/to/webhook /path/to/webhook.backup
   ```

2. **Download New Version:**
   ```bash
   # Download from releases
   wget https://github.com/soulteary/webhook/releases/latest/download/webhook-linux-amd64
   
   # Or use Docker
   docker pull soulteary/webhook:latest
   ```

3. **Test Configuration:**
   ```bash
   # Validate configuration
   ./webhook -validate-config -hooks hooks.json
   ```

4. **Gradual Migration:**
   - Start with a test environment
   - Run both versions in parallel if possible
   - Monitor for issues
   - Gradually migrate production

5. **Update Configuration (Optional):**
   - Add security parameters (recommended)
   - Configure rate limiting
   - Set up monitoring endpoints

### Recommended Security Updates

After migration, enhance security:

```bash
# Add command path whitelist
webhook \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -hooks hooks.json

# Or via environment variables
export ALLOWED_COMMAND_PATHS="/usr/bin,/opt/scripts"
export STRICT_MODE=true
webhook -hooks hooks.json
```

---

## Upgrading Between Versions

### General Upgrade Process

1. **Review Release Notes:**
   - Check for breaking changes
   - Review new features
   - Note deprecated features

2. **Backup:**
   ```bash
   # Backup configuration
   cp hooks.json hooks.json.backup
   
   # Backup binary
   cp /usr/local/bin/webhook /usr/local/bin/webhook.backup
   ```

3. **Test in Staging:**
   - Deploy new version to staging
   - Run full test suite
   - Verify all hooks work correctly

4. **Update:**
   ```bash
   # Download new version
   wget https://github.com/soulteary/webhook/releases/download/vX.X.X/webhook-linux-amd64
   
   # Replace binary
   sudo mv webhook-linux-amd64 /usr/local/bin/webhook
   sudo chmod +x /usr/local/bin/webhook
   ```

5. **Restart Service:**
   ```bash
   # systemd
   sudo systemctl restart webhook
   
   # Docker
   docker restart webhook-container
   ```

6. **Verify:**
   ```bash
   # Check version
   webhook -version
   
   # Check health
   curl http://localhost:9000/health
   
   # Test a hook
   curl http://localhost:9000/hooks/test-hook
   ```

### Version-Specific Upgrades

#### Upgrading to 3.6.x

**New Features:**
- Prometheus metrics endpoint
- Health check endpoint
- Enhanced error responses
- Improved logging

**Configuration Changes:**
- No breaking changes
- Optional: Add security parameters
- Optional: Configure rate limiting

**Migration:**
```bash
# No configuration changes required
# Optional: Add security settings
webhook \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -hooks hooks.json
```

#### Upgrading to 3.5.x

**New Features:**
- Rate limiting support
- Enhanced timeout configuration
- Improved concurrency control

**Configuration Changes:**
- New optional parameters for rate limiting
- New timeout parameters

**Migration:**
```bash
# Existing configuration works as-is
# Optional: Enable rate limiting
webhook \
  -rate-limit-enabled \
  -rate-limit-rps=100 \
  -rate-limit-burst=20 \
  -hooks hooks.json
```

---

## Breaking Changes

### Configuration File Format

**Status:** No breaking changes in hook configuration format.

Hook configurations remain fully compatible. All existing configurations will work without modification.

### Command-Line Arguments

**Status:** Backward compatible with additions.

All original command-line arguments are supported. New arguments are optional and have sensible defaults.

### API Endpoints

**Status:** Fully backward compatible.

All original endpoints work as before. New endpoints (`/health`, `/metrics`) are additions and don't affect existing functionality.

### Behavior Changes

1. **Error Responses:**
   - Error responses now include JSON format with request ID
   - Plain text format still supported for backward compatibility
   - More detailed error messages

2. **Logging:**
   - Enhanced logging format
   - Request ID tracking
   - Better error context

3. **Security:**
   - Stricter default behaviors (can be configured)
   - Enhanced validation
   - Better error handling

---

## Configuration Migration

### Adding Security Parameters

**Before:**
```bash
webhook -hooks hooks.json
```

**After (Recommended):**
```bash
webhook \
  -allowed-command-paths="/usr/bin,/opt/scripts" \
  -strict-mode \
  -max-arg-length=1048576 \
  -max-total-args-length=10485760 \
  -max-args-count=1000 \
  -hooks hooks.json
```

### Adding Performance Parameters

**Before:**
```bash
webhook -hooks hooks.json
```

**After (Optional):**
```bash
webhook \
  -max-concurrent-hooks=20 \
  -hook-timeout-seconds=60 \
  -rate-limit-enabled \
  -rate-limit-rps=100 \
  -hooks hooks.json
```

### Environment Variables

You can migrate from command-line arguments to environment variables:

**Before:**
```bash
webhook -hooks hooks.json -verbose -port 9000
```

**After:**
```bash
export HOOKS=/path/to/hooks.json
export VERBOSE=true
export PORT=9000
webhook
```

### Docker Migration

**Before (Original):**
```bash
docker run -d \
  -p 9000:9000 \
  -v /path/to/hooks.json:/etc/webhook/hooks.json \
  adnanh/webhook:latest
```

**After (This Fork):**
```bash
docker run -d \
  -p 9000:9000 \
  -v /path/to/hooks.json:/etc/webhook/hooks.json \
  -e ALLOWED_COMMAND_PATHS="/usr/bin,/opt/scripts" \
  -e STRICT_MODE=true \
  soulteary/webhook:latest
```

---

## Feature Additions

### New Endpoints

1. **Health Check:**
   ```bash
   curl http://localhost:9000/health
   # Response: {"status":"ok"}
   ```

2. **Metrics:**
   ```bash
   curl http://localhost:9000/metrics
   # Prometheus format metrics
   ```

### New Configuration Options

1. **Security:**
   - `--allowed-command-paths`
   - `--strict-mode`
   - `--max-arg-length`
   - `--max-total-args-length`
   - `--max-args-count`

2. **Performance:**
   - `--rate-limit-enabled`
   - `--rate-limit-rps`
   - `--rate-limit-burst`
   - `--max-concurrent-hooks`
   - `--hook-timeout-seconds`
   - `--hook-execution-timeout`

3. **HTTP Server:**
   - `--read-header-timeout-seconds`
   - `--read-timeout-seconds`
   - `--write-timeout-seconds`
   - `--idle-timeout-seconds`
   - `--max-header-bytes`

### Enhanced Features

1. **Error Handling:**
   - JSON error responses
   - Request ID tracking
   - Better error context

2. **Logging:**
   - Structured logging
   - Request ID correlation
   - Enhanced debug output

3. **Monitoring:**
   - Prometheus metrics
   - Health check endpoint
   - System metrics

### Observability and Compliance

This fork also supports:

- **Distributed tracing (OpenTelemetry/OTLP):** Export traces to any OTLP-compatible backend. Use `-tracing-enabled`, `-otlp-endpoint`, and `-tracing-service-name`.
- **Audit logging:** Record hook executions and requests to file, Redis, or database. Use `-audit-enabled`, `-audit-storage-type`, `-audit-file-path`, and related options.
- **Redis-based distributed rate limiting:** Share rate-limit state across multiple webhook instances. Use `-redis-enabled`, `-redis-addr`, `-rate-limit-window`, and related options.

For full parameter lists and environment variables, see [Webhook Parameters](Webhook-Parameters.md).

---

## Migration Checklist

### Pre-Migration

- [ ] Review release notes and changelog
- [ ] Backup current configuration
- [ ] Backup current binary/container
- [ ] Test in staging environment
- [ ] Document current setup

### Migration Steps

- [ ] Download new version
- [ ] Validate configuration
- [ ] Update configuration (if needed)
- [ ] Deploy to staging
- [ ] Run test suite
- [ ] Monitor for issues
- [ ] Deploy to production
- [ ] Verify functionality

### Post-Migration

- [ ] Verify all hooks work
- [ ] Check monitoring endpoints
- [ ] Review logs for errors
- [ ] Update documentation
- [ ] Train team on new features
- [ ] Set up alerts (if using metrics)

### Security Enhancements (Recommended)

- [ ] Add command path whitelist
- [ ] Enable strict mode
- [ ] Set argument limits
- [ ] Configure rate limiting
- [ ] Review and update trigger rules
- [ ] Set up monitoring

### Performance Tuning (Optional)

- [ ] Configure concurrency limits
- [ ] Set appropriate timeouts
- [ ] Enable rate limiting
- [ ] Set up metrics collection
- [ ] Configure alerting

---

## Rollback Procedure

If you need to rollback:

1. **Stop Current Service:**
   ```bash
   sudo systemctl stop webhook
   # or
   docker stop webhook-container
   ```

2. **Restore Backup:**
   ```bash
   # Restore binary
   sudo cp /usr/local/bin/webhook.backup /usr/local/bin/webhook
   
   # Restore configuration (if changed)
   cp hooks.json.backup hooks.json
   ```

3. **Restart Service:**
   ```bash
   sudo systemctl start webhook
   # or
   docker start webhook-container
   ```

4. **Verify:**
   ```bash
   curl http://localhost:9000/health
   ```

---

## Getting Help

If you encounter issues during migration:

1. **Check Documentation:**
   - [Troubleshooting Guide](Troubleshooting.md)
   - [API Reference](API-Reference.md)
   - [Configuration Parameters](Webhook-Parameters.md)

2. **Review Logs:**
   ```bash
   # Enable verbose logging
   webhook -hooks hooks.json -verbose
   
   # Check for errors
   grep -i error /var/log/webhook.log
   ```

3. **Test Configuration:**
   ```bash
   webhook -validate-config -hooks hooks.json
   ```

4. **Report Issues:**
   - Include version information
   - Provide configuration (sanitized)
   - Include relevant logs
   - Describe migration steps taken

---

## Additional Resources

- [Security Best Practices](Security-Best-Practices.md) - Security recommendations
- [Performance Tuning](Performance-Tuning.md) - Performance optimization
- [Troubleshooting Guide](Troubleshooting.md) - Common issues and solutions
- [API Reference](API-Reference.md) - API documentation

