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
    go build -o ksa ./cmd/ksa
    ```

## Configuration

1.  Copy the example configuration:
    ```bash
    cp configs/config.yaml.example configs/config.yaml
    ```
    *Note: If `config.yaml.example` is missing, you can start with a minimal config.*

2.  Set your OpenAI API Key:
    ```bash
    export KSA_LLM_OPENAI_API_KEY="your-api-key"
    ```

## Running Dependencies

Start ChromaDB (for RAG) and Redis (for Task Queue):

```bash
docker run -d --rm --name chromadb -p 8000:8000 -v "$(pwd)/chroma-data":/chroma -e IS_PERSISTENT=TRUE -e ANONYMIZED_TELEMETRY=FALSE chromadb/chroma
docker run -d --rm --name redis -p 6379:6379 redis:alpine
```

## Usage

### 1. Diagnose a Service

To diagnose a specific middleware instance (e.g., Redis):

```bash
./ksa diagnose redis --namespace default --instance my-redis
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
