# KSA CLI Usage Guide

## Overview

The `ksa` command-line tool is the primary interface for interacting with KubeStack AI. It provides AI-powered diagnostics for both Kubernetes resources and middleware infrastructure.

## Installation

### From Source

```bash
cd kubestack-ai
go build -o ksa ./cmd/ksa
sudo mv ksa /usr/local/bin/
```

### From Release

```bash
# Download latest release
curl -LO https://github.com/kubestack-ai/kubestack-ai/releases/latest/download/ksa-linux-amd64

# Make executable and install
chmod +x ksa-linux-amd64
sudo mv ksa-linux-amd64 /usr/local/bin/ksa
```

### Verify Installation

```bash
ksa version
# Output: KubeStack-AI version 0.1.0
```

## Basic Usage

### Getting Help

```bash
# General help
ksa --help

# Command-specific help
ksa diagnose --help
ksa plugin --help
```

### Global Flags

```bash
--config string      # Config file path (default: /etc/kubestack-ai/config.yaml or ~/.ksa.yaml)
--log-level string   # Log level: debug, info, warn, error, fatal (default: info)
-o, --output string  # Output format: text, json, yaml (default: text)
```

## Diagnosing Middleware

### Redis Diagnostics

**Basic diagnosis:**
```bash
ksa diagnose redis localhost:6379
```

**With authentication:**
```bash
ksa diagnose redis localhost:6379 --password your-password
```

**Specific categories:**
```bash
ksa diagnose redis localhost:6379 --categories memory,replication
```

**Deep analysis:**
```bash
ksa diagnose redis localhost:6379 --depth deep
```

**JSON output:**
```bash
ksa diagnose redis localhost:6379 -o json | jq .
```

**Example Output (Text):**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Redis Diagnostic Report - localhost:6379
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: ⚠️  Warning
Duration: 1.23s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Findings
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️  High Memory Usage (Warning)
   Category: memory
   Memory usage is at 85.3% of maximum
   
   Evidence:
     - used_memory: 8.53 GB
     - maxmemory: 10.00 GB
     - fragmentation_ratio: 1.12
   
   Remediation:
     Consider increasing maxmemory or implementing eviction policies

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Metrics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
memory_used_bytes           8530157568
memory_max_bytes            10000000000
memory_fragmentation_ratio  1.12
connected_clients           42
keyspace_hit_rate           0.95
ops_per_sec                 1250

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Suggestions
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  1. Increase maxmemory to 12GB or implement volatile-lru eviction policy (Priority: High)
  2. Monitor memory usage trends and set up alerts (Priority: Medium)
  3. Review large keys using MEMORY USAGE command (Priority: Low)
```

**Example Output (JSON):**
```json
{
  "plugin_id": "redis-diagnostics",
  "target_name": "localhost:6379",
  "status": "warning",
  "findings": [
    {
      "severity": "warning",
      "category": "memory",
      "title": "High Memory Usage",
      "description": "Memory usage is at 85.3% of maximum",
      "evidence": {
        "used_memory": 8530157568,
        "maxmemory": 10000000000,
        "fragmentation_ratio": 1.12
      },
      "remediation": "Consider increasing maxmemory or implementing eviction policies"
    }
  ],
  "metrics": {
    "memory_used_bytes": 8530157568,
    "memory_max_bytes": 10000000000,
    "memory_fragmentation_ratio": 1.12,
    "connected_clients": 42,
    "keyspace_hit_rate": 0.95,
    "ops_per_sec": 1250
  },
  "suggestions": [
    {
      "description": "Increase maxmemory to 12GB or implement volatile-lru eviction policy",
      "priority": "high"
    }
  ],
  "timestamp": "2025-12-17T08:00:00Z",
  "duration_ms": 1230
}
```

### Kafka Diagnostics

**Basic diagnosis:**
```bash
ksa diagnose kafka broker1:9092,broker2:9092,broker3:9092
```

**With SASL authentication:**
```bash
ksa diagnose kafka broker1:9092 \
  --sasl-mechanism SCRAM-SHA-512 \
  --sasl-username user \
  --sasl-password pass
```

**Consumer lag monitoring:**
```bash
ksa diagnose kafka broker1:9092 \
  --consumer-group my-group \
  --categories consumer
```

**With TLS:**
```bash
ksa diagnose kafka broker1:9092 \
  --tls \
  --ca-cert /path/to/ca.crt
```

### MySQL Diagnostics

**Basic diagnosis:**
```bash
ksa diagnose mysql "user:pass@tcp(localhost:3306)/mysql"
```

**Replication status:**
```bash
ksa diagnose mysql "user:pass@tcp(localhost:3306)/mysql" \
  --categories replication
```

**Slow query analysis:**
```bash
ksa diagnose mysql "user:pass@tcp(localhost:3306)/mysql" \
  --categories performance \
  --slow-query-threshold 1s
```

**Connection pool diagnostics:**
```bash
ksa diagnose mysql "user:pass@tcp(localhost:3306)/mysql" \
  --categories connection
