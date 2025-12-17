# Plugin Architecture Design

## 1. Overview

### 1.1 Design Goals

- **Extensibility**: Enable third-party developers to create custom middleware plugins
- **Hot-Loading**: Support dynamic plugin loading/unloading without service restart
- **Isolation**: Plugin failures should not crash the core system
- **Standardization**: Unified interfaces for all plugin types
- **Performance**: Minimal overhead for plugin operations

### 1.2 Core Concepts

**Plugin**: A self-contained module that provides specific functionality (diagnostics, metrics, actions).

**MiddlewarePlugin**: Specialized plugin for middleware diagnostics (Redis, Kafka, MySQL, etc.).

**PluginManager**: Centralized orchestrator for all plugin operations.

**Lifecycle**: Well-defined states for plugins (Unloaded → Loaded → Initialized → Running → Stopped).

**Sandbox**: Isolated execution environment with resource limits and timeout control.

## 2. Architecture

### 2.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      KubeStack AI                            │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐   │
│  │                   PluginManager                        │   │
│  │  ┌──────────┐  ┌──────────┐  ┌────────────────────┐   │   │
│  │  │  Loader  │  │ Registry │  │ LifecycleManager   │   │   │
│  │  │          │  │          │  │                    │   │   │
│  │  │ Builtin  │  │ By Type  │  │ Init → Start → Stop│   │   │
│  │  │ External │  │ By State │  │ Reload             │   │   │
│  │  │ Factory  │  │ ThreadSafe│  │ Health Check       │   │   │
│  │  └────┬─────┘  └─────┬────┘  └──────┬─────────────┘   │   │
│  │       │              │               │                 │   │
│  │       └──────────────┴───────────────┘                 │   │
│  │                      │                                 │   │
│  │             ┌────────▼────────┐                        │   │
│  │             │    Sandbox      │                        │   │
│  │             │  - Timeout      │                        │   │
│  │             │  - Panic Recov. │                        │   │
│  │             │  - Resource Lim.│                        │   │
│  │             │  - Op Whitelist │                        │   │
│  │             └─────────────────┘                        │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐   │
│  │                 Plugin Ecosystem                       │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐             │   │
│  │  │  Redis   │  │  Kafka   │  │  MySQL   │             │   │
│  │  │  Plugin  │  │  Plugin  │  │  Plugin  │             │   │
│  │  │          │  │          │  │          │             │   │
│  │  │ Memory   │  │ Lag      │  │ Replica  │             │   │
│  │  │ Connect  │  │ Broker   │  │ Slowlog  │             │   │
│  │  │ Replica  │  │ Topic    │  │ Pool     │             │   │
│  │  └──────────┘  └──────────┘  └──────────┘             │   │
│  │                                                         │   │
│  │  ┌──────────┐  ┌──────────┐                            │   │
│  │  │PostgreSQL│  │  Elastic │ (Future)                   │   │
│  │  │  Plugin  │  │  Plugin  │                            │   │
│  │  └──────────┘  └──────────┘                            │   │
│  └───────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Lifecycle State Machine

```
┌────────────┐
│ Unloaded   │
└─────┬──────┘
      │ Load()
      ▼
┌────────────┐
│  Loaded    │◄─────┐
└─────┬──────┘      │
      │ Init()      │ Reload()
      ▼             │
┌────────────┐      │
│Initialized │──────┘
└─────┬──────┘
      │ Start()
      ▼
┌────────────┐
│  Running   │
└─────┬──────┘
      │ Stop()
      ▼
┌────────────┐
│  Stopped   │
└────────────┘

     ┌─────────────┐
     │   Failed    │ (from any state on error)
     └─────────────┘
```

### 2.3 Interface Design

#### Base Plugin Interface

```go
type Plugin interface {
    // Metadata
    Info() PluginInfo
    
    // Lifecycle
    Init(ctx context.Context, config PluginConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Health
    HealthCheck(ctx context.Context) error
}
```

#### MiddlewarePlugin Interface

```go
type MiddlewarePlugin interface {
    Plugin
    
    // Middleware Type
    MiddlewareType() string
    SupportedVersions() []string
    
    // Connection
    Connect(ctx context.Context, target MiddlewareTarget) error
    Disconnect(ctx context.Context) error
    
    // Diagnostics
    Diagnose(ctx context.Context, opts DiagnoseOptions) (*DiagnosticResult, error)
    GetMetrics(ctx context.Context) (map[string]any, error)
    
    // Actions
    Execute(ctx context.Context, action string, params map[string]any) (any, error)
}
```

