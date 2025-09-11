# KubeStack-AI REST API Reference

This document provides a reference for the KubeStack-AI REST API. The API allows for programmatic interaction with the diagnosis, analysis, and knowledge base features of the platform, enabling integration with other systems and custom tooling.

**Note:** This API is currently a design specification. Endpoints and schemas are subject to change during implementation.

## 1. Authentication

All API requests must be authenticated. Authentication is performed by providing a bearer token (API key) in the `Authorization` header.

`Authorization: Bearer <YOUR_API_KEY>`

API keys can be generated and managed through the application's configuration or a future admin interface.

## 2. Versioning

The API is versioned to ensure backward compatibility. The current version is `v1`. All endpoint paths are prefixed with `/api/v1`.

Example: `https://ksa.example.com/api/v1/diagnose`

## 3. Endpoints

### Diagnosis

#### `POST /api/v1/diagnose`

Triggers a new, asynchronous diagnosis on a specified middleware instance.

**Request Body:** (`application/json`)

```json
{
  "targetMiddleware": "redis",
  "instance": "my-redis-instance",
  "namespace": "default",
  "options": {
    "skipCache": true
  }
}
```

**Response Body (202 Accepted):**

The API immediately returns a diagnosis job object with a unique ID. The client can use this ID to poll for the result.

```json
{
  "jobId": "diag-job-a1b2c3d4",
  "status": "Pending",
  "createdAt": "2025-03-15T12:00:00Z"
}
```

#### `GET /api/v1/diagnose/results/{jobId}`

Retrieves the result of a diagnosis job.

**Response Body (200 OK):**

Returns a `DiagnosisResult` object once the job is complete.

```json
{
  "id": "diag-job-a1b2c3d4",
  "timestamp": "2025-03-15T12:05:00Z",
  "status": "Warning",
  "summary": "Redis diagnosis complete. Found 2 potential issues.",
  "issues": [
    {
      "title": "High Memory Fragmentation",
      "severity": "Warning",
      "description": "...",
      "recommendations": [{...}]
    }
  ]
}
```

### Natural Language Query

#### `POST /api/v1/ask`

Asks a question to the AI assistant. This endpoint supports streaming responses for real-time interaction.

**Request Body:** (`application/json`)

```json
{
  "question": "Why is my Redis memory fragmentation high?",
  "context": {
    "diagnosisId": "diag-job-a1b2c3d4"
  }
}
```

**Response Body (200 OK with `Content-Type: text/event-stream`):**

The response is a stream of Server-Sent Events (SSE). Each event contains a chunk of the AI's response.

```
data: {"content": "High memory fragmentation in Redis can be caused by..."}

data: {"content": " a pattern of frequent allocations and deallocations of keys with varying sizes."}

data: {"content": " Based on your diagnosis report, the fragmentation ratio is 1.8. You can often resolve this by restarting the Redis instance."}

event: done
data: {}
```

### Plugins

#### `GET /api/v1/plugins`

Lists all available plugins discovered by the registry.

**Response Body (200 OK):**

```json
{
  "plugins": [
    {
      "name": "redis",
      "version": "0.1.0",
      "description": "Provides diagnostics for Redis instances.",
      "supportedVersions": ["5.x", "6.x", "7.x"]
    },
    {
      "name": "mysql",
      "version": "0.1.0",
      "description": "Provides diagnostics for MySQL and compatible databases.",
      "supportedVersions": ["5.7", "8.0"]
    }
  ]
}
```

## 4. Error Handling

API errors are returned in a standard JSON format, consistent with the error types defined in `internal/common/types/errors`.

**Example Error Response (404 Not Found):**

```json
{
  "error": {
    "code": 1001,
    "type": "PluginError",
    "message": "Plugin 'postgres' not found in registry",
    "suggestion": "Ensure the plugin is installed correctly in the plugins directory or run 'ksa plugin list' to see available plugins."
  }
}
```

<!-- Personal.AI order the ending -->
