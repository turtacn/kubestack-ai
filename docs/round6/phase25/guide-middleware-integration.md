# Middleware Integration Guide

## Overview

This guide explains how to integrate KubeStack AI's diagnostic capabilities with various middleware systems in your infrastructure.

## Supported Middleware

### Redis
- **Versions**: 5.x, 6.x, 7.x
- **Modes**: Standalone, Sentinel, Cluster
- **Diagnostics**: Memory, connections, replication, persistence, performance
- **Authentication**: Password, ACL
- **TLS**: Supported

### Kafka
- **Versions**: 2.x, 3.x
- **Features**: Broker health, consumer lag, topic analysis
- **Authentication**: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
- **TLS**: Supported

### MySQL
- **Versions**: 5.7, 8.x
- **Features**: Replication status, slow query analysis, connection pool
- **Authentication**: Username/password
- **TLS**: Supported

## Connection Setup

### 1. Direct Connection (Command Line)

```bash
# Redis
ksa diagnose redis localhost:6379 --password mypass

# Kafka
ksa diagnose kafka broker1:9092,broker2:9092 \
  --sasl-mechanism SCRAM-SHA-512 \
  --sasl-username user \
  --sasl-password pass

# MySQL
ksa diagnose mysql "user:pass@tcp(localhost:3306)/mysql"
```

### 2. Configuration File

Create `~/.ksa.yaml`:

```yaml
middleware:
  redis:
    production:
      endpoints:
        - redis.prod.svc:6379
      password: ${REDIS_PASSWORD}
  kafka:
    production:
      endpoints:
        - kafka-0:9092
        - kafka-1:9092
      auth:
        mechanism: SCRAM-SHA-512
        username: ${KAFKA_USERNAME}
        password: ${KAFKA_PASSWORD}
  mysql:
    production:
      endpoint: mysql.prod.svc:3306
      username: ${MYSQL_USERNAME}
      password: ${MYSQL_PASSWORD}
```

Usage:
```bash
ksa diagnose redis production
ksa diagnose kafka production
ksa diagnose mysql production
```

### 3. Kubernetes Secret Integration

Store credentials in secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: redis-credentials
type: Opaque
stringData:
  password: your-redis-password
```

Reference in configuration:

```yaml
redis:
  production:
    endpoints:
      - redis.svc:6379
    auth:
      secret_ref: redis-credentials
      secret_key: password
```

## Diagnostic Categories

### Redis

```bash
# All categories
ksa diagnose redis localhost:6379

# Specific categories
ksa diagnose redis localhost:6379 --categories memory,replication

# Available categories:
# - memory: Memory usage, fragmentation, eviction
# - connection: Client connections, idle connections
# - replication: Master-slave lag, sync status
# - persistence: RDB/AOF status, backups
# - performance: Ops/sec, slow logs, hit rate
```

### Kafka

```bash
# All categories
ksa diagnose kafka broker:9092

# Specific categories
ksa diagnose kafka broker:9092 --categories consumer,broker

# Available categories:
# - broker: Broker availability, ISR status
# - consumer: Consumer lag, group status
# - topic: Topic configuration, partitions
# - partition: Partition distribution, under-replicated
```

### MySQL

```bash
# All categories
ksa diagnose mysql "user:pass@tcp(host:3306)/mysql"

# Specific categories
ksa diagnose mysql "..." --categories replication,performance

# Available categories:
# - replication: Master-slave status, lag
# - connection: Connection pool, idle connections
# - performance: Slow queries, full table scans
# - innodb: Buffer pool, transactions
```

## Diagnostic Depth

Control the depth of analysis:

```bash
# Quick (< 5s)
ksa diagnose redis localhost:6379 --depth quick

# Standard (< 30s, default)
ksa diagnose redis localhost:6379 --depth standard

# Deep (< 2m)
ksa diagnose redis localhost:6379 --depth deep
```

## Output Formats

### Text (Default)

```bash
ksa diagnose redis localhost:6379
```

Output:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Redis Diagnostic Report - localhost:6379
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: ⚠️  Warning
Duration: 1.23s
...
```

### JSON

```bash
ksa diagnose redis localhost:6379 -o json | jq .
```

### YAML

```bash
ksa diagnose redis localhost:6379 -o yaml
```

## Automation Examples

### Health Check Script

```bash
#!/bin/bash
result=$(ksa diagnose redis production -o json)
status=$(echo "$result" | jq -r '.status')

if [ "$status" == "critical" ]; then
  echo "CRITICAL: Redis issues detected"
  exit 1
fi
```