## 3. Core Components

### 3.1 Enhanced Registry

**Purpose**: Centralized plugin storage with efficient indexing.

**Features**:
- Type-based indexing (Middleware, Diagnostic, Action)
- State tracking for all plugins
- Thread-safe operations (RWMutex)
- Middleware-specific lookups

**Key Methods**:
```go
Register(plugin Plugin, config PluginConfig) error
Unregister(id string) error
Get(id string) (Plugin, error)
GetMiddlewarePlugin(mtype string) (MiddlewarePlugin, error)
ListByType(ptype PluginType) []PluginInfo
GetState(id string) PluginState
SetState(id string, state PluginState) error
```

### 3.2 Lifecycle Manager

**Purpose**: Orchestrate plugin lifecycle operations.

**Features**:
- State validation before operations
- Hook system for lifecycle events
- Health check scheduling
- Graceful shutdown coordination

**Key Methods**:
```go
InitPlugin(ctx context.Context, id string, config PluginConfig) error
StartPlugin(ctx context.Context, id string) error
StopPlugin(ctx context.Context, id string) error
ReloadPlugin(ctx context.Context, id string, newConfig PluginConfig) error
StartAll(ctx context.Context) error
StopAll(ctx context.Context) error
```

### 3.3 Sandbox

**Purpose**: Isolate plugin execution with resource controls.

**Features**:
- Timeout enforcement (context.WithTimeout)
- Panic recovery (defer/recover)
- Resource limit tracking
- Operation whitelisting

**Key Methods**:
```go
Execute(ctx context.Context, fn func(context.Context) (any, error)) (any, error)
ExecutePlugin(ctx context.Context, plugin Plugin, action string, params map[string]any) (any, error)
```

**Implementation**:
```go
func (s *Sandbox) Execute(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
    // Create timeout context
    ctx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()
    
    // Panic recovery
    defer func() {
        if r := recover(); r != nil {
            // Log and convert panic to error
        }
    }()
    
    // Execute function
    return fn(ctx)
}
```

### 3.4 Loader

**Purpose**: Load plugins from various sources.

**Features**:
- Builtin plugin factories (registered at compile time)
- External plugin loading (.so files via plugin.Open)
- Dependency resolution
- Configuration-based loading

**Key Methods**:
```go
RegisterBuiltin(id string, factory PluginFactory)
LoadBuiltin(id string) (MiddlewarePlugin, error)
LoadExternal(path string) (MiddlewarePlugin, error)
LoadFromConfig(configs []PluginConfig) error
DiscoverPlugins(dir string) ([]string, error)
```

### 3.5 Discovery

**Purpose**: Automatically find and identify plugins.

**Features**:
- Directory scanning for .so files
- Manifest parsing (plugin.yaml)
- Version compatibility checking
- Dependency validation

## 4. Built-in Plugins

### 4.1 Redis Plugin

**ID**: `redis-diagnostics`  
**Type**: `Middleware`  
**Capabilities**: `["health-check", "metrics", "diagnose", "execute"]`

**Diagnostic Categories**:
1. **Memory**
   - Usage percentage (used_memory / maxmemory)
   - Fragmentation ratio (mem_fragmentation_ratio)
   - Eviction tracking (evicted_keys)
   - Warnings: >80% usage, fragmentation < 1 or > 1.5

2. **Connection**
   - Connected clients vs maxclients
   - Blocked clients
   - Connection source distribution
   - Idle connection detection

3. **Replication**
   - Master-slave link status
   - Seconds behind master
   - Replication offset lag
   - Backlog size analysis

4. **Persistence**
   - RDB last save status
   - AOF rewrite status
   - Disk space for snapshots
   - Backup verification

5. **Performance**
   - Operations per second
   - Slow log analysis
   - Hit rate calculation (keyspace_hits / (hits + misses))
   - Command statistics

**Mode Support**:
- **Standalone**: Single redis.Client
- **Sentinel**: redis.FailoverClient
- **Cluster**: redis.ClusterClient

### 4.2 Kafka Plugin

**ID**: `kafka-diagnostics`  
**Type**: `Middleware`  
**Capabilities**: `["health-check", "metrics", "diagnose", "lag-monitor"]`

