# Diagnosis API v1 Specification

## Overview

The Diagnosis API v1 provides a stable, versioned interface for executing middleware diagnostics and retrieving structured diagnosis reports. This API is the primary integration point for external consumers (CLI, Web UI, third-party tools).

## Report Schema Version

**Current Version:** `v1`

The `version` field in all DiagnosisReport responses indicates the schema version. This version is frozen for Phase 05 and represents a stable contract.

## Base URL

```
http://localhost:8080/api/v1
```

## Endpoints

### 1. Run Diagnosis (Synchronous)

Execute a diagnosis and receive an immediate, structured response.

**Endpoint:** `POST /diagnosis/run`

**Request Body:**

```json
{
  "target": "redis",
  "middleware": "redis",
  "instance": "redis-master-0",
  "filters": {
    "namespace": "default"
  }
}
```

**Request Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target` | string | Yes | Target middleware type (e.g., "redis", "mysql") |
| `middleware` | string | Yes | Middleware identifier (same as target) |
| `instance` | string | No | Instance name or connection string |
| `filters` | object | No | Additional filtering parameters |

**Response:** `200 OK`

```json
{
  "version": "v1",
  "id": "redis-master-0-1702823456",
  "timestamp": "2024-12-17T10:30:56Z",
  "target": {
    "middleware": "redis",
    "instance": "redis-master-0",
    "namespace": "default"
  },
  "status": "healthy",
  "summary": "Diagnosis completed for redis. Found 0 issues.",
  "issues": [],
  "metrics": {},
  "metadata": {}
}
```

**Response Schema: DiagnosisReport**

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Report schema version (always "v1") |
| `id` | string | Unique diagnosis session identifier |
| `timestamp` | string (ISO 8601) | When the diagnosis was completed |
| `target` | object | Information about what was diagnosed |
| `target.middleware` | string | Middleware type |
| `target.instance` | string | Instance name |
| `target.namespace` | string | Kubernetes namespace (if applicable) |
| `status` | string | Overall health status: "healthy", "warning", "critical" |
| `summary` | string | High-level summary of findings |
| `issues` | array | List of identified issues (see Issue schema) |
| `metrics` | object | Key diagnostic metrics |
| `metadata` | object | Additional contextual information |

**Issue Schema:**

```json
{
  "id": "issue-001",
  "source": "ai-analyzer",
  "title": "High Memory Usage",
  "severity": "warning",
  "description": "Memory usage is at 85%",
  "evidence": [
    {
      "type": "metric",
      "key": "memory_usage_percent",
      "value": 85,
      "context": "Current memory utilization"
    }
  ],
  "suggestions": [
    {
      "id": "sug-001",
      "description": "Increase memory allocation",
      "priority": "high",
      "fixHint": {
        "canAutoFix": true,
        "command": "kubectl scale",
        "parameters": {
          "replicas": "3"
        },
        "riskLevel": "medium"
      }
    }
  ],
  "category": "performance"
}
```

**Error Response:** `400 Bad Request` / `500 Internal Server Error`

```json
{
  "error": "Invalid middleware type"
}
```

### 2. Trigger Diagnosis (Asynchronous)

Start a diagnosis job that runs in the background. Receive progress updates via WebSocket.

**Endpoint:** `POST /diagnosis/trigger`

**Request/Response:** Same structure as synchronous endpoint, but returns immediately with a task ID.

**Response:** `202 Accepted`

```json
{
  "message": "Diagnosis started",
  "task_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 3. Get Diagnosis Result

Retrieve the result of a previously completed diagnosis.

**Endpoint:** `GET /diagnosis/result/:id`

**Response:** Same as the synchronous diagnosis response (DiagnosisReport).

## CLI Usage

The CLI automatically uses the standardized report format.

### JSON Output

```bash
ksa diagnose redis --instance redis-master-0 --output json
```

**Output:** Structured JSON matching the DiagnosisReport schema (v1).

### Text Output (Default)

```bash
ksa diagnose redis --instance redis-master-0
```

**Output:** Human-readable text formatted from the DiagnosisReport.

## Integration Examples

### cURL

```bash
curl -X POST http://localhost:8080/api/v1/diagnosis/run \
  -H "Content-Type: application/json" \
  -d '{
    "target": "redis",
    "middleware": "redis",
    "instance": "redis-master-0"
  }'
```

### Python

```python
import requests

response = requests.post(
    "http://localhost:8080/api/v1/diagnosis/run",
    json={
        "target": "redis",
        "middleware": "redis",
        "instance": "redis-master-0"
    }
)

report = response.json()
print(f"Report Version: {report['version']}")
print(f"Status: {report['status']}")
print(f"Issues Found: {len(report['issues'])}")
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type DiagnosisRequest struct {
    Target     string `json:"target"`
    Middleware string `json:"middleware"`
    Instance   string `json:"instance"`
}

type DiagnosisReport struct {
    Version   string                 `json:"version"`
    ID        string                 `json:"id"`
    Status    string                 `json:"status"`
    Summary   string                 `json:"summary"`
    Issues    []interface{}          `json:"issues"`
    Metrics   map[string]interface{} `json:"metrics"`
    Metadata  map[string]interface{} `json:"metadata"`
}

func main() {
    req := DiagnosisRequest{
        Target:     "redis",
        Middleware: "redis",
        Instance:   "redis-master-0",
    }
    
    body, _ := json.Marshal(req)
    resp, _ := http.Post(
        "http://localhost:8080/api/v1/diagnosis/run",
        "application/json",
        bytes.NewBuffer(body),
    )
    
    var report DiagnosisReport
    json.NewDecoder(resp.Body).Decode(&report)
    
    fmt.Printf("Report Version: %s\n", report.Version)
    fmt.Printf("Status: %s\n", report.Status)
}
```

## Version Evolution

Future schema changes will increment the version (e.g., "v2"). The API will support multiple versions simultaneously during transition periods.

### Backward Compatibility Guarantees

For v1:
- Field names and types will not change
- New optional fields may be added
- Existing fields will not be removed
- Enum values will not change meaning

## Status Codes

| Code | Description |
|------|-------------|
| 200 | Diagnosis completed successfully |
| 202 | Diagnosis accepted and running asynchronously |
| 400 | Invalid request (bad middleware type, missing required fields) |
| 404 | Diagnosis result not found (for GET by ID) |
| 500 | Internal server error during diagnosis |

## Rate Limiting

Currently, no rate limiting is enforced. In production deployments, consider implementing rate limits based on your infrastructure capacity.

## Authentication

The current implementation does not enforce authentication. For production use, integrate with your organization's authentication system.

## Support

For issues or questions:
- GitHub Issues: https://github.com/kubestack-ai/kubestack-ai/issues
- Documentation: https://github.com/kubestack-ai/kubestack-ai/tree/master/docs
