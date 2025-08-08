# KubeStack-AI 🚀

<div align="center">
  <img src="logo.png" alt="KubeStack-AI Logo" width="200" height="200">

**AI驱动的Kubernetes中间件管理平台**

[![构建状态](https://img.shields.io/github/workflow/status/turtacn/kubestack-ai/CI)](https://github.com/turtacn/kubestack-ai/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/turtacn/kubestack-ai)](https://goreportcard.com/report/github.com/turtacn/kubestack-ai)
[![许可证](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![发布版本](https://img.shields.io/github/release/turtacn/kubestack-ai.svg)](https://github.com/turtacn/kubestack-ai/releases)
[![Go 版本](https://img.shields.io/github/go-mod/go-version/turtacn/kubestack-ai)](https://golang.org/)

</div>

## 项目使命

KubeStack-AI 是一款革命性的命令行助手，将人工智能引入 Kubernetes 中间件管理。无需在 Redis、Kafka、PostgreSQL、MinIO 等多个工具之间切换，KubeStack-AI 提供统一的智能接口，理解您用自然语言描述的问题并提供可执行的解决方案。

## 为什么选择 KubeStack-AI？ 🎯

**问题现状：** 在 Kubernetes 中管理复杂的中间件栈需要跨多个领域、工具和故障排除方法的专业知识。团队经常面临以下困扰：

* 在不同的 CLI 工具之间切换上下文
* 关联 Kubernetes 层和应用层的问题
* 手动诊断复杂的多服务问题
* 团队成员间故障排除方法不一致

**解决方案：** KubeStack-AI 通过 AI 驱动的智能统一中间件管理：

* **单一接口**：满足所有中间件需求的一个工具
* **自然语言**：像与同事交谈一样描述问题
* **智能上下文**：自动收集 K8s 和服务数据
* **可执行见解**：不仅检测问题，还建议修复方案
* **可扩展架构**：基于插件的系统支持任何中间件

## 主要功能特性 ⭐

* 🤖 **AI驱动诊断**：自然语言问题描述与智能分析
* 🔌 **插件架构**：可扩展支持 Redis、Kafka、PostgreSQL、MinIO 等
* 🎯 **统一接口**：单一 CLI 管理整个中间件栈
* 📊 **自动上下文收集**：收集 K8s 指标、日志和服务特定数据
* 🔧 **智能修复建议**：AI 分析的解决方案与可选自动执行
* 📈 **性能优化**：主动的资源利用优化建议
* 🔒 **企业就绪**：RBAC 支持、审计日志和安全优先设计

## 快速开始 🚀

### 安装方式

#### 使用 Go Install

```bash
go install github.com/turtacn/kubestack-ai/cmd/ksai@latest
```

#### 使用 Homebrew (macOS/Linux)

```bash
brew install turtacn/tap/kubestack-ai
```

#### 下载二进制文件

从 [GitHub Releases](https://github.com/turtacn/kubestack-ai/releases) 下载最新版本

### 快速入门

1. **配置您的环境**：

```bash
# 配置您的 LLM 提供商（OpenAI、Claude 等）
ksai config set llm.provider openai
ksai config set llm.api-key YOUR_API_KEY

# 验证 Kubernetes 连接
ksai config verify
```

2. **基本用法示例**：

```bash
# 自然语言诊断
ksai diagnose "我的 redis pods 频繁重启"

# 服务特定分析
ksai analyze redis --namespace production

# 交互式修复并确认
ksai repair kafka --issue "消费者延迟增加" --confirm

# 所有中间件健康检查
ksai health-check --all-services
```

3. **高级用法**：

```bash
# 多服务关联分析
ksai diagnose "postgres 查询缓慢且 redis 缓存命中率低"

# 自动化监控设置
ksai setup monitoring --services redis,kafka,postgres

# 性能优化建议
ksai optimize --namespace production --service-type database
```

## 支持的中间件 🛠️

| 服务            | 状态     | 插件版本   |
| ------------- | ------ | ------ |
| Redis         | ✅ 稳定   | v1.0.0 |
| Kafka         | ✅ 稳定   | v1.0.0 |
| PostgreSQL    | ✅ 稳定   | v1.0.0 |
| MinIO         | 🚧 测试版 | v0.9.0 |
| MongoDB       | 📋 已规划 | -      |
| Elasticsearch | 📋 已规划 | -      |

## 贡献指南 🤝

我们欢迎社区贡献！KubeStack-AI 设计为可扩展性，我们很兴奋看到新的插件和功能。

### 贡献方式

* 🐛 **错误报告**：通过报告问题帮助我们改进
* 💡 **功能请求**：建议新功能或中间件支持
* 🔌 **插件开发**：为新的中间件服务创建插件
* 📚 **文档**：改进文档、教程和示例
* 🧪 **测试**：帮助我们在不同环境中测试

### 开发入门

```bash
git clone https://github.com/turtacn/kubestack-ai.git
cd kubestack-ai
make setup
make test
```

阅读我们的[贡献指南](CONTRIBUTING.md)获取详细信息。

## 文档资料 📖

* [架构概述](docs/architecture.md)
* [插件开发指南](docs/plugins/developing-plugins.md)
* [用户指南](docs/user-guide/getting-started.md)
* [API 参考](docs/api/plugin-api.md)
* [示例与教程](docs/examples/)

## 社区与支持 💬

* **GitHub Issues**：[报告错误或请求功能](https://github.com/turtacn/kubestack-ai/issues)
* **讨论**：[社区讨论与问答](https://github.com/turtacn/kubestack-ai/discussions)
* **Discord**：[加入我们的社区服务器](https://discord.gg/kubestack-ai)（即将推出）

## 许可证 📄

本项目采用 Apache License 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 致谢 🙏

* Kubernetes 社区提供的卓越生态系统
* 所有中间件项目维护者构建的出色工具
* 让这个项目成为可能的所有贡献者

---

<div align="center">
  由 KubeStack-AI 社区用 ❤️ 制作

[官网](https://kubestack-ai.dev) • [文档](https://docs.kubestack-ai.dev) • [社区](https://github.com/turtacn/kubestack-ai/discussions)

</div>