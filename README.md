# KubeStack-AI

<p align="center">
  <img src="logo.png" alt="KubeStack-AI Logo" width="200" height="200">
</p>

<p align="center">
  <strong>AI-Powered Unified Middleware Management for Kubernetes & Beyond</strong>
</p>

<p align="center">
  <a href="https://github.com/turtacn/kubestack-ai/actions"><img src="https://img.shields.io/github/actions/workflow/status/turtacn/kubestack-ai/ci.yml?branch=main" alt="Build Status"></a>
  <a href="https://github.com/turtacn/kubestack-ai/blob/main/LICENSE"><img src="https://img.shields.io/github/license/turtacn/kubestack-ai" alt="License"></a>
  <a href="https://github.com/turtacn/kubestack-ai/releases"><img src="https://img.shields.io/github/v/release/turtacn/kubestack-ai" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/turtacn/kubestack-ai"><img src="https://goreportcard.com/badge/github.com/turtacn/kubestack-ai" alt="Go Report Card"></a>
  <a href="https://codecov.io/gh/turtacn/kubestack-ai"><img src="https://codecov.io/gh/turtacn/kubestack-ai/branch/main/graph/badge.svg" alt="Coverage"></a>
</p>

<p align="center">
  <a href="README-zh.md">中文文档</a> •
  <a href="docs/architecture.md">Architecture</a> •
  <a href="docs/contributing.md">Contributing</a> •
  <a href="docs/plugins.md">Plugin Development</a>
</p>

---

## 🚀 Mission Statement

KubeStack-AI is a revolutionary, AI-powered command-line assistant that transforms how you diagnose, manage, and optimize your entire middleware stack running on Kubernetes and bare-metal environments. By combining the power of Large Language Models with deep middleware expertise, KubeStack-AI provides intelligent, natural language-driven operations for complex cloud-native infrastructures.

<img src="demo2.gif" width="100%"/>

## 🎯 Why KubeStack-AI?

### The Challenge
Modern cloud-native environments involve dozens of middleware components (Redis, Kafka, PostgreSQL, MinIO, ElasticSearch, etc.), each with unique operational complexities. Traditional approaches require:

- **Fragmented Tools**: Different CLI tools for each middleware
- **Deep Expertise**: Extensive knowledge of each system's internals  
- **Manual Correlation**: Connecting symptoms across multiple systems
- **Time-Consuming Diagnosis**: Hours spent troubleshooting complex issues

### Our Solution
KubeStack-AI provides a **unified, AI-driven interface** that:

✅ **Speaks Your Language**: Natural language queries instead of complex commands  
✅ **Thinks Holistically**: Cross-middleware correlation and root cause analysis  
✅ **Acts Intelligently**: AI-powered diagnosis with actionable recommendations  
✅ **Extends Seamlessly**: Plugin architecture for any middleware  
✅ **Operates Safely**: Interactive confirmation for critical operations  

## ⭐ Key Features

### 🔍 **Intelligent Diagnosis**
- **Multi-Layer Analysis**: System, Kubernetes, and middleware-specific checks
- **AI-Powered RCA**: Root cause analysis across complex distributed systems
- **Natural Language Queries**: Ask questions in plain English

### 🛠️ **Universal Middleware Support**
- **Database Systems**: MySQL, PostgreSQL, MongoDB, Redis, ClickHouse
- **Message Queues**: Kafka, RabbitMQ, Pulsar
- **Search & Analytics**: ElasticSearch, OpenSearch
- **Storage**: MinIO, Ceph
- **Monitoring**: Prometheus, Grafana
- **Service Discovery**: etcd, Consul

### 🧩 **Plugin Architecture**
- **Extensible Design**: Add support for any middleware through plugins
- **Community Driven**: Open plugin ecosystem
- **Hot-swappable**: Install, update, and remove plugins without downtime

### 🤖 **AI-Enhanced Operations**
- **Smart Recommendations**: Context-aware optimization suggestions  
- **Automated Fixes**: One-click resolution for common issues
- **Knowledge Integration**: Built-in best practices and troubleshooting guides

## 🚀 Getting Started

### Prerequisites
- Go 1.18+
- Git
- `make` (for development)

### Installation

#### Option 1: Go Install
```bash
go install github.com/turtacn/kubestack-ai/cmd/ksa@latest
````

#### Option 2: Homebrew (macOS/Linux)

```bash
brew tap turtacn/kubestack-ai
brew install kubestack-ai
```

#### Option 3: Download Binary

Visit our [releases page](https://github.com/turtacn/kubestack-ai/releases) to download pre-built binaries.

### Configuration

KubeStack-AI is configured via a `config.yaml` file and environment variables.

1.  **Initialize Configuration**: Run `ksa init` to create a default configuration file at `~/.kubestack-ai/config.yaml`.

2.  **Set LLM API Key**: For AI-powered features, you must provide an API key. This is best done via an environment variable.

    ```bash
    # For OpenAI
    export KSA_LLM_OPENAI_APIKEY="your-openai-api-key"

    # For Google Gemini
    export KSA_LLM_GEMINI_APIKEY="your-gemini-api-key"
    ```

    You can also set this in your `~/.kubestack-ai/config.yaml`:

    ```yaml
    llm:
      provider: openai
      openai:
        apiKey: "your-openai-api-key"
    ```

### Quick Start

```bash
# Diagnose all middleware in current namespace
ksa diagnose --all

