# KubeStack-AI 🚀

<div align="center">
  <img src="logo.png" alt="KubeStack-AI Logo" width="200" height="200">

**AI-Powered Kubernetes Middleware Management Platform**
 
  [![Build Status](https://github.com/turtacn/kubestack-ai/workflows/CI/badge.svg)](https://github.com/turtacn/kubestack-ai/actions)
  [![Go Report Card](https://goreportcard.com/badge/github.com/turtacn/kubestack-ai)](https://goreportcard.com/report/github.com/turtacn/kubestack-ai)
  [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
  [![GitHub release](https://img.shields.io/github/release/turtacn/kubestack-ai.svg)](https://github.com/turtacn/kubestack-ai/releases)
  [![Go Version](https://img.shields.io/badge/Go-1.20+-blue.svg)](https://golang.org)
</div>

> *An AI-powered unified command-line assistant for diagnosing, managing, and optimizing middleware stacks running on Kubernetes and bare-metal environments.*

📚 [中文版本 README](./README-zh.md)

## 🚀 Mission Statement

KubeStack-AI revolutionizes cloud-native middleware operations by providing a unified, intelligent CLI interface that moves beyond single-tool assistants (kubectl-ai, minio-cli-ai) to deliver holistic management across your entire middleware ecosystem. Drawing inspiration from k8sgpt's diagnostic capabilities and kubectl-ai's intelligent interface design, KubeStack-AI extends these concepts to support comprehensive middleware stack management with AI-powered insights.

## 🎯 Why KubeStack-AI?

### Pain Points We Solve

| Traditional Challenge | KubeStack-AI Solution |
|----------------------|----------------------|
| **Fragmented Tools** | Unified CLI for all middleware (Redis, Kafka, PostgreSQL, MinIO, etc.) |
| **Complex Troubleshooting** | AI-powered root cause analysis with natural language explanations |
| **Manual Optimization** | Intelligent performance tuning recommendations |
| **Knowledge Silos** | Built-in expertise for 10+ middleware technologies |
| **Reactive Operations** | Proactive health monitoring and predictive insights |

### Core Advantages

- 🧠 **AI-First Design**: Natural language interface for complex operations
- 🔌 **Plugin Architecture**: Extensible support for any middleware
- 🎯 **Unified Experience**: One tool, entire stack management
- 📊 **Intelligent Analysis**: Root cause analysis with actionable recommendations
- 🔐 **Security-Aware**: Permission analysis and security best practices
- 🚀 **Production-Ready**: Built for enterprise-scale environments

## ✨ Key Features

### 🔍 Automated Diagnosis
- **Universal Health Checks**: Container status, resource utilization, storage health
- **Middleware-Specific Analysis**: Custom diagnostics for MySQL, Redis, Kafka, ElasticSearch, RabbitMQ, PostgreSQL, MongoDB, ClickHouse, etcd, Prometheus, MinIO
- **Intelligent Log Analysis**: AI-powered pattern recognition and anomaly detection

### 🎯 Smart Operations
- **Natural Language Interface**: `kubestack-ai "Why is my Redis cluster slow?"`
- **Auto-Repair Suggestions**: Generate and execute fix commands with confirmation
- **Performance Optimization**: Real-time tuning recommendations
- **Security Auditing**: Permission analysis and compliance checking

### 🔧 Plugin Ecosystem
- **Hot-Pluggable Modules**: Add new middleware support without recompilation
- **Community Driven**: Extensible plugin marketplace
- **Custom Integrations**: Build plugins for proprietary systems

## 🚀 Getting Started

### Installation

#### Using Go Install
```bash
go install github.com/turtacn/kubestack-ai/cmd/kubestack-ai@latest
````

#### Using Homebrew (macOS/Linux)

```bash
brew tap turtacn/kubestack-ai
brew install kubestack-ai
```

#### Using Binary Releases

Download the latest release from [GitHub Releases](https://github.com/turtacn/kubestack-ai/releases)

### Quick Start

1. **Initialize Configuration**

```bash
kubestack-ai init
```

2. **Configure AI Provider**

```bash
kubestack-ai config set ai.provider openai
kubestack-ai config set ai.api_key YOUR_API_KEY
```

3. **Install Plugins**

```bash
# Install core middleware plugins
kubestack-ai plugin install redis mysql kafka elasticsearch

# List available plugins
kubestack-ai plugin list --available
```

### Basic Usage Examples

#### Natural Language Diagnostics

```bash
# Analyze overall cluster health
kubestack-ai analyze "Check my Kubernetes cluster health"

# Middleware-specific diagnosis
kubestack-ai diagnose redis "Why is my Redis cluster experiencing high latency?"

# Performance optimization
kubestack-ai optimize mysql "My MySQL queries are running slowly"
```

#### Structured Commands

```bash
# Comprehensive health check
kubestack-ai health check --middleware redis,mysql,kafka --namespace production

# Generate performance report
kubestack-ai report --type performance --output json --timerange 1h

# Security audit
kubestack-ai security audit --middleware all --severity high
```

#### Interactive Mode

```bash
# Start interactive session
kubestack-ai interactive

# Example interactive session:
> analyze redis performance in namespace production
> show me the slow queries in mysql
> optimize kafka topic configuration for high throughput
```

### Sample Output

```bash
$ kubestack-ai diagnose redis "Redis is using too much memory"

🔍 KubeStack-AI Analysis Report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 REDIS DIAGNOSIS - production/redis-cluster
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🟡 WARNINGS DETECTED (2)
├── Memory Usage: 89% (7.1GB / 8GB allocated)
├── Eviction Policy: allkeys-lru may cause data loss
└── Background Save: Last save failed 2h ago

🔍 ROOT CAUSE ANALYSIS
The primary issue appears to be memory pressure combined with 
failed background saves, leading to potential data loss scenarios.

💡 RECOMMENDATIONS
1. Increase memory limit from 8GB to 12GB
   ├── Command: kubectl patch deployment redis -p '{"spec":{"template":{"spec":{"containers":[{"name":"redis","resources":{"limits":{"memory":"12Gi"}}}]}}}}'
   └── Impact: Resolves immediate memory pressure

2. Fix backup configuration
   ├── Issue: Redis backup directory lacks write permissions
   ├── Command: kubectl exec redis-0 -- chmod 755 /data/backup
   └── Verify: Check backup job status after fix

3. Optimize memory usage
   ├── Enable compression: CONFIG SET rdbcompression yes
   ├── Adjust save intervals: CONFIG SET save "900 1 300 10"
   └── Expected reduction: ~20% memory usage

🚀 AUTO-FIX AVAILABLE
Run 'kubestack-ai fix redis --apply-recommendations' to execute suggested fixes
(Confirmation required for destructive operations)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏱  Analysis completed in 3.2s | 🤖 Powered by GPT-4
```

## 📖 Documentation

* [Architecture Overview](./docs/architecture.md)
* [Plugin Development Guide](./docs/plugins.md)
* [Configuration Reference](./docs/configuration.md)
* [API Documentation](./docs/api.md)
* [Contributing Guidelines](./CONTRIBUTING.md)

## 🤝 Contributing

We welcome contributions from the community! KubeStack-AI thrives on the collective expertise of DevOps engineers, SREs, and middleware specialists.

### Ways to Contribute

* 🐛 **Bug Reports**: Help us identify issues
* 💡 **Feature Requests**: Suggest new capabilities
* 🔌 **Plugin Development**: Add support for new middleware
* 📚 **Documentation**: Improve guides and examples
* 🧪 **Testing**: Help us maintain quality
* 🌐 **Translation**: Localize for global adoption

### Quick Start for Contributors

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run tests: `make test`
5. Submit a pull request

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines.

## 🛣 Roadmap

* [ ] **step 1**: Core Plugin Ecosystem (Redis, MySQL, PostgreSQL)
* [ ] **step 2**: Kafka, ElasticSearch, RabbitMQ Support
* [ ] **step 3**: Advanced AI Features (Predictive Analysis, Auto-Scaling)
* [ ] **step 4**: Enterprise Features (RBAC, Audit Logging, Custom AI Models)

## 📄 License

KubeStack-AI is licensed under the Apache License 2.0. See [LICENSE](./LICENSE) for details.

## 🙏 Acknowledgments

* Inspired by the excellent work of [k8sgpt-ai](https://github.com/k8sgpt-ai/k8sgpt) and [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai)
* Built with ❤️ by the cloud-native community
* Special thanks to all contributors and beta testers

---

<div align="center">
  <p><strong>⭐ Star us on GitHub if KubeStack-AI helps your DevOps journey!</strong></p>
  <p>For questions, join our <a href="https://discord.gg/kubestack-ai">Discord Community</a> or open an <a href="https://github.com/turtacn/kubestack-ai/issues">issue</a>.</p>
</div>