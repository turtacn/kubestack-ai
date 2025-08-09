# KubeStack-AI 🚀

<div align="center">
  <img src="logo.png" alt="KubeStack-AI Logo" width="200" height="200">

**AI驱动的Kubernetes中间件管理平台**

 
  [![构建状态](https://github.com/turtacn/kubestack-ai/workflows/CI/badge.svg)](https://github.com/turtacn/kubestack-ai/actions)
  [![Go Report Card](https://goreportcard.com/badge/github.com/turtacn/kubestack-ai)](https://goreportcard.com/report/github.com/turtacn/kubestack-ai)
  [![许可证](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
  [![GitHub 发布](https://img.shields.io/github/release/turtacn/kubestack-ai.svg)](https://github.com/turtacn/kubestack-ai/releases)
  [![Go 版本](https://img.shields.io/badge/Go-1.20+-blue.svg)](https://golang.org)
</div>

> *基于 AI 的统一命令行助手，用于诊断、管理和优化运行在 Kubernetes 和裸机环境中的中间件栈。*

📚 [English README](./README.md)

## 🚀 项目使命

KubeStack-AI 革新云原生中间件运维方式，提供统一的智能 CLI 接口，超越单一工具助手（kubectl-ai、minio-cli-ai）的局限，为整个中间件生态系统提供全面管理。借鉴 k8sgpt 的诊断能力和 kubectl-ai 的智能接口设计，KubeStack-AI 将这些概念扩展到支持具有 AI 驱动洞察的综合中间件栈管理。

## 🎯 为什么选择 KubeStack-AI？

### 解决的痛点

| 传统挑战 | KubeStack-AI 解决方案 |
|---------|---------------------|
| **工具分散** | 统一 CLI 管理所有中间件（Redis、Kafka、PostgreSQL、MinIO 等） |
| **复杂故障排查** | AI 驱动的根因分析与自然语言解释 |
| **手动优化** | 智能性能调优推荐 |
| **知识孤岛** | 内置 10+ 中间件技术专业知识 |
| **被动运维** | 主动健康监控与预测性洞察 |

### 核心优势

- 🧠 **AI 优先设计**：复杂操作的自然语言界面
- 🔌 **插件架构**：可扩展支持任意中间件
- 🎯 **统一体验**：一个工具，全栈管理
- 📊 **智能分析**：根因分析与可执行建议
- 🔐 **安全感知**：权限分析与安全最佳实践
- 🚀 **生产就绪**：为企业级环境而构建

## ✨ 主要功能

### 🔍 自动化诊断
- **通用健康检查**：容器状态、资源利用率、存储健康
- **中间件专项分析**：MySQL、Redis、Kafka、ElasticSearch、RabbitMQ、PostgreSQL、MongoDB、ClickHouse、etcd、Prometheus、MinIO 的定制诊断
- **智能日志分析**：AI 驱动的模式识别与异常检测

### 🎯 智能运维
- **自然语言接口**：`kubestack-ai "为什么我的 Redis 集群很慢？"`
- **自动修复建议**：生成并执行修复命令（需确认）
- **性能优化**：实时调优推荐
- **安全审计**：权限分析与合规检查

### 🔧 插件生态
- **热插拔模块**：无需重新编译即可添加新中间件支持
- **社区驱动**：可扩展的插件市场
- **自定义集成**：为专有系统构建插件

## 🚀 快速开始

### 安装

#### 使用 Go Install
```bash
go install github.com/turtacn/kubestack-ai/cmd/kubestack-ai@latest
````

#### 使用 Homebrew (macOS/Linux)

```bash
brew tap turtacn/kubestack-ai
brew install kubestack-ai
```

#### 使用二进制发布

从 [GitHub Releases](https://github.com/turtacn/kubestack-ai/releases) 下载最新版本

### 快速开始

1. **初始化配置**

```bash
kubestack-ai init
```

2. **配置 AI 提供商**

```bash
kubestack-ai config set ai.provider openai
kubestack-ai config set ai.api_key YOUR_API_KEY
```

3. **安装插件**

```bash
# 安装核心中间件插件
kubestack-ai plugin install redis mysql kafka elasticsearch

# 列出可用插件
kubestack-ai plugin list --available
```

### 基本使用示例

#### 自然语言诊断

```bash
# 分析整体集群健康状况
kubestack-ai analyze "检查我的 Kubernetes 集群健康状况"

# 中间件专项诊断
kubestack-ai diagnose redis "为什么我的 Redis 集群延迟很高？"

# 性能优化
kubestack-ai optimize mysql "我的 MySQL 查询运行很慢"
```

#### 结构化命令

```bash
# 综合健康检查
kubestack-ai health check --middleware redis,mysql,kafka --namespace production

# 生成性能报告
kubestack-ai report --type performance --output json --timerange 1h

# 安全审计
kubestack-ai security audit --middleware all --severity high
```

#### 交互模式

```bash
# 启动交互式会话
kubestack-ai interactive

# 交互式会话示例：
> 分析生产命名空间中的 redis 性能
> 显示 mysql 中的慢查询
> 为高吞吐量优化 kafka topic 配置
```

### 示例输出

```bash
$ kubestack-ai diagnose redis "Redis 内存使用过高"

🔍 KubeStack-AI 分析报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 REDIS 诊断 - production/redis-cluster
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🟡 检测到警告 (2)
├── 内存使用率: 89% (7.1GB / 8GB 分配)
├── 淘汰策略: allkeys-lru 可能导致数据丢失
└── 后台保存: 最后保存在 2 小时前失败

🔍 根因分析
主要问题是内存压力结合后台保存失败，
导致潜在的数据丢失场景。

💡 建议措施
1. 将内存限制从 8GB 增加到 12GB
   ├── 命令: kubectl patch deployment redis -p '{"spec":{"template":{"spec":{"containers":[{"name":"redis","resources":{"limits":{"memory":"12Gi"}}}]}}}}'
   └── 影响: 解决当前内存压力

2. 修复备份配置
   ├── 问题: Redis 备份目录缺少写权限
   ├── 命令: kubectl exec redis-0 -- chmod 755 /data/backup
   └── 验证: 修复后检查备份作业状态

3. 优化内存使用
   ├── 启用压缩: CONFIG SET rdbcompression yes
   ├── 调整保存间隔: CONFIG SET save "900 1 300 10"
   └── 预期减少: ~20% 内存使用

🚀 可用自动修复
运行 'kubestack-ai fix redis --apply-recommendations' 执行建议修复
（破坏性操作需要确认）

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏱  分析完成用时 3.2 秒 | 🤖 由 GPT-4 驱动
```

## 📖 文档

* [架构概览](./docs/architecture.md)
* [插件开发指南](./docs/plugins.md)
* [配置参考](./docs/configuration.md)
* [API 文档](./docs/api.md)
* [贡献指南](./CONTRIBUTING.md)

## 🤝 贡献

我们欢迎社区贡献！KubeStack-AI 依靠 DevOps 工程师、SRE 和中间件专家的集体专业知识而茁壮成长。

### 贡献方式

* 🐛 **错误报告**：帮助我们识别问题
* 💡 **功能请求**：建议新功能
* 🔌 **插件开发**：添加新中间件支持
* 📚 **文档完善**：改进指南和示例
* 🧪 **测试**：帮助我们保持质量
* 🌐 **翻译**：为全球采用本地化

### 贡献者快速开始

1. Fork 仓库
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 进行更改并添加测试
4. 运行测试：`make test`
5. 提交拉取请求

详细指南请参见 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 🛣 路线图

* [ ] **第一步**：核心插件生态系统（Redis、MySQL、PostgreSQL）
* [ ] **第二步**：Kafka、ElasticSearch、RabbitMQ 支持
* [ ] **第三步**：高级 AI 功能（预测分析、自动扩缩容）
* [ ] **第四步**：企业功能（RBAC、审计日志、自定义 AI 模型）

## 📄 许可证

KubeStack-AI 采用 Apache License 2.0 许可。详情请见 [LICENSE](./LICENSE)。

## 🙏 致谢

* 受到 [k8sgpt-ai](https://github.com/k8sgpt-ai/k8sgpt) 和 [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai) 优秀工作的启发
* 由云原生社区用 ❤️ 构建
* 特别感谢所有贡献者和测试用户

---

<div align="center">
  <p><strong>⭐ 如果 KubeStack-AI 帮助了您的 DevOps 之旅，请在 GitHub 上给我们加星！</strong></p>
  <p>如有问题，请加入我们的 <a href="https://discord.gg/kubestack-ai">Discord 社区</a> 或提出 <a href="https://github.com/turtacn/kubestack-ai/issues">issue</a>。</p>
</div>