# Ask natural language questions
ksa ask "Why is my Redis cluster slow?"

# Get specific middleware status
ksa status redis --namespace production

# List available plugins
ksa plugin list
```

## 📖 Documentation

* [**Architecture Overview**](docs/architecture.md) - Technical deep-dive into system design
* [**Plugin Development Guide**](docs/plugins.md) - Build your own middleware plugins
* [**Configuration Reference**](docs/configuration.md) - Complete config options
* [**Troubleshooting Guide**](docs/troubleshooting.md) - Common issues and solutions
* [**API Reference**](docs/api.md) - REST API and SDK documentation

## 🏗️ Architecture & Codebase

KubeStack-AI is built on a modular, layered architecture designed for extensibility and maintainability. Understanding this structure is key to contributing effectively.

- **`/cmd/ksa`**: The main application entry point. This is where the Cobra CLI is initialized and the core `Orchestrator` is created and injected with its dependencies.

- **`/internal`**: Contains all core application logic, following standard Go project layout. This code is not meant for external import.
    - **`/internal/cli`**: Defines the command-line interface using Cobra. It handles command parsing, flags, and user interaction, but delegates all real work to the Orchestrator.
    - **`/internal/core`**: The heart of the application.
        - **`/interfaces`**: Defines the key Go interfaces (`Orchestrator`, `DiagnosisManager`, `MiddlewarePlugin`, etc.) that form the application's backbone.
        - **`/models`**: Contains the data structures (structs) passed between components (e.g., `DiagnosisResult`, `ExecutionPlan`).
        - **`/orchestrator`**: The central coordinator. It receives requests from the CLI and uses its managers to fulfill them.
        - **`/diagnosis`**: The diagnosis engine. It uses analyzers (rule-based, AI-based) to inspect data and find issues.
        - **`/execution`**: The execution engine. It takes a plan and safely executes actions to fix issues.
    - **`/internal/plugins`**: The plugin system.
        - **`/manager`**: Responsible for loading, registering, and managing plugin lifecycles.
        - **`/builtin`**: Contains the source code for all standard, built-in middleware plugins (e.g., `redis`, `kafka`).
    - **`/internal/llm`**: Abstractions for interacting with Large Language Models.
        - **`/client`**: Contains specific clients for different providers (OpenAI, Gemini).
        - **`/rag`**: The Retrieval-Augmented Generation (RAG) pipeline, which enriches prompts with knowledge base context.
    - **`/internal/knowledge`**: The knowledge base system. It includes components for crawling data sources, storing them (in vector and document stores), and searching.
    - **`/internal/common`**: Shared utilities used across the application, such as structured logging, configuration management, and custom error types.

- **`/pkg`**: Shared utility packages that are safe for external use (though none are intended for it currently).
- **`/docs`**: All project documentation, including this README and the detailed architecture diagrams.
- **`/scripts`**: Helper scripts for development (build, test, lint).

## 🤝 Contributing

We welcome contributions from the community! KubeStack-AI is built by middleware experts for middleware experts.

### Development Setup

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/turtacn/kubestack-ai.git
    cd kubestack-ai
    ```

2.  **Install dependencies:**
    This command will download the necessary Go modules and development tools.
    ```bash
    make dev-setup
    ```

3.  **Build the binary:**
    ```bash
    make build
    ```
    The `ksa` executable will be placed in the `/bin` directory.

4.  **Run tests and linters:**
    Before submitting a change, ensure all tests and quality checks pass.
    ```bash
    make test
    make lint
    ```

5.  **Run your local build:**
    To test your changes, run the binary from the `/bin` directory.
    ```bash
    # Make sure you have configured your API keys as mentioned in the Configuration section
    ./bin/ksa --config /path/to/your/config.yaml ask "Why is my redis slow?"
    ```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 🏆 Community & Support

* **💬 Discussions**: [GitHub Discussions](https://github.com/turtacn/kubestack-ai/discussions)
* **🐛 Issues**: [GitHub Issues](https://github.com/turtacn/kubestack-ai/issues)
* **📧 Email**: [kubestack-ai@turtacn.com](mailto:kubestack-ai@turtacn.com)
* **🐦 Twitter**: [@KubeStackAI](https://twitter.com/KubeStackAI)

## 📜 License

KubeStack-AI is licensed under the [Apache License 2.0](LICENSE). See [LICENSE](LICENSE) file for details.

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=turtacn/kubestack-ai\&type=Date)](https://star-history.com/#turtacn/kubestack-ai&Date)

---

<p align="center">
  Built with ❤️ by the KubeStack-AI community
</p>