# Rule Engine Usage Guide

This guide explains how to define and manage diagnostic rules for the KubeStack-AI Knowledge Base.

## Rule Structure

Rules are defined in YAML format. Each rule must have a unique ID and a condition expression.

### Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `id` | string | Unique identifier | Yes |
| `name` | string | Human-readable name | Yes |
| `middleware_type` | string | Target middleware (redis, mysql, kafka) | Yes |
| `category` | string | Issue category (performance, stability, security) | No |
| `severity` | string | Severity level (CRITICAL, HIGH, MEDIUM, LOW) | Yes |
| `condition` | string | Expression to evaluate | Yes |
| `recommendation` | string | Steps to resolve the issue | Yes |
| `priority` | int | Higher values mean higher priority | No (Default 0) |
| `tags` | list | Keywords for categorization | No |

### Condition Syntax

The condition engine supports standard logical and comparison operators.

- **Comparisons**: `>`, `<`, `>=`, `<=`, `==`, `!=`
- **Logic**: `&&` (AND), `||` (OR), `!` (NOT)
- **Parentheses**: `()` for grouping

**Example Conditions**:

- `memory_usage > 80`
- `connection_count > 1000 && hit_rate < 0.9`
- `role == "master" || role == "slave"`

### Variables

Variables in the condition expression must match the keys in the `Metrics` map collected by the diagnostic plugins.

Common variables include:
- `cpu_usage`
- `memory_usage`
- `disk_usage`
- `connection_count`
- `replication_lag`

## Adding New Rules

1.  Navigate to `internal/knowledge/repository/`.
2.  Open or create a YAML file for the middleware (e.g., `postgres_rules.yaml`).
3.  Add the rule definition.

**Example**:

```yaml
- id: pg-conn-limit
  name: PostgreSQL Connection Limit Reached
  middleware_type: postgres
  severity: HIGH
  condition: active_connections > max_connections * 0.9
  recommendation: |
    1. Check for connection leaks.
    2. Increase max_connections in postgresql.conf.
    3. Use a connection pooler like PgBouncer.
  priority: 90
  tags: [connection, configuration]
```

## Testing Rules

You can test rules using the REST API:

1.  **Start the server**: `go run cmd/ksa/main.go server start`
2.  **Add a rule**:
    ```bash
    curl -X POST http://localhost:8080/api/v1/knowledge/rules \
      -H "Content-Type: application/json" \
      -d '{
        "id": "test-rule-1",
        "name": "Test Rule",
        "middleware_type": "redis",
        "severity": "LOW",
        "condition": "test_metric > 10",
        "recommendation": "Fix it."
      }'
    ```
3.  **Verify**: Trigger a diagnosis that produces the matching metric.