**Diagnostic Categories**:
1. **Broker**
   - Broker availability
   - Metadata validation
   - ISR status
   - Under-replicated partitions

2. **Consumer**
   - Consumer group status
   - Lag calculation (latest offset - consumer offset)
   - Partition assignment
   - Consumer stability

3. **Topic**
   - Topic configuration validation
   - Partition count vs broker count
   - Replication factor adequacy
   - Leader distribution

**Authentication**:
- PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
- TLS/SSL support

### 4.3 MySQL Plugin

**ID**: `mysql-diagnostics`  
**Type**: `Middleware`  
**Capabilities**: `["health-check", "metrics", "diagnose", "slow-query"]`

**Diagnostic Categories**:
1. **Replication**
   - Slave_IO_Running / Slave_SQL_Running
   - Seconds_Behind_Master
   - Last_Error detection
   - Relay log space

2. **Performance**
   - Slow query identification (performance_schema)
   - Full table scan detection
   - Rows examined analysis
   - Query time percentiles

3. **Connection**
   - Connection count vs max_connections
   - Aborted connections
   - Connection source distribution

4. **InnoDB**
   - Buffer pool usage
   - Transaction lock wait time
   - Deadlock detection

## 5. Extension Development

### 5.1 Creating a Custom Plugin

**Step 1**: Implement the MiddlewarePlugin interface

```go
package myplugin

import (
    "context"
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

type MyPlugin struct {
    info   plugin.PluginInfo
    config plugin.PluginConfig
    client *MyClient
}

func NewMyPlugin() plugin.MiddlewarePlugin {
    return &MyPlugin{
        info: plugin.PluginInfo{
            ID:          "my-plugin",
            Name:        "My Custom Plugin",
            Version:     "1.0.0",
            Type:        plugin.PluginTypeMiddleware,
            Description: "Custom middleware diagnostics",
            Capabilities: []string{"health-check", "diagnose"},
        },
    }
}
```

**Step 2**: Implement lifecycle methods

```go
func (p *MyPlugin) Init(ctx context.Context, config plugin.PluginConfig) error {
    p.config = config
    // Initialize resources
    return nil
}

func (p *MyPlugin) Start(ctx context.Context) error {
    // Start background tasks if needed
    return nil
}

func (p *MyPlugin) Stop(ctx context.Context) error {
    // Cleanup resources
    return nil
}
```

**Step 3**: Implement middleware-specific methods

```go
func (p *MyPlugin) Connect(ctx context.Context, target plugin.MiddlewareTarget) error {
    p.client = NewMyClient(target.Endpoints, target.Auth)
    return p.client.Ping(ctx)
}

func (p *MyPlugin) Diagnose(ctx context.Context, opts plugin.DiagnoseOptions) (*plugin.DiagnosticResult, error) {
    result := &plugin.DiagnosticResult{
        PluginID:   p.info.ID,
        TargetName: "my-middleware",
        Status:     plugin.DiagnosticStatusHealthy,
    }
    
    // Run diagnostics
    // Add findings, metrics, suggestions
    
    return result, nil
}
```

**Step 4**: Register the plugin

```go
// For built-in plugins
func init() {
    plugin.RegisterBuiltin("my-plugin", NewMyPlugin)
}

// For external plugins (.so)
var Plugin plugin.MiddlewarePlugin = NewMyPlugin()
```

### 5.2 Plugin Development Best Practices

1. **Error Handling**: Always return descriptive errors
2. **Timeout Awareness**: Respect context cancellation
3. **Resource Cleanup**: Implement proper Stop() cleanup
4. **Thread Safety**: Protect shared state with mutexes
5. **Logging**: Use structured logging with plugin ID
6. **Configuration Validation**: Validate config in Init()
7. **Graceful Degradation**: Handle partial failures
8. **Version Compatibility**: Check middleware versions

### 5.3 Testing Plugins

