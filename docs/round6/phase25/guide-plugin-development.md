# Plugin Development Guide

## Overview

This guide walks you through creating custom middleware diagnostic plugins for KubeStack AI. The plugin architecture provides a standardized way to add support for any middleware system.

## Quick Start

### 1. Create Plugin Package

```bash
mkdir -p plugins/mymiddleware
cd plugins/mymiddleware
```

### 2. Implement the MiddlewarePlugin Interface

```go
package mymiddleware

import (
    "context"
    "fmt"
    "time"
    
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

type MyMiddlewarePlugin struct {
    info      plugin.PluginInfo
    config    plugin.PluginConfig
    client    *MyClient  // Your middleware client
    target    plugin.MiddlewareTarget
    connected bool
}

func NewMyMiddlewarePlugin() plugin.MiddlewarePlugin {
    return &MyMiddlewarePlugin{
        info: plugin.PluginInfo{
            ID:          "mymiddleware-diagnostics",
            Name:        "MyMiddleware Diagnostics Plugin",
            Version:     "1.0.0",
            Type:        plugin.PluginTypeMiddleware,
            Description: "Diagnostics for MyMiddleware",
            Author:      "Your Name",
            Capabilities: []string{
                "health-check",
                "metrics",
                "diagnose",
            },
        },
    }
}
```

### 3. Implement Required Methods

#### Plugin Interface Methods

```go
// Info returns plugin metadata
func (p *MyMiddlewarePlugin) Info() plugin.PluginInfo {
    return p.info
}

// Init initializes the plugin with configuration
func (p *MyMiddlewarePlugin) Init(ctx context.Context, config plugin.PluginConfig) error {
    p.config = config
    // Initialize any resources
    return nil
}

// Start starts the plugin (if needed)
func (p *MyMiddlewarePlugin) Start(ctx context.Context) error {
    // Start background tasks if any
    return nil
}

// Stop stops the plugin and cleans up
func (p *MyMiddlewarePlugin) Stop(ctx context.Context) error {
    if p.connected {
        return p.Disconnect(ctx)
    }
    return nil
}

// HealthCheck verifies plugin is operational
func (p *MyMiddlewarePlugin) HealthCheck(ctx context.Context) error {
    if !p.connected {
        return fmt.Errorf("not connected")
    }
    // Perform health check
    return p.client.Ping(ctx)
}
```

#### MiddlewarePlugin Interface Methods

```go
// MiddlewareType returns the middleware type identifier
func (p *MyMiddlewarePlugin) MiddlewareType() string {
    return "mymiddleware"
}

// SupportedVersions returns supported versions
func (p *MyMiddlewarePlugin) SupportedVersions() []string {
    return []string{"1.x", "2.x"}
}

// Connect establishes connection to middleware
func (p *MyMiddlewarePlugin) Connect(ctx context.Context, target plugin.MiddlewareTarget) error {
    p.target = target
    
    // Create client using target.Endpoints, target.Auth, target.TLS
    client, err := NewMyClient(target)
    if err != nil {
        return fmt.Errorf("failed to create client: %w", err)
    }
    
    // Test connection
    if err := client.Ping(ctx); err != nil {
        return fmt.Errorf("connection test failed: %w", err)
    }
    
    p.client = client
    p.connected = true
    return nil
}

// Disconnect closes the connection
func (p *MyMiddlewarePlugin) Disconnect(ctx context.Context) error {
    if p.client != nil {
        err := p.client.Close()
        p.client = nil
        p.connected = false
        return err
    }
    return nil
}

// Diagnose performs comprehensive diagnostics
func (p *MyMiddlewarePlugin) Diagnose(ctx context.Context, opts plugin.DiagnoseOptions) (*plugin.DiagnosticResult, error) {
    if !p.connected {
        return nil, fmt.Errorf("not connected")
    }
    
    result := &plugin.DiagnosticResult{
        PluginID:   p.info.ID,
        TargetName: p.target.Name,
        Status:     plugin.DiagnosticStatusHealthy,
        Findings:   []plugin.Finding{},
        Metrics:    make(map[string]interface{}),
        Timestamp:  time.Now(),
    }
    
    startTime := time.Now()
    
    // Determine categories to diagnose
    categories := opts.Categories
    if len(categories) == 0 {
        categories = []string{"health", "performance", "resources"}
    }
    
    // Run diagnostics for each category
    for _, category := range categories {
        switch category {
        case "health":
            if err := p.diagnoseHealth(ctx, result); err != nil {
                return nil, err
            }
        case "performance":
            if err := p.diagnosePerformance(ctx, result); err != nil {
                return nil, err
            }
        case "resources":
            if err := p.diagnoseResources(ctx, result); err != nil {
                return nil, err
            }
        }
    }
    
    result.Duration = time.Since(startTime)
    
    // Determine overall status from findings
    for _, finding := range result.Findings {
        if finding.Severity == plugin.SeverityCritical {
            result.Status = plugin.DiagnosticStatusCritical
            break
        } else if finding.Severity == plugin.SeverityError {
            result.Status = plugin.DiagnosticStatusCritical
        } else if finding.Severity == plugin.SeverityWarning && 
                  result.Status == plugin.DiagnosticStatusHealthy {
            result.Status = plugin.DiagnosticStatusWarning
        }
    }
    
    return result, nil
}

// GetMetrics retrieves current metrics
func (p *MyMiddlewarePlugin) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
    if !p.connected {
        return nil, fmt.Errorf("not connected")
    }
    
    metrics := make(map[string]interface{})
    
    // Collect metrics from your middleware
    stats, err := p.client.GetStats(ctx)
    if err != nil {
        return nil, err
    }
    
    metrics["connections"] = stats.Connections
    metrics["requests_per_sec"] = stats.RequestsPerSec
    // ... more metrics
    
    return metrics, nil
}

// Execute performs custom actions
func (p *MyMiddlewarePlugin) Execute(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    if !p.connected {
        return nil, fmt.Errorf("not connected")
    }
    
    switch action {
    case "get-config":
        return p.client.GetConfig(ctx)
    case "flush-cache":
        return nil, p.client.FlushCache(ctx)
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
}
```

