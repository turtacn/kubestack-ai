# Monitoring API

## Get Metrics

Retrieves metrics for a specific type and time range.

**Endpoint:** `GET /api/v1/monitor/metrics`

**Parameters:**

*   `type` (required): The type of metric (e.g., `redis`, `k8s_node`).
*   `instance` (optional): Filter by instance name.
*   `range` (optional): Time range duration (default: `1h`).

**Response:**

```json
{
  "metrics": [
    {
      "Name": "redis_memory_usage_bytes",
      "Value": 102400,
      "Timestamp": "2023-10-27T10:00:00Z",
      "Labels": {
        "instance": "redis-0"
      }
    }
  ],
  "count": 1
}
```

## Get Alert History

Retrieves the history of alert events.

**Endpoint:** `GET /api/v1/monitor/alerts/history`

**Parameters:**

*   `severity` (optional): Filter by severity (`critical`, `warning`, `info`).
*   `status` (optional): Filter by status (`firing`, `resolved`).
*   `limit` (optional): Limit the number of results (default: `100`).

**Response:**

```json
{
  "alerts": [
    {
      "rule_name": "high-cpu",
      "severity": "critical",
      "status": "firing",
      "value": 95.5,
      "fired_at": "2023-10-27T10:05:00Z"
    }
  ],
  "total": 1
}
```

## Create Silence

Silences an alert rule for a specified duration.

**Endpoint:** `POST /api/v1/monitor/alerts/silence`

**Body:**

```json
{
  "rule_name": "high-cpu",
  "duration": "1h",
  "comment": "Maintenance window",
  "labels": {
    "instance": "node-1"
  }
}
```

**Response:**

```json
{
  "silence_id": "uuid-string",
  "expires_at": "2023-10-27T11:05:00Z"
}
```