### Prometheus Exporter

```bash
#!/bin/bash
while true; do
  result=$(ksa diagnose redis production -o json)
  
  # Extract metrics
  memory_used=$(echo "$result" | jq '.metrics.memory_used_bytes')
  
  # Write metrics file
  cat > /var/lib/node_exporter/ksa_redis.prom <<EOF
ksa_redis_memory_used_bytes $memory_used
EOF
  
  sleep 60
done
```

### Alerting Integration

```bash
#!/bin/bash
result=$(ksa diagnose kafka production -o json)
lag=$(echo "$result" | jq '.metrics.total_lag')

if [ "$lag" -gt 10000 ]; then
  curl -X POST $SLACK_WEBHOOK \
    -d "{\"text\":\"Kafka lag: $lag messages\"}"
fi
```

## Best Practices

### 1. Credential Management

- Use environment variables for passwords
- Store secrets in Kubernetes Secrets
- Never commit credentials to version control
- Rotate credentials regularly

### 2. Regular Diagnostics

- Run health checks every 5 minutes
- Deep diagnostics weekly or monthly
- Automate with cron or Kubernetes CronJob

### 3. Monitoring Integration

- Export metrics to Prometheus
- Create Grafana dashboards
- Set up alerts for critical issues

### 4. Performance Considerations

- Use `--depth quick` for frequent checks
- Run deep diagnostics during maintenance windows
- Limit concurrent diagnostics

### 5. Security

- Enable TLS for production
- Use least-privilege credentials
- Audit diagnostic access
- Secure diagnostic outputs

## Troubleshooting

### Connection Issues

```bash
# Test connectivity first
telnet redis.prod.svc 6379

# Increase timeout
ksa diagnose redis production --timeout 60s

# Check authentication
ksa diagnose redis production --password $REDIS_PASSWORD -v
```

### Authentication Failures

```bash
# Verify credentials
echo $REDIS_PASSWORD

# Test with redis-cli
redis-cli -h redis.prod.svc -a $REDIS_PASSWORD ping

# Check ACL permissions (Redis 6+)
redis-cli -h redis.prod.svc -a $REDIS_PASSWORD ACL WHOAMI
```

### TLS Issues

```bash
# Verify certificate
openssl s_client -connect redis.prod.svc:6380

# Disable certificate verification (development only)
ksa diagnose redis production --tls --insecure-skip-verify
```

## Integration Patterns

### CI/CD Integration

```yaml
# .gitlab-ci.yml
test-redis-health:
  stage: test
  script:
    - ksa diagnose redis staging -o json > redis-report.json
    - |
      if [ "$(jq -r '.status' redis-report.json)" == "critical" ]; then
        echo "Redis health check failed"
        exit 1
      fi
  artifacts:
    reports:
      redis-report.json
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: redis-diagnostics
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: ksa
            image: kubestack-ai/ksa:latest
            command:
            - /bin/sh
            - -c
            - |
              ksa diagnose redis production -o json > /tmp/report.json
              # Post to monitoring system
          restartPolicy: OnFailure
```

### Webhook Integration

```bash
# Send diagnostics to webhook
result=$(ksa diagnose redis production -o json)
curl -X POST https://monitoring.example.com/diagnostics \
  -H "Content-Type: application/json" \
  -d "$result"
```

## Advanced Features

### Custom Thresholds

Configure custom alert thresholds in `~/.ksa.yaml`:

```yaml
plugins:
  redis-diagnostics:
    settings:
      memory_warning_percent: 75
      memory_critical_percent: 90
      connection_warning_percent: 70
```

### Multiple Instances

Diagnose multiple instances:

```bash
for instance in redis-01 redis-02 redis-03; do
  ksa diagnose redis $instance -o json > ${instance}-report.json
done
```

### Aggregated Reports

```bash
# Diagnose all middleware
ksa diagnose redis production -o json > redis.json
ksa diagnose kafka production -o json > kafka.json
ksa diagnose mysql production -o json > mysql.json

# Combine reports
jq -s '{redis: .[0], kafka: .[1], mysql: .[2]}' \
  redis.json kafka.json mysql.json > combined-report.json
```

## Support

For issues or questions:
- GitHub: https://github.com/kubestack-ai/kubestack-ai
- Documentation: https://docs.kubestack.ai
- Discord: https://discord.gg/kubestack-ai
