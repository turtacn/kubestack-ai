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

### Quick Start

```bash
# Initialize KubeStack-AI
ksa init

# Diagnose all middleware in current namespace
ksa diagnose --all

# Ask natural language questions
ksa ask "Why is my Redis cluster slow?"

# Get specific middleware status
ksa status redis --namespace production

# List available plugins
ksa plugin list

# Install a new plugin
ksa plugin install mongodb
```

### Basic Usage Examples

#### Example 1: Comprehensive System Health Check

```bash
$ ksa diagnose --middleware redis,mysql,kafka
🔍 Analyzing Redis cluster...
✅ Redis: Healthy (3/3 nodes up, memory usage: 45%)

🔍 Analyzing MySQL primary-replica...
⚠️  MySQL: Warning detected
   • Replica lag: 2.3s (threshold: 1s)
   • Slow queries: 23 in last hour

🔍 Analyzing Kafka cluster...
❌ Kafka: Critical issues found
   • Topic 'orders': 50K messages backed up
   • Consumer group 'payment-service': 5min lag

💡 AI Recommendations:
   1. MySQL: Consider tuning innodb_buffer_pool_size
   2. Kafka: Scale consumer group or check processing logic
```

#### Example 2: Natural Language Troubleshooting

```bash
$ ksa ask "My application can't connect to the database"
🤔 Analyzing connection issues...

🔍 Discovered Issues:
   • PostgreSQL max_connections (100) reached
   • Connection pool exhaustion in app pods
   • Network policy blocking traffic on port 5432

🛠️  Suggested Actions:
   1. Increase max_connections: `ksa exec postgres --set max_connections=200`
   2. Scale app replicas: `ksa scale app --replicas 5`
   3. Review network policies: `ksa network analyze postgres`

Execute fixes? [y/N]: 
```

#### Example 3: Plugin Management

```bash
$ ksa plugin install clickhouse
📦 Installing ClickHouse plugin v1.2.0...
✅ Plugin installed successfully

$ ksa diagnose clickhouse --cluster analytics
🔍 ClickHouse Cluster Analysis:
   • Merge queue: 145 items (high)
   • Query latency P95: 2.3s
   • Disk usage: 78% on shard-2

💡 Recommendations:
   • Consider adding more background merge threads
   • Archive old partitions in 'events' table
```

## 📖 Documentation

* [**Architecture Overview**](docs/architecture.md) - Technical deep-dive into system design
* [**Plugin Development Guide**](docs/plugins.md) - Build your own middleware plugins
* [**Configuration Reference**](docs/configuration.md) - Complete config options
* [**Troubleshooting Guide**](docs/troubleshooting.md) - Common issues and solutions
* [**API Reference**](docs/api.md) - REST API and SDK documentation

## 🤝 Contributing

We welcome contributions from the community! KubeStack-AI is built by middleware experts for middleware experts.

### How to Contribute

1. **🐛 Report Issues**: Found a bug? [Open an issue](https://github.com/turtacn/kubestack-ai/issues)
2. **💡 Feature Requests**: Have ideas? [Start a discussion](https://github.com/turtacn/kubestack-ai/discussions)
3. **🔧 Code Contributions**: [Fork, develop, and submit PRs](docs/contributing.md)
4. **📝 Documentation**: Help improve our docs
5. **🧩 Plugin Development**: Build plugins for new middleware

### Development Setup

```bash
git clone https://github.com/turtacn/kubestack-ai.git
cd kubestack-ai
make dev-setup
make test
make build
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