# 插件开发指南

## 概述

本文档指导第三方开发者为智能诊断系统开发自定义中间件诊断插件。

## 插件接口

所有插件必须实现 `DiagnosticPlugin` 接口：

```go
type DiagnosticPlugin interface {
    Name() string
    SupportedTypes() []string
    Version() string
    Init(config map[string]interface{}) error
    Diagnose(ctx context.Context, req *DiagnosisRequest) (*DiagnosisResult, error)
    Shutdown() error
}
```

### 方法说明

1. **Name()**: 返回插件的唯一名称（如 "redis", "mysql"）
2. **SupportedTypes()**: 返回插件支持的中间件类型列表
3. **Version()**: 返回插件版本号（遵循语义化版本规范）
4. **Init()**: 插件初始化方法，接收配置参数
5. **Diagnose()**: 核心诊断逻辑，返回诊断结果
6. **Shutdown()**: 清理资源，关闭连接

## 开发示例

以下是一个简单的 Memcached 插件示例：

```go
package memcached

import (
    "context"
    "fmt"
    "github.com/bradfitz/gomemcache/memcache"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

func init() {
    plugin.RegisterPluginFactory("memcached", func() plugin.DiagnosticPlugin {
        return &MemcachedPlugin{}
    })
}

type MemcachedPlugin struct {
    client *memcache.Client
}

func (p *MemcachedPlugin) Name() string {
    return "memcached"
}

func (p *MemcachedPlugin) SupportedTypes() []string {
    return []string{"memcached"}
}

func (p *MemcachedPlugin) Version() string {
    return "1.0.0"
}

func (p *MemcachedPlugin) Init(config map[string]interface{}) error {
    servers := config["servers"].([]interface{})
    var serverStrings []string
    for _, s := range servers {
        serverStrings = append(serverStrings, s.(string))
    }
    p.client = memcache.New(serverStrings...)
    return p.client.Ping()
}

func (p *MemcachedPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
    result := &models.DiagnosisResult{
        Issues: []*models.Issue{},
    }

    // 检查连接状态
    if err := p.client.Ping(); err != nil {
        result.Issues = append(result.Issues, &models.Issue{
            Title:       "Memcached连接失败",
            Severity:    "CRITICAL",
            Description: err.Error(),
            Source:      "MemcachedPlugin",
        })
    }

    // 更多诊断逻辑...

    return result, nil
}

func (p *MemcachedPlugin) Shutdown() error {
    return nil // Memcached客户端无需显式关闭
}
```

## 注册插件

### 方式1: 配置文件注册

在 `configs/plugin/plugins.yaml` 中添加：

```yaml
enabled_plugins:
  - name: memcached
    enabled: true
    config:
      servers:
        - "localhost:11211"
```

### 方式2: 代码注册

如果您正在扩展核心代码库，请确保在插件包的 `init()` 函数中调用 `plugin.RegisterPluginFactory`。

```go
func init() {
    plugin.RegisterPluginFactory("memcached", func() plugin.DiagnosticPlugin {
        return &MemcachedPlugin{}
    })
}
```

## 最佳实践

1. **错误处理**: 在 `Diagnose()` 方法中妥善处理所有错误，避免 panic
2. **超时控制**: 尊重 context 的超时设置
3. **资源清理**: 在 `Shutdown()` 中释放所有资源
4. **日志记录**: 使用结构化日志记录关键操作
5. **测试覆盖**: 为插件编写完整的单元测试和集成测试

## 安全要求

1. 不得访问系统敏感路径（如 `/etc/passwd`）
2. 不得执行系统命令（除非明确必要且经过审核）
3. 不得建立不受信任的网络连接
4. 配置中的敏感信息（如密码）应加密存储

## 发布插件

1. 创建 GitHub 仓库
2. 添加 `README.md` 和使用文档
3. 编写测试并确保覆盖率 > 80%
4. 发布到 Go Module Registry
5. 提交 PR 到官方插件目录

## 示例插件

官方维护的插件仓库：

* [redis-plugin](https://github.com/diagnosis/plugin-redis)
* [mysql-plugin](https://github.com/diagnosis/plugin-mysql)
* [kafka-plugin](https://github.com/diagnosis/plugin-kafka)

## 故障排查

### 插件加载失败

1. 检查插件名称是否与配置文件一致
2. 确认插件实现了所有接口方法
3. 查看日志中的详细错误信息

### 诊断结果为空

1. 确认 `Diagnose()` 方法正确返回了 `DiagnosisResult`
2. 检查中间件连接配置是否正确
3. 验证中间件实例是否正常运行

## 社区支持

* 讨论组: [https://github.com/diagnosis/discussions](https://github.com/diagnosis/discussions)
* Issue: [https://github.com/diagnosis/issues](https://github.com/diagnosis/issues)
* 邮件列表: [dev@diagnosis.io](mailto:dev@diagnosis.io)
