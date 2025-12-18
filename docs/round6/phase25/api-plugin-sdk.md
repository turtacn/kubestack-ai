# Plugin SDK API Reference

## Core Interfaces

### Plugin Interface

Base interface that all plugins must implement.

```go
type Plugin interface {
    // Info returns plugin metadata
    Info() PluginInfo
    
    // Lifecycle methods
    Init(ctx context.Context, config PluginConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Health check
    HealthCheck(ctx context.Context) error
}
```

### MiddlewarePlugin Interface

Extended interface for middleware diagnostic plugins.

```go
type MiddlewarePlugin interface {
    Plugin
    
    // Middleware identification
    MiddlewareType() string
    SupportedVersions() []string
    
    // Connection management
    Connect(ctx context.Context, target MiddlewareTarget) error
    Disconnect(ctx context.Context) error
    
    // Diagnostics
    Diagnose(ctx context.Context, opts DiagnoseOptions) (*DiagnosticResult, error)
    GetMetrics(ctx context.Context) (map[string]interface{}, error)
    
    // Custom actions
    Execute(ctx context.Context, action string, params map[string]interface{}) (interface{}, error)
}
```

## Core Types

### PluginInfo

Metadata about a plugin.

```go
type PluginInfo struct {
    ID          string        // Unique identifier (e.g., "redis-diagnostics")
    Name        string        // Display name
    Version     string        // Semantic version
    Type        PluginType    // Plugin type (Middleware, Diagnostic, etc.)
    Description string        // Brief description
    Author      string        // Author or organization
    Homepage    string        // Documentation or project URL
    License     string        // License type
    Requires    []string      // Dependency plugin IDs
    Capabilities []string     // Supported capabilities
}
```

### PluginConfig

Runtime configuration for a plugin.

```go
type PluginConfig struct {
    Enabled  bool                   // Whether plugin is enabled
    Priority int                    // Loading priority (higher first)
    Settings map[string]interface{} // Plugin-specific settings
    Timeout  time.Duration          // Operation timeout
}
```

### MiddlewareTarget

Connection information for a middleware instance.

```go
type MiddlewareTarget struct {
    Type      string            // Middleware type
    Name      string            // Instance name
    Endpoints []string          // Connection endpoints
    Auth      *AuthConfig       // Authentication
    TLS       *TLSConfig        // TLS configuration
    Options   map[string]string // Additional options
}
```

### AuthConfig

Authentication configuration.

```go
type AuthConfig struct {
    Username  string // Username
    Password  string // Password
    Token     string // API token
    SecretRef string // Kubernetes Secret reference
}
```

### TLSConfig

TLS/SSL configuration.

```go
type TLSConfig struct {
    Enabled            bool   // Enable TLS
    CertPath           string // Client certificate path
    KeyPath            string // Client key path
    CAPath             string // CA certificate path
    InsecureSkipVerify bool   // Skip certificate verification
}
```

### DiagnoseOptions

Options for diagnostic execution.

```go
type DiagnoseOptions struct {
    Categories []string      // Diagnostic categories to run
    Depth      string        // quick / standard / deep
    Timeout    time.Duration // Diagnostic timeout
}
```

### DiagnosticResult

Result of a diagnostic operation.

```go
type DiagnosticResult struct {
    PluginID    string                 // Plugin that generated result
    TargetName  string                 // Target instance name
    Status      DiagnosticStatus       // Overall status
    Findings    []Finding              // Discovered issues
    Metrics     map[string]interface{} // Collected metrics
    Suggestions []string               // Recommendations
    Timestamp   time.Time              // When diagnostic ran
    Duration    time.Duration          // How long it took
}
```

### Finding

A single diagnostic finding.

```go
type Finding struct {
    Severity    Severity               // Info / Warning / Error / Critical
    Category    string                 // Category (e.g., "memory", "connection")
    Title       string                 // Short title
    Description string                 // Detailed description
    Evidence    map[string]interface{} // Supporting data
    Remediation string                 // Suggested fix
}
```

## Enumerations

### PluginType

```go
type PluginType string

const (
    PluginTypeMiddleware  PluginType = "middleware"
    PluginTypeDiagnostic  PluginType = "diagnostic"
    PluginTypeAction      PluginType = "action"
    PluginTypeIntegration PluginType = "integration"
)
```

### PluginState

```go
type PluginState int

const (
    PluginStateUnloaded PluginState = iota
    PluginStateLoaded
    PluginStateInitialized
    PluginStateRunning
    PluginStateStopped
    PluginStateFailed
)
```

### DiagnosticStatus