```

## Plugin Management

### List Plugins

```bash
ksa plugin list
```

**Output:**
```
ID                     NAME                VERSION  TYPE        STATE      CAPABILITIES
redis-diagnostics      Redis Diagnostics   1.0.0    Middleware  Running    health-check,metrics,diagnose,execute
kafka-diagnostics      Kafka Diagnostics   1.0.0    Middleware  Running    health-check,metrics,diagnose,lag-monitor
mysql-diagnostics      MySQL Diagnostics   1.0.0    Middleware  Running    health-check,metrics,diagnose,slow-query
```

**JSON output:**
```bash
ksa plugin list -o json | jq .
```

### Plugin Information

```bash
ksa plugin info redis-diagnostics
```

**Output:**
```
Plugin: redis-diagnostics
Name: Redis Diagnostics Plugin
Version: 1.0.0
Type: Middleware
State: Running
Description: Comprehensive Redis diagnostics including memory, connections, replication, and performance
Author: KubeStack AI Team
Homepage: https://github.com/kubestack-ai/kubestack-ai
License: Apache-2.0

Capabilities:
  - health-check
  - metrics
  - diagnose
  - execute

Supported Versions:
  - 5.x
  - 6.x
  - 7.x

Configuration Schema:
  {
    "type": "object",
    "properties": {
      "default_timeout": {
        "type": "string",
        "default": "30s"
      },
      "max_slow_logs": {
        "type": "integer",
        "default": 100
      }
    }
  }
```

### Enable/Disable Plugins

```bash
# Disable a plugin
ksa plugin disable redis-diagnostics

# Enable a plugin
ksa plugin enable redis-diagnostics
```

## Configuration

### Configuration File

Create a configuration file at `~/.ksa.yaml`:

```yaml
# LLM Configuration
llm:
  provider: openai
  model: gpt-4
  api_key: ${OPENAI_API_KEY}  # Or use environment variable
  temperature: 0.7
  max_tokens: 2000

# Plugin Configuration
plugins:
  directory: /etc/kubestack-ai/plugins
  auto_discover: true
  builtin:
    redis-diagnostics:
      enabled: true
      settings:
        default_timeout: "30s"
        max_slow_logs: 100
    kafka-diagnostics:
      enabled: true
      settings:
        default_timeout: "60s"
        lag_threshold: 10000
    mysql-diagnostics:
      enabled: true
      settings:
        default_timeout: "30s"
        slow_query_threshold: "1s"

# Output Configuration
output:
  format: text  # text, json, yaml
  color: auto   # auto, always, never
  verbose: false

# Logging
logging:
  level: info
  format: json
  output: /var/log/kubestack-ai/ksa.log

# Middleware Connections (optional presets)
middleware:
  redis:
    production:
      endpoints:
        - redis-cluster-0:6379
        - redis-cluster-1:6379
        - redis-cluster-2:6379
      password: ${REDIS_PASSWORD}
      mode: cluster
    development:
      endpoints:
        - localhost:6379
```

### Environment Variables

```bash
# LLM Provider
export OPENAI_API_KEY=sk-...

# Middleware Credentials
export REDIS_PASSWORD=secret
export KAFKA_SASL_PASSWORD=secret
export MYSQL_PASSWORD=secret

# KSA Configuration
export KSA_CONFIG=/path/to/config.yaml
export KSA_LOG_LEVEL=debug
```

### Using Configuration

```bash
# Use specific config file
ksa --config /path/to/config.yaml diagnose redis localhost:6379

# Use preset connection
ksa diagnose redis production

# Override with flags
ksa --log-level debug diagnose redis localhost:6379
```

## Advanced Usage

### Diagnosis Options

**Depth levels:**
- `quick`: Fast checks only (< 5s)
- `standard`: Normal diagnostics (default, < 30s)
- `deep`: Comprehensive analysis (< 2m)

```bash
ksa diagnose redis localhost:6379 --depth deep
```

**Timeout:**
```bash
ksa diagnose redis localhost:6379 --timeout 60s
```

**Category selection:**
```bash
# Single category
ksa diagnose redis localhost:6379 --categories memory

# Multiple categories
ksa diagnose redis localhost:6379 --categories memory,connection,replication
```

### Output Formats

**Text (default):**
```bash
ksa diagnose redis localhost:6379
```

**JSON:**
```bash
ksa diagnose redis localhost:6379 -o json
```

**YAML:**
```bash
ksa diagnose redis localhost:6379 -o yaml
```

**Piping to jq:**
```bash
ksa diagnose redis localhost:6379 -o json | \
  jq '.findings[] | select(.severity == "error")'
```

### Scripting and Automation

**Check for critical issues:**
```bash
#!/bin/bash
result=$(ksa diagnose redis localhost:6379 -o json)
status=$(echo "$result" | jq -r '.status')

if [ "$status" == "critical" ]; then
  echo "CRITICAL: Redis has critical issues!"
  echo "$result" | jq '.findings'
  exit 1
