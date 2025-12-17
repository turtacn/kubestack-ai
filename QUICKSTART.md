# Quickstart Guide

This guide will help you get started with KubeStack-AI quickly.

## Prerequisites

*   Go 1.21+
*   Docker (for running dependencies like ChromaDB, Redis)
*   An OpenAI API key (or compatible LLM provider)

## Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/kubestack-ai/kubestack-ai.git
    cd kubestack-ai
    ```

2.  Build the project:
    ```bash
    go mod tidy
    go build -o ksa ./cmd/ksa/main.go
    ```

## Configuration

1.  Copy the example configuration:
    ```bash
    mkdir -p ~/.kubestack-ai
    cp configs/config.yaml.example ~/.kubestack-ai/config.yaml
    # OR set up in current directory
    cp configs/config.yaml.example config.yaml
    ```

2.  Set your OpenAI API Key:
    ```bash
    export KSA_LLM_OPENAI_API_KEY="your-api-key"
    ```

## Running Dependencies

Start ChromaDB (for RAG) and Redis (for Task Queue):

```bash
# Start Redis
docker run -d --rm --name redis -p 6379:6379 redis:alpine

# Start ChromaDB
mkdir -p chroma-data
docker run -d --rm --name chromadb -p 8000:8000 -v "$(pwd)/chroma-data":/chroma -e IS_PERSISTENT=TRUE -e ANONYMIZED_TELEMETRY=FALSE chromadb/chroma
```

## Running the Server (Web Console & Alerts)

KubeStack-AI includes a web console and an alert receiver.

1.  Start the server:
    ```bash
    ./ksa server start
    ```

2.  Open your browser and navigate to:
    [http://localhost:8080](http://localhost:8080)

3.  **Manual Diagnosis**:
    *   **Middleware**: Select the type (e.g., Redis).
    *   **Target Host**: e.g., `localhost:6379`.
    *   **Instance Name**: e.g., `my-redis`.
    *   Click "Start Diagnosis" and watch the real-time logs.

4.  **Alert Webhook Integration**:
    *   Configure Alertmanager to send webhooks to: `http://<server-ip>:8080/api/v1/webhook/alertmanager`
    *   Configure Grafana to send webhooks to: `http://<server-ip>:8080/api/v1/webhook/grafana`
    *   When an alert fires, KubeStack-AI will automatically trigger a diagnosis and notify configured channels (Slack, DingTalk).

## CLI Usage

### 1. Diagnose a Service

To diagnose a specific middleware instance via CLI:

```bash
# Redis - Text output (default)
./ksa diagnose redis --namespace default --instance my-redis

# Redis - JSON output (structured, version v1)
./ksa diagnose redis --namespace default --instance my-redis --output json

# PostgreSQL
export KSA_POSTGRES_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable"
./ksa diagnose postgresql --instance my-postgres

# MySQL with JSON output
./ksa diagnose mysql --instance mysql-primary --output json
```

**Output Formats:**
- `--output text` (default): Human-readable text format
- `--output json`: Structured JSON format (DiagnosisReport v1 schema)

**JSON Output Example:**
```json
{
  "version": "v1",
  "id": "redis-my-redis-1702823456",
  "timestamp": "2024-12-17T10:30:56Z",
  "target": {
    "middleware": "redis",
    "instance": "my-redis",
    "namespace": "default"
  },
  "status": "healthy",
  "summary": "Diagnosis completed for redis. Found 0 issues.",
  "issues": [],
  "metrics": {},
  "metadata": {}
}
```

### 2. Search Knowledge Base

To search for solutions in the knowledge base:

```bash
./ksa kb search "Redis OOM"
```

## Running Tests

To run all tests:

```bash
make test
```