### 4. Implement Diagnostic Methods

```go
func (p *MyMiddlewarePlugin) diagnoseHealth(ctx context.Context, result *plugin.DiagnosticResult) error {
    // Check if service is responsive
    if err := p.client.Ping(ctx); err != nil {
        result.Findings = append(result.Findings, plugin.Finding{
            Severity:    plugin.SeverityCritical,
            Category:    "health",
            Title:       "Service Unresponsive",
            Description: "Failed to ping service",
            Evidence: map[string]interface{}{
                "error": err.Error(),
            },
            Remediation: "Check if service is running and accessible",
        })
        return nil
    }
    
    // Add health metrics
    result.Metrics["status"] = "healthy"
    result.Metrics["uptime_seconds"] = p.client.GetUptime()
    
    return nil
}

func (p *MyMiddlewarePlugin) diagnosePerformance(ctx context.Context, result *plugin.DiagnosticResult) error {
    stats, err := p.client.GetPerformanceStats(ctx)
    if err != nil {
        return err
    }
    
    result.Metrics["avg_response_time_ms"] = stats.AvgResponseTime
    result.Metrics["requests_per_sec"] = stats.RequestsPerSec
    
    // Check for performance issues
    if stats.AvgResponseTime > 1000 { // > 1 second
        result.Findings = append(result.Findings, plugin.Finding{
            Severity:    plugin.SeverityWarning,
            Category:    "performance",
            Title:       "High Response Time",
            Description: fmt.Sprintf("Average response time is %.2f ms", stats.AvgResponseTime),
            Evidence: map[string]interface{}{
                "avg_response_time": stats.AvgResponseTime,
                "threshold":         1000,
            },
            Remediation: "Investigate slow queries or increase resources",
        })
    }
    
    return nil
}

func (p *MyMiddlewarePlugin) diagnoseResources(ctx context.Context, result *plugin.DiagnosticResult) error {
    resources, err := p.client.GetResourceUsage(ctx)
    if err != nil {
        return err
    }
    
    result.Metrics["cpu_percent"] = resources.CPUPercent
    result.Metrics["memory_bytes"] = resources.MemoryBytes
    
    // Check resource thresholds
    if resources.CPUPercent > 80 {
        result.Findings = append(result.Findings, plugin.Finding{
            Severity:    plugin.SeverityWarning,
            Category:    "resources",
            Title:       "High CPU Usage",
            Description: fmt.Sprintf("CPU usage is %.1f%%", resources.CPUPercent),
            Evidence: map[string]interface{}{
                "cpu_percent": resources.CPUPercent,
            },
            Remediation: "Consider scaling up or optimizing workload",
        })
    }
    
    return nil
}
```

