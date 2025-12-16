# E2E测试指南 (E2E Testing Guide)

## 概述
E2E (End-to-End) 测试旨在验证 KubeStack-AI 从用户输入到最终修复的全链路流程。测试框架位于 `test/e2e/framework`，场景位于 `test/e2e/scenarios`。

## 运行方式

### 运行所有E2E测试
```bash
make e2e-test
```

### 运行特定场景
```bash
go test -v ./test/e2e/scenarios/ -run TestE2E_Diagnosis_Flow
```

## 测试编写规范

### 1. 初始化测试套件
每个测试函数应首先初始化 `E2ETestSuite`:
```go
suite := framework.NewE2ETestSuite(t)
suite.Setup()
defer suite.Teardown()
```

### 2. 构建测试数据
使用 `suite.GraphStore` 构建测试所需的拓扑结构，或使用 `LoadFixture` 加载预定义的 YAML 数据。
```go
ctx := context.Background()
suite.GraphStore.AddNode(ctx, &graph.Node{ID: "svc1", Type: graph.NodeTypeService})
// ... 添加边 ...
```

### 3. 模拟用户交互
通过 `suite.HTTPClient` 向 Mock Server 发送请求，模拟 CLI 或 Web Console 的行为。

### 4. 验证结果
使用 `assert` 包验证 HTTP 状态码、JSON 响应字段以及内部状态变化。

## Fixtures 数据管理
测试数据存储在 `test/e2e/testdata/` 目录下，格式为 YAML。
示例 `topology.yaml`:
```yaml
nodes:
  - id: "service:default/app"
    type: service
edges:
  - from_id: "service:default/app"
    to_id: "middleware:default/redis"
    type: depends_on
```

## CI集成
E2E测试默认使用内存实现的图存储和向量存储，不依赖外部数据库，可在 CI 环境中直接运行。
对于需要真实 Neo4j 或 Kubernetes 的测试，应标记为 Integration Test 并通过 Build Tags 控制。