```go
type DiagnosticStatus string

const (
    DiagnosticStatusHealthy  DiagnosticStatus = "healthy"
    DiagnosticStatusWarning  DiagnosticStatus = "warning"
    DiagnosticStatusCritical DiagnosticStatus = "critical"
    DiagnosticStatusUnknown  DiagnosticStatus = "unknown"
)
```

### Severity

```go
type Severity string

const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityError    Severity = "error"
    SeverityCritical Severity = "critical"
)
```

## Plugin Manager

### Manager

Centralized plugin management.

```go
type Manager struct {
    // Private fields
}

func NewManager() *Manager
func GetGlobalManager() *Manager

// Plugin registration
func (m *Manager) RegisterBuiltinPlugin(id string, factory PluginFactory)

// Plugin lifecycle
func (m *Manager) LoadPlugin(ctx context.Context, id string, config PluginConfig) error
func (m *Manager) GetPlugin(id string) (Plugin, error)
func (m *Manager) GetMiddlewarePlugin(middlewareType string) (MiddlewarePlugin, error)
func (m *Manager) ListPlugins() map[string]PluginInfo
func (m *Manager) Shutdown(ctx context.Context) error
```

## Lifecycle Manager

### LifecycleManager

Manages plugin lifecycle operations.

```go
type LifecycleManager struct {
    // Private fields
}

func NewLifecycleManager(registry *Registry, loader *Loader) *LifecycleManager

// Lifecycle operations
func (m *LifecycleManager) InitPlugin(ctx context.Context, id string, config PluginConfig) error
func (m *LifecycleManager) StartPlugin(ctx context.Context, id string) error
func (m *LifecycleManager) StopPlugin(ctx context.Context, id string) error
func (m *LifecycleManager) ReloadPlugin(ctx context.Context, id string, newConfig PluginConfig) error
func (m *LifecycleManager) StartAll(ctx context.Context) error
func (m *LifecycleManager) StopAll(ctx context.Context) error
```

## Sandbox

### Sandbox

Isolated execution environment with resource limits.

```go
type Sandbox struct {
    // Private fields
}

func DefaultSandbox() *Sandbox
func NewSandbox(opts SandboxOptions) *Sandbox

// Execution
func (s *Sandbox) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error)
func (s *Sandbox) ExecutePlugin(ctx context.Context, plugin Plugin, action string, params map[string]interface{}) (interface{}, error)
```

### SandboxOptions

```go
type SandboxOptions struct {
    Timeout           time.Duration  // Execution timeout
    MemoryLimit       int64          // Memory limit in bytes
    CPULimit          float64        // CPU limit (cores)
    AllowedOperations []string       // Whitelisted operations
}
```

## Helper Functions

### Creating Findings

```go
// Create a warning finding
finding := plugin.Finding{
    Severity:    plugin.SeverityWarning,
    Category:    "memory",
    Title:       "High Memory Usage",
    Description: "Memory usage exceeds 80%",
    Evidence: map[string]interface{}{
        "used":    8.5,
        "total":   10.0,
        "percent": 85.0,
    },
    Remediation: "Consider increasing memory allocation",
}
```

### Determining Overall Status

```go
func DetermineStatus(findings []plugin.Finding) plugin.DiagnosticStatus {
    status := plugin.DiagnosticStatusHealthy
    
    for _, finding := range findings {
        switch finding.Severity {
        case plugin.SeverityCritical:
            return plugin.DiagnosticStatusCritical
        case plugin.SeverityError:
            status = plugin.DiagnosticStatusCritical
        case plugin.SeverityWarning:
            if status == plugin.DiagnosticStatusHealthy {
                status = plugin.DiagnosticStatusWarning
            }
        }
    }
    
    return status
}
```

## Usage Examples

### Basic Plugin Implementation

```go
package myplugin

import (
    "context"
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

type MyPlugin struct {
    info   plugin.PluginInfo
    config plugin.PluginConfig
}

func NewMyPlugin() plugin.MiddlewarePlugin {
    return &MyPlugin{
        info: plugin.PluginInfo{
            ID:      "my-plugin",
            Name:    "My Plugin",
            Version: "1.0.0",
            Type:    plugin.PluginTypeMiddleware,
        },
    }
}

func (p *MyPlugin) Info() plugin.PluginInfo {
    return p.info
}

func (p *MyPlugin) Init(ctx context.Context, config plugin.PluginConfig) error {
    p.config = config
    return nil
}

// ... implement other methods
```

### Using the Plugin Manager