```go
func TestMyPlugin(t *testing.T) {
    plugin := NewMyPlugin()
    
    // Test Init
    config := plugin.PluginConfig{
        Enabled: true,
        Options: map[string]interface{}{
            "timeout": "30s",
        },
    }
    err := plugin.Init(context.Background(), config)
    require.NoError(t, err)
    
    // Test Connect
    target := plugin.MiddlewareTarget{
        Type:      "my-middleware",
        Endpoints: []string{"localhost:1234"},
    }
    err = plugin.Connect(context.Background(), target)
    require.NoError(t, err)
    
    // Test Diagnose
    result, err := plugin.Diagnose(context.Background(), plugin.DiagnoseOptions{
        Categories: []string{"health"},
    })
    require.NoError(t, err)
    assert.Equal(t, plugin.DiagnosticStatusHealthy, result.Status)
}
```

## 6. Performance Considerations

### 6.1 Benchmarks

- **Plugin Loading**: < 100ms per plugin
- **Registry Lookup**: O(1) for type-based queries
- **Lifecycle Operations**: < 50ms for Init/Start/Stop
- **Sandbox Overhead**: < 10ms per operation
- **Memory Footprint**: ~5MB per plugin instance

### 6.2 Optimization Strategies

1. **Lazy Loading**: Load plugins on-demand
2. **Connection Pooling**: Reuse middleware connections
3. **Caching**: Cache diagnostic results with TTL
4. **Concurrent Execution**: Run independent diagnostics in parallel
5. **Resource Limits**: Enforce memory and CPU limits per plugin

## 7. Security

### 7.1 Plugin Isolation

- **Sandbox Execution**: All plugin operations run in sandbox
- **Timeout Control**: Prevent infinite loops
- **Panic Recovery**: Isolate plugin crashes
- **Resource Limits**: Memory and CPU restrictions

### 7.2 Authentication

- **Credential Management**: Support for Secret references
- **TLS/SSL**: Encrypted connections to middleware
- **Token Refresh**: Automatic credential rotation
- **Audit Logging**: Track plugin actions

## 8. Deployment

### 8.1 Built-in Plugins

Built-in plugins are compiled into the binary:

```go
func init() {
    RegisterBuiltin("redis-diagnostics", NewRedisPlugin)
    RegisterBuiltin("kafka-diagnostics", NewKafkaPlugin)
    RegisterBuiltin("mysql-diagnostics", NewMySQLPlugin)
}
```

### 8.2 External Plugins

External plugins are loaded from a directory:

```
/etc/kubestack-ai/plugins/
├── my-plugin.so
├── my-plugin.yaml
└── another-plugin/
    ├── plugin.so
    └── manifest.yaml
```

**Manifest Format** (`my-plugin.yaml`):
```yaml
id: my-plugin
name: My Custom Plugin
version: 1.0.0
type: middleware
description: Custom middleware diagnostics
author: Your Name
homepage: https://example.com
license: Apache-2.0
requires:
  - redis-diagnostics >= 1.0.0
capabilities:
  - health-check
  - diagnose
config_schema:
  type: object
  properties:
    timeout:
      type: string
      default: "30s"
```

## 9. Future Enhancements

### 9.1 Plugin Marketplace
- Central repository for community plugins
- Version management and updates
- Security scanning and validation
- Plugin ratings and reviews

### 9.2 Advanced Isolation
- **cgroups**: Resource limits on Linux
- **seccomp**: System call filtering
- **namespaces**: Process isolation

### 9.3 Plugin Dependencies
- Automatic dependency resolution
- Version compatibility checks
- Transitive dependency handling

### 9.4 Hot Reload
- Zero-downtime plugin updates
- Version rollback support
- A/B testing for plugin versions

## 10. Troubleshooting

### 10.1 Common Issues

**Plugin fails to load**:
- Check plugin ID uniqueness
- Verify dependencies are installed
- Review plugin logs

**Diagnostic timeout**:
- Increase timeout in plugin config
- Check middleware connectivity
- Review sandbox settings

**Memory leaks**:
- Verify Stop() cleanup implementation
- Check for goroutine leaks
- Monitor resource usage

### 10.2 Debug Mode

Enable debug logging:
```bash
ksa --log-level debug diagnose redis localhost:6379
```

View plugin state:
```bash
ksa plugin list
ksa plugin info redis-diagnostics
```

## Conclusion

The plugin architecture provides a robust, extensible foundation for middleware diagnostics in KubeStack AI. With standardized interfaces, lifecycle management, sandboxed execution, and comprehensive middleware support, the system enables rapid development of new plugins while maintaining stability and performance.

The three initial plugins (Redis, Kafka, MySQL) demonstrate the architecture's capabilities and serve as templates for future extensions.
