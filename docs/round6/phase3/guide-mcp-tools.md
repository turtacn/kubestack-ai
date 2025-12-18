# MCP Tools Development Guide

## Introduction

This guide provides comprehensive instructions for developing, integrating, and using MCP tools in KubeStack AI. Whether you're connecting to existing MCP servers or creating new ones, this guide covers everything you need to know.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Using MCP Client](#using-mcp-client)
3. [Creating MCP Servers](#creating-mcp-servers)
4. [Tool Discovery](#tool-discovery)
5. [Connection Management](#connection-management)
6. [Best Practices](#best-practices)
7. [Troubleshooting](#troubleshooting)

## Quick Start

### Connecting to an MCP Server

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/kubestack-ai/kubestack-ai/internal/mcp/client"
)

func main() {
    // Configure client
    cfg := client.ClientConfig{
        ServerCommand: "mcp-server-filesystem",
        ServerArgs:    []string{"--root", "/data"},
        Timeout:       30 * time.Second,
    }

    // Create client
    mcpClient := client.NewClient(cfg)

    // Connect
    ctx := context.Background()
    if err := mcpClient.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer mcpClient.Disconnect()

    // List available tools
    tools, err := mcpClient.ListTools(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d tools\n", len(tools))
    for _, tool := range tools {
        fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
    }

    // Call a tool
    result, err := mcpClient.CallTool(ctx, "read_file", map[string]any{
        "path": "/data/example.txt",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %v\n", result)
}
```

### Using the MCP Bridge

```go
package main

import (
    "context"
    "log"

    "github.com/kubestack-ai/kubestack-ai/internal/mcp"
    "github.com/kubestack-ai/kubestack-ai/internal/tools"
)

func main() {
    // Create tool registry
    registry := tools.NewRegistry()

    // Configure bridge with multiple servers
    cfg := mcp.BridgeConfig{
        Servers: []mcp.ServerEntry{
            {
                ID:      "filesystem",
                Command: "mcp-server-filesystem",
                Args:    []string{"--root", "/data"},
            },
            {
                ID:      "github",
                Command: "mcp-server-github",
                Args:    []string{},
                Env:     []string{"GITHUB_TOKEN=your_token"},
            },
        },
        AutoDiscover: true,
    }

    // Create bridge
    bridge, err := mcp.NewMCPBridge(registry, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer bridge.Close()

    // Initialize (connects and discovers tools)
    ctx := context.Background()
    if err := bridge.Initialize(ctx); err != nil {
        log.Printf("Warning: %v", err)
    }

    // List all tools (including MCP tools)
    allTools := registry.List()
    log.Printf("Total tools: %d", len(allTools))

    // Call a tool through the bridge
    result, err := bridge.CallTool(ctx, "filesystem", "read_file",
        map[string]any{"path": "/data/config.yaml"})
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("File content: %v", result)
}
```

## Using MCP Client

### Creating and Configuring a Client

```go
cfg := client.ClientConfig{
    ServerCommand: "path/to/mcp-server",
    ServerArgs:    []string{"--arg1", "value1"},
    ServerEnv:     []string{"VAR=value"},
    Timeout:       30 * time.Second,
    AutoReconnect: false,
}

mcpClient := client.NewClient(cfg)
```

### Connection Lifecycle

```go
// Connect to server
ctx := context.Background()
if err := mcpClient.Connect(ctx); err != nil {
    return err
}

// Check connection status
if mcpClient.IsConnected() {
    fmt.Println("Connected!")
}

// Get server info
info := mcpClient.GetServerInfo()
fmt.Printf("Server: %s v%s\n", info.Name, info.Version)

// Get server capabilities
caps := mcpClient.GetServerCapabilities()
if caps.Tools != nil {
    fmt.Println("Server supports tools")
}

// Disconnect when done
defer mcpClient.Disconnect()
```

### Calling Tools

```go
// Simple tool call
result, err := mcpClient.CallTool(ctx, "tool_name", map[string]any{
    "param1": "value1",
    "param2": 42,
})
if err != nil {
    // Handle error
}

// Process result
if len(result.Content) > 0 {
    text := result.Content[0].Text
    fmt.Println(text)
}

// Check for errors
if result.IsError {
    // Tool reported an error
}
```

### Listing and Caching Tools

```go
// List tools (fetches from server)
tools, err := mcpClient.ListTools(ctx)
if err != nil {
    return err
}

// Get cached tools (no network call)
cachedTools := mcpClient.GetCachedTools()

// Iterate tools
for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name)
    fmt.Printf("Description: %s\n", tool.Description)
    
    // Parse schema
    var schema map[string]any
    json.Unmarshal(tool.InputSchema, &schema)
    fmt.Printf("Schema: %+v\n", schema)
}
```

### Health Checking

```go
// Ping server
if err := mcpClient.Ping(ctx); err != nil {
    log.Printf("Server not responding: %v", err)
    // Attempt reconnect
}
```

## Creating MCP Servers

### Using the Server Framework

```go
package main

import (
    "context"
    "log"

    "github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
    "github.com/kubestack-ai/kubestack-ai/internal/mcp/server"
    "github.com/kubestack-ai/kubestack-ai/internal/tools"
)

func main() {
    // Create tool registry
    registry := tools.NewRegistry()

    // Register local tools
    registerTools(registry)

    // Configure server
    cfg := server.ServerConfig{
        Name:    "my-mcp-server",
        Version: "1.0.0",
        Capabilities: protocol.ServerCapabilities{
            Tools: &protocol.ToolsCapability{
                ListChanged: true,
            },
        },
    }

    // Create server
    mcpServer := server.NewServer(cfg, registry)

    // Serve via stdio
    ctx := context.Background()
    if err := mcpServer.ServeStdio(ctx); err != nil {
        log.Fatal(err)
    }
}

func registerTools(registry tools.Registry) {
    // Example tool
    echoTool := &tools.Tool{
        Name:        "echo",
        Description: "Echoes the input message",
        Source:      tools.SourceLocal,
        Schema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "description": "Message to echo"
                }
            },
            "required": ["message"]
        }`),
        Handler: func(ctx context.Context, args map[string]any) (any, error) {
            message, ok := args["message"].(string)
            if !ok {
                return nil, fmt.Errorf("message must be a string")
            }
            return message, nil
        },
    }

    registry.Register(echoTool)
}
```

### Custom Request Handlers

```go
import "github.com/kubestack-ai/kubestack-ai/internal/mcp/server"

// Create custom handler
type CustomHandler struct {
    data map[string]string
}

func (h *CustomHandler) Handle(ctx context.Context, params any) (any, error) {
    // Parse params
    // Process request
    // Return result
    return map[string]string{"status": "ok"}, nil
}

// Register handler
mcpServer.RegisterHandler("custom/method", &CustomHandler{
    data: make(map[string]string),
})
```

## Tool Discovery

### Automatic Discovery

```go
import (
    "github.com/kubestack-ai/kubestack-ai/internal/mcp/client"
    "github.com/kubestack-ai/kubestack-ai/internal/tools"
)

// Create discovery service
discovery := client.NewDiscovery(mcpClient, registry, "server-id")

// Discover and register tools
count, err := discovery.DiscoverAndRegister(ctx)
if err != nil {
    log.Printf("Discovery error: %v", err)
}
fmt.Printf("Registered %d tools\n", count)

// Tools are now available in registry with "mcp:server-id:" prefix
tool, _ := registry.Get("mcp:server-id:read_file")
```

### Refreshing Tools

```go
// Refresh tools (removes old, discovers new)
if err := discovery.RefreshTools(ctx); err != nil {
    log.Printf("Refresh failed: %v", err)
}
```

### Parsing MCP Tool Names

```go
import "github.com/kubestack-ai/kubestack-ai/internal/mcp/client"

fullName := "mcp:filesystem:read_file"
serverID, toolName, isMCP := client.ParseMCPToolName(fullName)

if isMCP {
    fmt.Printf("Server: %s, Tool: %s\n", serverID, toolName)
}
```

## Connection Management

### Connection Pool

```go
import "github.com/kubestack-ai/kubestack-ai/internal/mcp/client"

// Create pool
pool := client.NewConnectionPool(5 * time.Minute)
defer pool.CloseAll()

// Add servers
pool.AddServer("server1", client.ClientConfig{
    ServerCommand: "mcp-server-1",
})
pool.AddServer("server2", client.ClientConfig{
    ServerCommand: "mcp-server-2",
})

// Get client (creates connection if needed)
cli, err := pool.GetClient(ctx, "server1")
if err != nil {
    return err
}

// Use client
result, _ := cli.CallTool(ctx, "tool", nil)

// Release (marks as available)
pool.ReleaseClient("server1")

// Remove server
pool.RemoveServer("server1")

// Get statistics
stats := pool.GetStats()
fmt.Printf("Active: %d, Idle: %d\n", 
    stats.ActiveConnections, stats.IdleConnections)
```

### Connection Pool with Bridge

```go
// Bridge automatically uses connection pool
bridge, _ := mcp.NewMCPBridge(registry, cfg)

// Get pool stats
stats := bridge.GetPoolStats()
fmt.Printf("Total servers: %d\n", stats.TotalServers)

// Access client directly if needed
cli, _ := bridge.GetServerClient(ctx, "server-id")
```

## Best Practices

### 1. Error Handling

```go
// Always handle connection errors
if err := mcpClient.Connect(ctx); err != nil {
    log.Printf("Connection failed: %v", err)
    // Implement retry logic
    return
}

// Check tool execution errors
result, err := mcpClient.CallTool(ctx, "tool", args)
if err != nil {
    log.Printf("Tool call failed: %v", err)
    return
}

if result.IsError {
    log.Printf("Tool returned error: %s", result.Content[0].Text)
    return
}
```

### 2. Resource Management

```go
// Always defer cleanup
mcpClient := client.NewClient(cfg)
if err := mcpClient.Connect(ctx); err != nil {
    return err
}
defer mcpClient.Disconnect()

// Use context timeouts
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

result, err := mcpClient.CallTool(ctx, "tool", args)
```

### 3. Connection Reuse

```go
// Use connection pool for multiple calls
pool := client.NewConnectionPool(5 * time.Minute)
defer pool.CloseAll()

for i := 0; i < 100; i++ {
    cli, _ := pool.GetClient(ctx, "server1")
    cli.CallTool(ctx, "tool", args)
    pool.ReleaseClient("server1")
}
```

### 4. Schema Validation

```go
// Define clear schemas for tools
schema := json.RawMessage(`{
    "type": "object",
    "properties": {
        "path": {
            "type": "string",
            "description": "File path to read"
        }
    },
    "required": ["path"],
    "additionalProperties": false
}`)

// Validate arguments before calling
// Use a JSON schema validator library
```

### 5. Logging and Monitoring

```go
// Log connection lifecycle
log.Printf("Connecting to %s...", cfg.ServerCommand)
if err := mcpClient.Connect(ctx); err != nil {
    log.Printf("Connection failed: %v", err)
    return err
}
log.Printf("Connected successfully")

// Log tool calls
log.Printf("Calling tool %s with args: %+v", toolName, args)
result, err := mcpClient.CallTool(ctx, toolName, args)
log.Printf("Tool completed in %v", duration)

// Monitor pool statistics
stats := pool.GetStats()
log.Printf("Pool stats: %+v", stats)
```

## Troubleshooting

### Common Issues

#### 1. Connection Timeout

**Problem**: Connection times out during initialization

**Solution**:
```go
// Increase timeout
cfg := client.ClientConfig{
    ServerCommand: "slow-server",
    Timeout:       60 * time.Second, // Increase from default 30s
}
```

#### 2. Tool Not Found

**Problem**: `CallTool` returns "tool not found" error

**Solution**:
```go
// Verify tool exists
tools, _ := mcpClient.ListTools(ctx)
for _, tool := range tools {
    fmt.Println(tool.Name)
}

// Use exact tool name from list
result, _ := mcpClient.CallTool(ctx, "exact_tool_name", args)
```

#### 3. Invalid Arguments

**Problem**: Tool returns "invalid arguments" error

**Solution**:
```go
// Check tool schema
tools, _ := mcpClient.ListTools(ctx)
tool := tools[0]
var schema map[string]any
json.Unmarshal(tool.InputSchema, &schema)
fmt.Printf("Required schema: %+v\n", schema)

// Ensure arguments match schema
args := map[string]any{
    "required_param": "value",
}
```

#### 4. Process Not Starting

**Problem**: Server process fails to start

**Solution**:
```go
// Check if command exists
if _, err := exec.LookPath(cfg.ServerCommand); err != nil {
    log.Printf("Command not found: %s", cfg.ServerCommand)
}

// Check environment variables
cfg.ServerEnv = append(os.Environ(), cfg.ServerEnv...)

// Check stderr for error messages
// The transport provides GetStderr() method
```

#### 5. Memory Leaks

**Problem**: Memory usage grows over time

**Solution**:
```go
// Ensure proper cleanup
defer mcpClient.Disconnect()
defer pool.CloseAll()
defer bridge.Close()

// Set idle timeout for pool
pool := client.NewConnectionPool(5 * time.Minute)

// Periodically refresh tool cache
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        discovery.RefreshTools(ctx)
    }
}()
```

### Debug Mode

```go
// Enable detailed logging (implementation specific)
// Add to your logger configuration

import "log"

// Log all requests and responses
// Wrap client methods with logging
func loggedCallTool(client *client.Client, ctx context.Context, 
    name string, args map[string]any) (any, error) {
    log.Printf("→ Calling tool: %s with args: %+v", name, args)
    result, err := client.CallTool(ctx, name, args)
    if err != nil {
        log.Printf("← Tool error: %v", err)
    } else {
        log.Printf("← Tool success: %+v", result)
    }
    return result, err
}
```

## Advanced Topics

### Custom Transport Implementation

```go
import "github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"

type CustomTransport struct {
    // Your fields
}

func (t *CustomTransport) Send(data []byte) error {
    // Implement send
}

func (t *CustomTransport) Receive() ([]byte, error) {
    // Implement receive
}

func (t *CustomTransport) Close() error {
    // Implement close
}

// Use custom transport
session := protocol.NewSession(&CustomTransport{})
```

### Tool Schema Patterns

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "File path",
      "pattern": "^/.*"
    },
    "mode": {
      "type": "string",
      "enum": ["read", "write", "append"],
      "default": "read"
    },
    "options": {
      "type": "object",
      "properties": {
        "encoding": {"type": "string"},
        "maxSize": {"type": "integer"}
      }
    }
  },
  "required": ["path"],
  "additionalProperties": false
}
```

## Conclusion

This guide covers the essential aspects of working with MCP tools in KubeStack AI. For more information, refer to:

- [MCP Protocol Design](design-mcp-protocol.md)
- [MCP Specification](https://spec.modelcontextprotocol.io/)
- API Documentation (generated from code)

For questions or issues, please open an issue on the project repository.