```go
package main

import (
    "context"
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
    "github.com/kubestack-ai/kubestack-ai/plugins/redis"
)

func main() {
    // Get global manager
    manager := plugin.GetGlobalManager()
    
    // Register plugin
    manager.RegisterBuiltinPlugin("redis-diagnostics", func() plugin.Plugin {
        return redis.NewRedisPlugin()
    })
    
    // Load and initialize plugin
    config := plugin.PluginConfig{
        Enabled: true,
        Settings: map[string]interface{}{
            "timeout": "30s",
        },
    }
    
    if err := manager.LoadPlugin(context.Background(), "redis-diagnostics", config); err != nil {
        panic(err)
    }
    
    // Get middleware plugin
    p, err := manager.GetMiddlewarePlugin("redis")
    if err != nil {
        panic(err)
    }
    
    // Connect to middleware
    target := plugin.MiddlewareTarget{
        Type:      "redis",
        Endpoints: []string{"localhost:6379"},
    }
    
    if err := p.Connect(context.Background(), target); err != nil {
        panic(err)
    }
    
    // Run diagnostics
    opts := plugin.DiagnoseOptions{
        Categories: []string{"memory", "connection"},
        Depth:      "standard",
    }
    
    result, err := p.Diagnose(context.Background(), opts)
    if err != nil {
        panic(err)
    }
    
    // Process result
    fmt.Printf("Status: %s\n", result.Status)
    fmt.Printf("Findings: %d\n", len(result.Findings))
}
```

### Error Handling

```go
func (p *MyPlugin) Diagnose(ctx context.Context, opts plugin.DiagnoseOptions) (*plugin.DiagnosticResult, error) {
    if !p.connected {
        return nil, fmt.Errorf("plugin not connected")
    }
    
    result := &plugin.DiagnosticResult{
        PluginID:  p.info.ID,
        Status:    plugin.DiagnosticStatusHealthy,
        Timestamp: time.Now(),
    }
    
    // Collect data with error handling
    data, err := p.client.GetData(ctx)
    if err != nil {
        // Add finding instead of returning error
        result.Findings = append(result.Findings, plugin.Finding{
            Severity:    plugin.SeverityError,
            Category:    "data-collection",
            Title:       "Failed to Collect Data",
            Description: err.Error(),
            Remediation: "Check middleware connectivity and permissions",
        })
        result.Status = plugin.DiagnosticStatusCritical
        return result, nil // Return result with error finding
    }
    
    // Process data...
    
    return result, nil
}
```

## Testing Utilities

### Mock Plugin

```go
type MockPlugin struct {
    mock.Mock
}

func (m *MockPlugin) Info() plugin.PluginInfo {
    args := m.Called()
    return args.Get(0).(plugin.PluginInfo)
}

func (m *MockPlugin) Init(ctx context.Context, config plugin.PluginConfig) error {
    args := m.Called(ctx, config)
    return args.Error(0)
}

// ... implement other methods
```

### Test Helpers

```go
func CreateTestPlugin() plugin.MiddlewarePlugin {
    return &TestPlugin{
        info: plugin.PluginInfo{
            ID:   "test-plugin",
            Name: "Test Plugin",
        },
    }
}

func CreateTestTarget() plugin.MiddlewareTarget {
    return plugin.MiddlewareTarget{
        Type:      "test",
        Endpoints: []string{"localhost:1234"},
    }
}
```

## Best Practices

1. **Always check context cancellation** in long-running operations
2. **Return descriptive errors** with context
3. **Use appropriate severity levels** for findings
4. **Include evidence** in findings for debugging
5. **Implement proper cleanup** in Stop() method
6. **Validate configuration** in Init() method
7. **Use mutex** for thread-safe operations
8. **Document public APIs** with godoc comments

## Migration from Legacy Plugins

If you have existing plugins using the old `DiagnosticPlugin` interface:

```go
// Old interface
type DiagnosticPlugin interface {
    Name() string
    Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)
}

// New interface (use EnhancedMiddlewarePlugin)
type MiddlewarePlugin interface {
    Plugin
    // ... additional methods
}
```

Use the `LegacyPluginAdapter` to wrap old plugins:

```go
adapter := &plugin.LegacyPluginAdapter{
    plugin: oldPlugin,
}
manager.Register(adapter)
```

## Support and Resources

- **API Documentation**: https://pkg.go.dev/github.com/kubestack-ai/kubestack-ai
- **Examples**: https://github.com/kubestack-ai/kubestack-ai/tree/main/plugins
- **Issues**: https://github.com/kubestack-ai/kubestack-ai/issues
- **Discord**: https://discord.gg/kubestack-ai