fi
```

**Monitor consumer lag:**
```bash
#!/bin/bash
lag=$(ksa diagnose kafka broker:9092 \
  --consumer-group my-group \
  -o json | jq '.metrics.total_lag')

if [ "$lag" -gt 10000 ]; then
  echo "ALERT: Consumer lag is $lag messages"
  # Send alert
fi
```

**Weekly slow query report:**
```bash
#!/bin/bash
ksa diagnose mysql "user:pass@tcp(db:3306)/mysql" \
  --categories performance \
  -o yaml > /reports/mysql_$(date +%Y%m%d).yaml
```

## Shell Completion

### Bash

```bash
# Generate completion script
ksa completion bash > /etc/bash_completion.d/ksa

# Or for current session
source <(ksa completion bash)
```

### Zsh

```bash
# Generate completion script
ksa completion zsh > "${fpath[1]}/_ksa"

# Reload completions
compinit
```

### Fish

```bash
# Generate completion script
ksa completion fish > ~/.config/fish/completions/ksa.fish
```

## Troubleshooting

### Common Issues

**Connection refused:**
```bash
# Check connectivity first
telnet localhost 6379

# Try with explicit timeout
ksa diagnose redis localhost:6379 --timeout 10s
```

**Authentication failed:**
```bash
# Verify credentials
ksa diagnose redis localhost:6379 --password correct-password

# Use environment variable
export REDIS_PASSWORD=correct-password
ksa diagnose redis localhost:6379
```

**Plugin not found:**
```bash
# List available plugins
ksa plugin list

# Check plugin status
ksa plugin info redis-diagnostics

# Enable plugin if disabled
ksa plugin enable redis-diagnostics
```

**Timeout errors:**
```bash
# Increase timeout
ksa diagnose redis localhost:6379 --timeout 120s

# Use quick depth for faster results
ksa diagnose redis localhost:6379 --depth quick
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
ksa --log-level debug diagnose redis localhost:6379
```

View detailed plugin information:

```bash
ksa plugin info redis-diagnostics --verbose
```

## Best Practices

1. **Use Configuration Files**: Store connection presets in `~/.ksa.yaml`
2. **Environment Variables**: Use env vars for sensitive credentials
3. **JSON Output**: Use `-o json` for automation and scripting
4. **Depth Selection**: Use `--depth quick` for frequent checks, `--depth deep` for investigation
5. **Category Focus**: Use `--categories` to focus on specific areas
6. **Timeouts**: Set appropriate `--timeout` values for slow systems
7. **Shell Completion**: Enable completion for better UX

## Examples

### Daily Health Check

```bash
#!/bin/bash
# daily_check.sh

echo "Running daily middleware health checks..."

# Redis
ksa diagnose redis production -o json > /tmp/redis_health.json
if [ $? -ne 0 ]; then
  echo "❌ Redis check failed"
else
  echo "✅ Redis check passed"
fi

# Kafka
ksa diagnose kafka production -o json > /tmp/kafka_health.json
if [ $? -ne 0 ]; then
  echo "❌ Kafka check failed"
else
  echo "✅ Kafka check passed"
fi

# MySQL
ksa diagnose mysql production -o json > /tmp/mysql_health.json
if [ $? -ne 0 ]; then
  echo "❌ MySQL check failed"
else
  echo "✅ MySQL check passed"
fi
```

### Prometheus Exporter

```bash
#!/bin/bash
# Export KSA metrics to Prometheus format

while true; do
  result=$(ksa diagnose redis localhost:6379 -o json)
  
  # Extract metrics
  mem_used=$(echo "$result" | jq '.metrics.memory_used_bytes')
  ops_per_sec=$(echo "$result" | jq '.metrics.ops_per_sec')
  
  # Write to metrics file
  cat > /var/lib/prometheus/node_exporter/ksa_redis.prom <<EOF
# HELP ksa_redis_memory_used_bytes Redis memory usage in bytes
# TYPE ksa_redis_memory_used_bytes gauge
ksa_redis_memory_used_bytes $mem_used

# HELP ksa_redis_ops_per_sec Redis operations per second
# TYPE ksa_redis_ops_per_sec gauge
ksa_redis_ops_per_sec $ops_per_sec
EOF

  sleep 60
done
```

### Alert on Replication Lag

```bash
#!/bin/bash
# Check MySQL replication and alert if lagging

lag=$(ksa diagnose mysql "user:pass@tcp(db:3306)/mysql" \
  --categories replication \
  -o json | jq '.metrics.seconds_behind_master')

if [ "$lag" -gt 60 ]; then
  # Send Slack alert
  curl -X POST -H 'Content-type: application/json' \
    --data "{\"text\":\"MySQL replication lag: ${lag} seconds\"}" \
    $SLACK_WEBHOOK_URL
fi
```

## Support

For issues, feature requests, or contributions:
- GitHub: https://github.com/kubestack-ai/kubestack-ai
- Documentation: https://docs.kubestack.ai
- Discord: https://discord.gg/kubestack-ai
