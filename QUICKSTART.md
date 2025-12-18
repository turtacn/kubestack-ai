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

KubeStack-AI provides a powerful command-line interface for diagnosing, analyzing, and fixing middleware issues.

### CLI Validation

Before using the CLI, you can verify that all commands are working properly:

```bash
# Run smoke tests to verify CLI functionality
./scripts/cli_smoke_test.sh

# Check version
./ksa version

# Get help for any command
./ksa --help
./ksa diagnose --help
```

### 1. Diagnose a Service

To diagnose a specific middleware instance via CLI:

```bash
# Redis - Text output (default)
./ksa diagnose redis --instance localhost:6379

# Redis - JSON output (structured, version v1)
./ksa diagnose redis --instance localhost:6379 --output json

# Redis with namespace (Kubernetes)
./ksa diagnose redis --instance my-redis --namespace default

# PostgreSQL
./ksa diagnose postgresql --instance localhost:5432

# MySQL with JSON output
./ksa diagnose mysql --instance localhost:3306 --output json

# Kafka
./ksa diagnose kafka --instance localhost:9092

# Elasticsearch  
./ksa diagnose elasticsearch --instance localhost:9200
```

**Output Formats:**
- `--output text` (default): Human-readable text format
- `--output json`: Structured JSON format (DiagnosisReport v1 schema)
- `--output yaml`: YAML format for easy reading and editing

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

### 2. Ask Questions

Ask natural language questions about middleware:

```bash
# Ask a general question
./ksa ask "What causes high CPU usage in Redis?"

# Ask with specific context
./ksa ask "How to optimize MySQL query performance?" --context "MySQL 8.0, InnoDB engine"

# Get structured output
./ksa ask "Explain Kafka replication" --output json
```

### 3. Apply Fixes

After diagnosing issues, you can apply recommended fixes:

```bash
# Get diagnosis ID from previous diagnose command
DIAG_ID=$(./ksa diagnose redis --instance localhost:6379 -o json | jq -r '.id')

# Review and apply fixes interactively
./ksa fix --id $DIAG_ID

# Apply fixes automatically (use with caution!)
./ksa fix --id $DIAG_ID --auto-approve

# Dry run to see what would be changed
./ksa fix --id $DIAG_ID --dry-run
```

### 4. Search Knowledge Base

Search the built-in knowledge base for solutions and best practices:

```bash
# Search for Redis memory issues
./ksa kb search "Redis OOM"

# Search with severity filter
./ksa kb search "memory" --severity critical

# Search with middleware filter
./ksa kb search "performance" --middleware redis

# Get detailed entry
./ksa kb get kb-redis-001

# JSON output
./ksa kb search "optimization" -o json

# Update knowledge base
./ksa kb update
```

### 5. Manage Plugins

View and manage diagnostic plugins:

```bash
# List all plugins
./ksa plugin list

# Get plugin details
./ksa plugin info redis-diagnostics

# JSON output
./ksa plugin list -o json

# Enable a plugin
./ksa plugin enable redis-diagnostics

# Disable a plugin
./ksa plugin disable redis-diagnostics
```

### 6. Supported Middleware

Currently supported middleware types:
- **Redis** - In-memory data store
- **MySQL** - Relational database
- **Kafka** - Event streaming platform
- **Elasticsearch** - Search and analytics engine
- **PostgreSQL** - Object-relational database

Coming soon:
- MongoDB, RabbitMQ, MinIO, Prometheus, ClickHouse

### 7. Complete CLI Reference

For detailed information about all commands, flags, and options, see:
- [CLI Command Reference](docs/round6/phase26/cli-command-reference.md)

## Running Tests

Run the complete test suite to verify your installation:

```bash
# Run all unit tests
make test

# Run E2E tests
make e2e-test

# Run CLI smoke tests
./scripts/cli_smoke_test.sh

# Check test coverage
go test -cover ./...
```
