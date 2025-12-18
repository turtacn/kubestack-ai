# KSA CLI Command Reference

**Version**: 0.1.0  
**Phase**: P26 - CLI Full-Scenario Validation  
**Last Updated**: 2024-12-18

## Table of Contents

1. [Overview](#overview)
2. [Global Options](#global-options)
3. [Commands](#commands)
   - [ksa](#ksa-root-command)
   - [ksa diagnose](#ksa-diagnose)
   - [ksa ask](#ksa-ask)
   - [ksa fix](#ksa-fix)
   - [ksa server](#ksa-server)
   - [ksa plugin](#ksa-plugin)
   - [ksa monitor](#ksa-monitor)
   - [ksa alert](#ksa-alert)
   - [ksa version](#ksa-version)
4. [Output Formats](#output-formats)
5. [Middleware Types](#middleware-types)
6. [Examples](#examples)
7. [Environment Variables](#environment-variables)
8. [Exit Codes](#exit-codes)

## Overview

The `ksa` command-line interface provides intelligent middleware diagnosis, analysis, and remediation capabilities powered by AI. It supports multiple middleware types including Redis, MySQL, Kafka, Elasticsearch, and PostgreSQL.

### Installation

```bash
# From source
git clone https://github.com/kubestack-ai/kubestack-ai
cd kubestack-ai
make build

# The binary will be available at ./ksa
```

### Quick Start

```bash
# Check version
ksa version

# Diagnose a Redis instance
ksa diagnose redis --instance localhost:6379

# Ask a question
ksa ask "How to optimize MySQL query performance?"

# Start the API server
ksa server
```

## Global Options

These options are available for all commands:

### --config

Specify a configuration file path.

**Default**: `configs/config.yaml`

**Usage**:
```bash
ksa --config /path/to/config.yaml diagnose redis --instance localhost:6379
```

### --log-level

Set the logging level.

**Values**: `debug`, `info`, `warn`, `error`, `fatal`

**Default**: `info`

**Usage**:
```bash
ksa --log-level debug diagnose redis --instance localhost:6379
```

### --output, -o

Set the output format.

**Values**: `text`, `json`, `yaml`, `table`

**Default**: `text`

**Usage**:
```bash
ksa diagnose redis --instance localhost:6379 -o json
```

## Commands

### ksa (Root Command)

The root command displays help information and available subcommands.

**Usage**:
```bash
ksa [command]
```

**Available Commands**:
- `diagnose` - Diagnose middleware issues
- `ask` - Ask questions about middleware
- `fix` - Apply fixes for diagnosed issues
- `server` - Start the API server
- `monitor` - Monitor middleware instances
- `alert` - Manage alert rules
- `version` - Display version information

**Flags**:
- `-h, --help` - Help for ksa

**Examples**:
```bash
# Show help
ksa --help

# Show help for a specific command
ksa diagnose --help
```

---

### ksa diagnose

Diagnose a middleware instance to identify performance issues, configuration errors, and anomalies.

**Usage**:
```bash
ksa diagnose [middleware-type] [flags]
```

**Arguments**:
- `middleware-type` - Type of middleware (required)

**Flags**:

#### --instance, -i (required)

Instance name or connection string.

**Examples**:
```bash
# Host:port format
--instance localhost:6379

# Connection string format
--instance redis://localhost:6379/0

# Kubernetes service
--instance redis-master.default.svc.cluster.local:6379
```

#### --namespace, -n

Kubernetes namespace (if applicable).

**Default**: `default`

**Usage**:
```bash
ksa diagnose redis --instance my-redis --namespace production
```

#### --output, -o

Output format.

**Values**: `text`, `json`, `yaml`

**Default**: `text`

#### --async

Run diagnosis asynchronously.

**Default**: `false`

**Usage**:
```bash
ksa diagnose redis --instance localhost:6379 --async
```

#### --dry-run

Perform a dry run without executing actions.

**Default**: `false`

**Examples**:
```bash
# Basic diagnosis
ksa diagnose redis --instance localhost:6379

# Diagnosis with JSON output
ksa diagnose redis --instance localhost:6379 -o json

# Diagnosis in specific namespace
ksa diagnose mysql --instance db-01 --namespace prod

# Async diagnosis
ksa diagnose kafka --instance localhost:9092 --async

# Dry run
ksa diagnose postgresql --instance localhost:5432 --dry-run
```

**Output**:

Text format example:
```
========================================
Diagnosis Report
========================================
Report ID: diag-20240118-143022
Target: Redis @ localhost:6379
Status: Completed
Duration: 5.2s

Issues Found: 2
1. [WARNING] High memory usage (85%)
2. [INFO] Slow query detected

Recommendations: 3
1. Increase maxmemory limit
2. Enable query logging
3. Review slow queries

For more details, run with -o json
```

JSON format example:
```json
{
  "version": "v1",
  "id": "diag-20240118-143022",
  "target": {
    "middleware": "Redis",
    "instance": "localhost:6379",
    "namespace": "default"
  },
  "status": "completed",
  "start_time": "2024-01-18T14:30:22Z",
  "end_time": "2024-01-18T14:30:27Z",
  "duration": "5.2s",
  "summary": "Found 2 issues with minor severity",
  "issues": [
    {
      "id": "issue-001",
      "severity": "warning",
      "category": "performance",
      "title": "High memory usage",
      "description": "Memory usage is at 85%",
      "impact": "May affect performance under load"
    }
  ],
  "recommendations": [
    {
      "id": "rec-001",
      "priority": "medium",
      "category": "optimization",
      "title": "Increase memory limit",
      "description": "Consider increasing Redis memory limit",
      "actions": [
        {
          "type": "config_change",
          "description": "Update maxmemory setting",
          "command": "CONFIG SET maxmemory 2gb"
        }
      ]
    }
  ]
}
```

---

### ksa ask

Ask natural language questions about middleware and get AI-powered answers.

**Usage**:
```bash
ksa ask [question]
```

**Arguments**:
- `question` - Natural language question (required)

**Flags**:

#### --context, -c

Provide additional context for the question.

**Usage**:
```bash
ksa ask "How to optimize this query?" --context "MySQL 8.0, 1TB database"
```

#### --stream

Enable streaming output (receive answer in real-time).

**Default**: `true`

**Examples**:
```bash
# Basic question
ksa ask "What causes high CPU in Redis?"

# Question with context
ksa ask "How to fix replication lag?" --context "MySQL master-slave setup"

# Non-streaming mode
ksa ask "Best practices for Kafka tuning" --stream=false

# With JSON output
ksa ask "Explain Redis persistence" -o json
```

**Output**:
```
Question: What causes high CPU in Redis?

Answer:
High CPU usage in Redis can be caused by several factors:

1. Expensive commands: O(N) commands like KEYS, SMEMBERS on large sets
2. High query rate: Too many operations per second
3. Persistence overhead: BGSAVE/AOF rewrite operations
4. Memory management: Frequent evictions or memory defragmentation

Recommendations:
- Use SCAN instead of KEYS
- Implement connection pooling
- Monitor slow log
- Adjust maxmemory-policy

For more specific advice, share your Redis configuration and metrics.
```

---

### ksa fix

Apply recommended fixes for diagnosed issues.

**Usage**:
```bash
ksa fix [flags]
```

**Flags**:

#### --id (required)

Diagnosis ID from a previous diagnose command.

**Usage**:
```bash
ksa fix --id diag-20240118-143022
```

#### --auto-approve

Automatically approve all fixes without prompting.

**Default**: `false`

**Usage**:
```bash
ksa fix --id diag-20240118-143022 --auto-approve
```

#### --select

Select specific fixes to apply (comma-separated IDs).

**Usage**:
```bash
ksa fix --id diag-20240118-143022 --select rec-001,rec-003
```

#### --dry-run

Show what would be fixed without applying changes.

**Default**: `false`

**Examples**:
```bash
# Interactive fix (prompts for approval)
ksa fix --id diag-20240118-143022

# Auto-approve all fixes
ksa fix --id diag-20240118-143022 --auto-approve

# Apply specific fixes only
ksa fix --id diag-20240118-143022 --select rec-001,rec-002

# Dry run to preview changes
ksa fix --id diag-20240118-143022 --dry-run
```

**Output**:
```
========================================
Fix Plan for diag-20240118-143022
========================================

The following fixes will be applied:

1. [rec-001] Increase memory limit
   Action: Update maxmemory setting to 2gb
   Command: CONFIG SET maxmemory 2gb
   Risk: Low

2. [rec-002] Enable query logging
   Action: Enable slow log
   Command: CONFIG SET slowlog-log-slower-than 10000
   Risk: Low

Apply these fixes? [y/N]: y

Applying fixes...
✓ rec-001: Memory limit updated
✓ rec-002: Query logging enabled

All fixes applied successfully!
```

---

### ksa server

Start the KubeStack-AI API server.

**Usage**:
```bash
ksa server [flags]
```

**Flags**:

#### --port

Port to listen on.

**Default**: `8080` (from config)

**Usage**:
```bash
ksa server --port 9090
```

#### --host

Host address to bind to.

**Default**: `0.0.0.0` (from config)

**Usage**:
```bash
ksa server --host 127.0.0.1
```

#### --enable-cors

Enable CORS for web access.

**Default**: `true`

#### --tls-cert

TLS certificate file path (for HTTPS).

**Usage**:
```bash
ksa server --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem
```

#### --tls-key

TLS private key file path (for HTTPS).

**Examples**:
```bash
# Start server on default port
ksa server

# Start on custom port
ksa server --port 9090

# Start with HTTPS
ksa server --tls-cert cert.pem --tls-key key.pem

# Start on localhost only
ksa server --host 127.0.0.1
```

**Output**:
```
Starting KubeStack-AI API Server...
Version: v0.1.0
Configuration: configs/config.yaml

Server listening on http://0.0.0.0:8080
API documentation: http://0.0.0.0:8080/docs
Health check: http://0.0.0.0:8080/health
```

---

## ksa plugin

Manage KubeStack-AI plugins for middleware diagnostics and operations.

**Synopsis**:
```bash
ksa plugin [command] [flags]
```

**Available Commands**:
- `list` - List all available plugins
- `info` - Show detailed information about a plugin
- `enable` - Enable a plugin
- `disable` - Disable a plugin

### Subcommands

#### ksa plugin list

List all available plugins with their status.

**Usage**:
```bash
ksa plugin list [flags]
```

**Flags**:
- `-o, --output string` - Output format (text, json, yaml)

**Examples**:
```bash
# List all plugins
ksa plugin list

# List plugins in JSON format
ksa plugin list -o json
```

**Output**:
```
PLUGIN                 TYPE          VERSION  DESCRIPTION
------                 ----          -------  -----------
redis-diagnostics      diagnostics   1.0.0    Redis diagnostics and health checks
mysql-diagnostics      diagnostics   1.0.0    MySQL diagnostics and query analysis
kafka-diagnostics      diagnostics   1.0.0    Kafka cluster monitoring and diagnosis
elasticsearch-diag     diagnostics   1.0.0    Elasticsearch cluster health analysis
postgresql-diag        diagnostics   1.0.0    PostgreSQL performance diagnostics
```

#### ksa plugin info

Show detailed information about a specific plugin.

**Usage**:
```bash
ksa plugin info <plugin-name> [flags]
```

**Arguments**:
- `<plugin-name>` - Name of the plugin (required)

**Flags**:
- `-o, --output string` - Output format (text, json, yaml)

**Examples**:
```bash
# Show plugin information
ksa plugin info redis-diagnostics

# Show information in JSON format
ksa plugin info redis-diagnostics -o json
```

**Output**:
```
Plugin: redis-diagnostics
Type: diagnostics
Version: 1.0.0
Description: Redis diagnostics plugin for health checks, performance analysis, and troubleshooting
```

#### ksa plugin enable

Enable a plugin to make it available for use.

**Usage**:
```bash
ksa plugin enable <plugin-name>
```

**Arguments**:
- `<plugin-name>` - Name of the plugin (required)

**Examples**:
```bash
# Enable a plugin
ksa plugin enable redis-diagnostics
```

**Output**:
```
Plugin 'redis-diagnostics' enabled successfully
```

#### ksa plugin disable

Disable a plugin to prevent it from being used.

**Usage**:
```bash
ksa plugin disable <plugin-name>
```

**Arguments**:
- `<plugin-name>` - Name of the plugin (required)

**Examples**:
```bash
# Disable a plugin
ksa plugin disable redis-diagnostics
```

**Output**:
```
Plugin 'redis-diagnostics' disabled successfully

Press Ctrl+C to stop
```

---

### ksa monitor

Monitor middleware instances and collect metrics.

**Usage**:
```bash
ksa monitor [middleware-type] [flags]
```

**Flags**:

#### --instance, -i

Instance to monitor.

#### --interval

Monitoring interval.

**Default**: `60s`

**Usage**:
```bash
ksa monitor redis --instance localhost:6379 --interval 30s
```

#### --duration

Total monitoring duration.

**Default**: `infinite`

**Usage**:
```bash
ksa monitor redis --instance localhost:6379 --duration 10m
```

**Examples**:
```bash
# Continuous monitoring
ksa monitor redis --instance localhost:6379

# Monitor for specific duration
ksa monitor mysql --instance localhost:3306 --duration 1h

# Custom monitoring interval
ksa monitor kafka --instance localhost:9092 --interval 10s
```

---

### ksa alert

Manage alert rules and notifications.

**Usage**:
```bash
ksa alert [subcommand]
```

**Subcommands**:
- `list` - List all alert rules
- `create` - Create a new alert rule
- `delete` - Delete an alert rule
- `test` - Test an alert rule

**Examples**:
```bash
# List all alert rules
ksa alert list

# Create alert rule (interactive)
ksa alert create

# Delete alert rule
ksa alert delete --id alert-001

# Test alert rule
ksa alert test --id alert-001
```

---

### ksa version

Display version information.

**Usage**:
```bash
ksa version [flags]
```

**Flags**:

#### --short

Display only version number.

**Examples**:
```bash
# Full version info
ksa version

# Short version
ksa version --short

# JSON output
ksa version -o json
```

**Output**:
```
KubeStack-AI v0.1.0
Build Date: 2024-01-18
Git Commit: a1b2c3d
Go Version: go1.21.5
Platform: linux/amd64
```

## Output Formats

### Text Format

Human-readable output with colors and formatting. Best for interactive use.

**Features**:
- Colored output (if terminal supports it)
- Formatted sections
- Progress indicators
- Error highlighting

**Usage**:
```bash
ksa diagnose redis --instance localhost:6379 -o text
```

### JSON Format

Machine-parseable JSON output. Best for automation and scripting.

**Features**:
- Valid JSON syntax
- Pretty-printed (indented)
- Includes all data fields
- Schema compliant

**Usage**:
```bash
ksa diagnose redis --instance localhost:6379 -o json
```

### YAML Format

Human and machine-readable YAML output. Best for configuration and review.

**Features**:
- Valid YAML syntax
- Readable structure
- Comment-friendly
- Easy to edit

**Usage**:
```bash
ksa diagnose redis --instance localhost:6379 -o yaml
```

### Table Format

Tabular output for structured data. Best for lists and comparisons.

**Features**:
- Column-aligned
- Header rows
- Separator lines
- Compact display

**Usage**:
```bash
ksa monitor list -o table
```

## Middleware Types

Supported middleware types for the `diagnose` and `monitor` commands:

### Currently Implemented

| Middleware | Type String | Default Port | Connection String Example |
|------------|-------------|--------------|---------------------------|
| Redis | `redis` | 6379 | `redis://localhost:6379/0` |
| MySQL | `mysql` | 3306 | `mysql://user:pass@localhost:3306/db` |
| Kafka | `kafka` | 9092 | `localhost:9092` |
| Elasticsearch | `elasticsearch` | 9200 | `http://localhost:9200` |
| PostgreSQL | `postgresql` | 5432 | `postgres://user:pass@localhost:5432/db` |

### Planned (TODO)

| Middleware | Type String | Status |
|------------|-------------|--------|
| MongoDB | `mongodb` | Planned |
| RabbitMQ | `rabbitmq` | Planned |
| MinIO | `minio` | Planned |
| Prometheus | `prometheus` | Planned |
| ClickHouse | `clickhouse` | Planned |

## Examples

### Common Workflows

#### 1. Diagnose and Fix Workflow

```bash
# Step 1: Diagnose issue
ksa diagnose redis --instance localhost:6379 -o json > diagnosis.json

# Step 2: Extract diagnosis ID
DIAG_ID=$(cat diagnosis.json | jq -r '.id')

# Step 3: Review recommendations
cat diagnosis.json | jq '.recommendations'

# Step 4: Apply fixes
ksa fix --id $DIAG_ID --auto-approve
```

#### 2. Continuous Monitoring

```bash
# Monitor with periodic snapshots
while true; do
  ksa monitor redis --instance localhost:6379 --duration 5m -o json >> monitoring.log
  sleep 300
done
```

#### 3. Batch Diagnosis

```bash
# Diagnose multiple instances
for instance in redis-01 redis-02 redis-03; do
  ksa diagnose redis --instance $instance:6379 -o json > diag-$instance.json
done
```

#### 4. Question and Learn

```bash
# Ask related questions
ksa ask "What are Redis persistence options?"
ksa ask "How to configure AOF in Redis?"
ksa ask "Difference between RDB and AOF?"
```

### Advanced Usage

#### Custom Configuration

```bash
# Use custom config file
ksa --config /etc/ksa/production.yaml diagnose redis --instance prod-redis:6379
```

#### Debug Mode

```bash
# Enable debug logging
ksa --log-level debug diagnose mysql --instance localhost:3306
```

#### Scripting Integration

```bash
#!/bin/bash
# Health check script
RESULT=$(ksa diagnose redis --instance $REDIS_HOST -o json)
STATUS=$(echo $RESULT | jq -r '.status')

if [ "$STATUS" != "healthy" ]; then
  echo "Alert: Redis is not healthy!"
  # Send alert
fi
```

## Environment Variables

### Configuration

- `KSA_CONFIG` - Path to configuration file
- `KSA_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `KSA_OUTPUT_FORMAT` - Default output format

### LLM Configuration

- `OPENAI_API_KEY` - OpenAI API key
- `OPENAI_MODEL` - OpenAI model to use
- `ANTHROPIC_API_KEY` - Anthropic API key

### Middleware Credentials

- `REDIS_PASSWORD` - Redis password
- `MYSQL_PASSWORD` - MySQL password
- `POSTGRES_PASSWORD` - PostgreSQL password

### Test Mode

- `KSA_TEST_MODE` - Enable test mode (true/false)

### Usage

```bash
# Set environment variables
export KSA_CONFIG=/etc/ksa/config.yaml
export KSA_LOG_LEVEL=debug
export OPENAI_API_KEY=sk-xxxxx

# Run command
ksa diagnose redis --instance localhost:6379
```

## Exit Codes

The `ksa` CLI uses standard exit codes to indicate success or failure:

| Code | Meaning | Description |
|------|---------|-------------|
| 0 | Success | Command completed successfully |
| 1 | General Error | Unspecified error occurred |
| 2 | Usage Error | Invalid command syntax or flags |
| 3 | Configuration Error | Config file not found or invalid |
| 4 | Connection Error | Failed to connect to middleware |
| 5 | Execution Error | Command execution failed |

### Usage in Scripts

```bash
#!/bin/bash
ksa diagnose redis --instance localhost:6379
EXIT_CODE=$?

case $EXIT_CODE in
  0)
    echo "Diagnosis successful"
    ;;
  4)
    echo "Connection failed - check Redis is running"
    ;;
  *)
    echo "Error occurred: $EXIT_CODE"
    ;;
esac
```

## Troubleshooting

### Common Issues

#### 1. Command Not Found

```bash
# Error: ksa: command not found
# Solution: Add ksa to PATH or use absolute path
export PATH=$PATH:/path/to/ksa
# OR
./ksa version
```

#### 2. Config File Not Found

```bash
# Error: failed to load config: open configs/config.yaml: no such file or directory
# Solution: Specify config file location
ksa --config /path/to/config.yaml diagnose redis --instance localhost:6379
```

#### 3. Connection Refused

```bash
# Error: failed to connect to redis at localhost:6379: connection refused
# Solution: Verify middleware is running
redis-cli ping
# OR
telnet localhost 6379
```

#### 4. Permission Denied

```bash
# Error: permission denied
# Solution: Check file permissions or run with appropriate user
chmod +x ksa
# OR
sudo ksa diagnose redis --instance localhost:6379
```

### Getting Help

- Check command help: `ksa [command] --help`
- View logs: `ksa --log-level debug [command]`
- Check configuration: `cat configs/config.yaml`
- Verify connectivity: Test middleware connection independently

## Additional Resources

- **Documentation**: https://github.com/kubestack-ai/kubestack-ai/tree/main/docs
- **Examples**: https://github.com/kubestack-ai/kubestack-ai/tree/main/examples
- **Issues**: https://github.com/kubestack-ai/kubestack-ai/issues
- **Community**: https://github.com/kubestack-ai/kubestack-ai/discussions

---

**Document Version**: 1.0  
**Last Updated**: 2024-12-18  
**Maintained by**: KubeStack-AI Development Team
