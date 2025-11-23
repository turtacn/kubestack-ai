# Async Task Scheduling and Notification

## Overview

KubeStack-AI supports asynchronous execution of long-running diagnosis tasks. This prevents request timeouts and allows users to check status or receive notifications upon completion.

## Architecture

The system consists of the following components:

1.  **Scheduler**: Receives diagnosis requests, creates a Task record, and submits it to the Task Queue.
2.  **Task Queue**: A durable queue (Redis) that holds pending tasks.
3.  **Worker**: A background process that consumes tasks from the queue and executes the diagnosis logic.
4.  **Task Store**: Persists task status (`PENDING`, `RUNNING`, `COMPLETED`, `FAILED`) and results.
5.  **Notifier**: Sends notifications (Webhook, Email) when a task is completed.

## Configuration

### Task Queue

Configure the queue in `configs/task/queue_config.yaml`:

```yaml
type: redis
redis:
  addr: "localhost:6379"
  queue_name: "diagnosis_tasks"
```

### Notification

Configure notifications in `configs/notification/notification_config.yaml`:

```yaml
webhook:
  url: "https://my-webhook.com/endpoint"
email:
  host: "smtp.example.com"
  port: 587
  username: "user"
  password: "password"
  from: "kubestack-ai@example.com"
```

## API Usage

### Submit Async Task

**POST** `/console/diagnose?async=true`

Request Body:
```json
{
  "targetMiddleware": "redis",
  "instance": "my-redis-instance"
}
```

Response:
```json
{
  "task_id": "task-uuid-1234",
  "status": "PENDING",
  "message": "Diagnosis task submitted successfully"
}
```

### Check Status

**GET** `/console/task/status/:taskId`

Response (Pending/Running):
```json
{
  "task_id": "task-uuid-1234",
  "state": "RUNNING",
  "created_at": "2023-10-27T10:00:00Z"
}
```

Response (Completed):
```json
{
  "task_id": "task-uuid-1234",
  "state": "COMPLETED",
  "result": {
      "summary": "Redis is healthy",
      "issues": []
  }
}
```