### 5. Register the Plugin

Create a registration file in your plugin package:

```go
// In plugins/mymiddleware/register.go
package mymiddleware

import "github.com/kubestack-ai/kubestack-ai/internal/plugin"

func init() {
    // Register with global manager
    manager := plugin.GetGlobalManager()
    manager.RegisterBuiltinPlugin("mymiddleware-diagnostics", func() plugin.Plugin {
        return NewMyMiddlewarePlugin()
    })
}
```

### 6. Add to Builtin Registry

Edit `internal/plugins/builtin/register.go`:

```go
import (
    // ... other imports
    "github.com/kubestack-ai/kubestack-ai/plugins/mymiddleware"
)

func RegisterAll(manager *plugin.Manager) {
    // ... existing registrations
    
    // Register MyMiddleware plugin
    manager.RegisterBuiltinPlugin("mymiddleware-diagnostics", func() plugin.Plugin {
        return mymiddleware.NewMyMiddlewarePlugin()
    })
}
```

## Testing Your Plugin

### Unit Tests

```go
// plugins/mymiddleware/plugin_test.go
package mymiddleware

import (
    "context"
    "testing"
    
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewMyMiddlewarePlugin(t *testing.T) {
    p := NewMyMiddlewarePlugin()
    
    assert.NotNil(t, p)
    assert.Equal(t, "mymiddleware-diagnostics", p.Info().ID)
    assert.Equal(t, "mymiddleware", p.(plugin.MiddlewarePlugin).MiddlewareType())
}

func TestMyMiddlewarePlugin_Init(t *testing.T) {
    p := NewMyMiddlewarePlugin()
    
    config := plugin.PluginConfig{
        Enabled: true,
        Settings: map[string]interface{}{
            "timeout": "30s",
        },
    }
    
    err := p.Init(context.Background(), config)
    require.NoError(t, err)
}

func TestMyMiddlewarePlugin_Diagnose(t *testing.T) {
    p := NewMyMiddlewarePlugin()
    
    // Initialize plugin
    err := p.Init(context.Background(), plugin.PluginConfig{})
    require.NoError(t, err)
    
    // Connect to test instance (use mock or testcontainers)
    target := plugin.MiddlewareTarget{
        Type:      "mymiddleware",
        Endpoints: []string{"localhost:1234"},
    }
    err = p.(plugin.MiddlewarePlugin).Connect(context.Background(), target)
    require.NoError(t, err)
    
    // Run diagnostics
    opts := plugin.DiagnoseOptions{
        Categories: []string{"health"},
    }
    result, err := p.(plugin.MiddlewarePlugin).Diagnose(context.Background(), opts)
    require.NoError(t, err)
    
    assert.NotNil(t, result)
    assert.Equal(t, plugin.DiagnosticStatusHealthy, result.Status)
}
```

### Integration Tests

```go
// test/integration/mymiddleware_test.go
package integration

import (
    "context"
    "testing"
    
    "github.com/kubestack-ai/kubestack-ai/internal/plugin"
    "github.com/kubestack-ai/kubestack-ai/plugins/mymiddleware"
    "github.com/testcontainers/testcontainers-go"
)

func TestMyMiddlewareIntegration(t *testing.T) {
    ctx := context.Background()
    
    // Start middleware container
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "mymiddleware:latest",
            ExposedPorts: []string{"1234/tcp"},
        },
        Started: true,
    })
    require.NoError(t, err)
    defer container.Terminate(ctx)
    
    // Get connection details
    host, err := container.Host(ctx)
    require.NoError(t, err)
    port, err := container.MappedPort(ctx, "1234")
    require.NoError(t, err)
    
    // Test plugin
    p := mymiddleware.NewMyMiddlewarePlugin()
    err = p.Init(ctx, plugin.PluginConfig{})
    require.NoError(t, err)
    
    target := plugin.MiddlewareTarget{
        Type:      "mymiddleware",
        Endpoints: []string{fmt.Sprintf("%s:%s", host, port.Port())},
    }
    err = p.(plugin.MiddlewarePlugin).Connect(ctx, target)
    require.NoError(t, err)
    
    result, err := p.(plugin.MiddlewarePlugin).Diagnose(ctx, plugin.DiagnoseOptions{})
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Best Practices

### 1. Error Handling

- Always return descriptive errors
- Use `fmt.Errorf` with `%w` for error wrapping
- Log errors at appropriate levels

```go
if err := p.client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to %s: %w", p.target.Endpoints[0], err)
}
```

### 2. Context Awareness

- Always respect context cancellation
- Use context timeouts from opts.Timeout
- Cancel long-running operations when context is done

```go
func (p *MyMiddlewarePlugin) Diagnose(ctx context.Context, opts plugin.DiagnoseOptions) (*plugin.DiagnosticResult, error) {
    // Create timeout context
    if opts.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
        defer cancel()
    }
    
    // Check context before expensive operations
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue
    }
}
```

### 3. Resource Cleanup

- Implement proper cleanup in Stop()
- Use defer for cleanup in error paths
- Close connections and release resources

```go
func (p *MyMiddlewarePlugin) Stop(ctx context.Context) error {
    if p.client != nil {
        if err := p.client.Close(); err != nil {
            return fmt.Errorf("failed to close client: %w", err)
        }
        p.client = nil
    }
    p.connected = false
    return nil
}
```

### 4. Thread Safety

- Protect shared state with mutexes
- Use atomic operations for counters
- Consider using sync.RWMutex for read-heavy operations

```go
type MyMiddlewarePlugin struct {
    mu        sync.RWMutex
    client    *MyClient
    connected bool
}

func (p *MyMiddlewarePlugin) IsConnected() bool {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.connected
}
```

### 5. Configuration Validation

- Validate configuration in Init()
- Provide sensible defaults
- Document required vs optional settings

```go
func (p *MyMiddlewarePlugin) Init(ctx context.Context, config plugin.PluginConfig) error {
    p.config = config
    
    // Set defaults
    if config.Settings == nil {
        config.Settings = make(map[string]interface{})
    }
    
    if _, ok := config.Settings["timeout"]; !ok {
        config.Settings["timeout"] = "30s"
    }
    
    // Validate required settings
    if threshold, ok := config.Settings["error_threshold"]; ok {
        if t, ok := threshold.(int); !ok || t < 0 {
            return fmt.Errorf("invalid error_threshold: must be non-negative integer")
        }
    }
    
    return nil
}
```

### 6. Metric Collection

- Use consistent metric naming
- Include units in metric names
- Provide both raw and calculated metrics

```go
result.Metrics["memory_used_bytes"] = stats.MemoryUsed
result.Metrics["memory_total_bytes"] = stats.MemoryTotal
result.Metrics["memory_usage_percent"] = float64(stats.MemoryUsed) / float64(stats.MemoryTotal) * 100
```

### 7. Finding Severity

Use appropriate severity levels:

- **Critical**: Service is down or data loss imminent
- **Error**: Functionality impaired, requires immediate attention
- **Warning**: Potential issues, investigate soon
- **Info**: Informational, no action required

```go
if unavailable {
    severity = plugin.SeverityCritical
} else if errorRate > 10 {
    severity = plugin.SeverityError
} else if errorRate > 1 {
    severity = plugin.SeverityWarning
} else {
    severity = plugin.SeverityInfo
}
```

## Examples

See the following built-in plugins for reference:

- **Redis Plugin**: `plugins/redis/plugin.go` - Comprehensive example with multiple modes
- **Kafka Plugin**: `plugins/kafka/plugin.go` - Authentication and consumer lag monitoring
- **MySQL Plugin**: `plugins/mysql/plugin.go` - SQL-based diagnostics

## Troubleshooting

### Plugin Not Loaded

Check that the plugin is registered in `internal/plugins/builtin/register.go`.

### Import Cycle Errors

Ensure plugin packages don't import `internal/plugin` and vice versa. Use interfaces to break cycles.

### Connection Issues

- Verify endpoint format matches your middleware's requirements
- Test connection outside of plugin first
- Check authentication and TLS configuration

### Tests Failing

- Use mocks or testcontainers for external dependencies
- Ensure cleanup happens even on test failure
- Run tests with `-v` flag for detailed output

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add your plugin with tests
4. Update documentation
5. Submit a pull request

For questions, join our Discord or open a GitHub issue.
