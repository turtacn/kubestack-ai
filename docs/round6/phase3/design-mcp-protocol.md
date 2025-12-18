# MCP Protocol Implementation Design

## Overview

This document describes the complete implementation of the Model Context Protocol (MCP) in KubeStack AI. The implementation provides both client and server capabilities for MCP communication, enabling the system to connect to external MCP servers and expose local tools as MCP services.

## Architecture

### Component Hierarchy

```
internal/mcp/
├── protocol/           # Core protocol implementation
│   ├── jsonrpc.go     # JSON-RPC 2.0 codec
│   ├── message.go     # MCP message types
│   ├── transport.go   # Transport layer (stdio, SSE, WebSocket)
│   └── session.go     # Session management
├── client/            # MCP Client implementation
│   ├── client.go      # Client core
│   ├── discovery.go   # Tool discovery mechanism
│   └── pool.go        # Connection pooling
├── server/            # MCP Server implementation
│   ├── server.go      # Server core
│   ├── handler.go     # Request handlers
│   └── router.go      # Method routing
└── bridge.go          # High-level bridge API
```

## Protocol Layer

### JSON-RPC 2.0 Implementation

The protocol layer implements JSON-RPC 2.0 as the foundation for MCP communication:

**Key Features:**
- Request/Response encoding and decoding
- Notification support (requests without ID)
- Standard error codes (ParseError, InvalidRequest, MethodNotFound, etc.)
- Automatic newline handling for stdio transport

**Error Handling:**
```go
type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

### Transport Layer

The transport layer provides an abstraction for different communication mechanisms:

**Interface:**
```go
type Transport interface {
    Send(data []byte) error
    Receive() ([]byte, error)
    Close() error
}
```

**Implementations:**
1. **StdioTransport**: Communication via stdin/stdout pipes
   - Primary transport for spawned MCP servers
   - Line-based message framing
   - Automatic process lifecycle management

2. **SSETransport** (Future): Server-Sent Events over HTTP
3. **WebSocketTransport** (Future): WebSocket bidirectional communication

### Session Management

Sessions manage the lifecycle of MCP connections:

**States:**
- `Disconnected`: No active connection
- `Connecting`: Handshake in progress
- `Connected`: Ready for communication
- `Error`: Connection failure

**Features:**
- Automatic request ID generation
- Pending request tracking with timeouts
- Concurrent request handling
- Graceful shutdown

## MCP Client

### Client Core

The client provides a high-level API for connecting to MCP servers:

```go
type Client struct {
    config  ClientConfig
    session *protocol.Session
    tools   []protocol.ToolDefinition
}
```

**Capabilities:**
- Connection management (Connect/Disconnect)
- Tool invocation (CallTool)
- Tool discovery (ListTools)
- Health checking (Ping)
- Server capability negotiation

### Tool Discovery

The discovery mechanism automatically finds and registers remote tools:

**Process:**
1. Connect to MCP server
2. Send `tools/list` request
3. Parse tool definitions
4. Create local wrappers for each tool
5. Register with local tool registry

**Tool Naming:**
- Format: `mcp:<server-id>:<tool-name>`
- Example: `mcp:filesystem:read_file`
- Enables unique identification across multiple servers

### Connection Pool

The connection pool manages multiple MCP server connections:

**Features:**
- Automatic connection creation and reuse
- Idle connection cleanup
- Connection health monitoring
- Concurrent access safety
- Statistics reporting

**Configuration:**
```go
type ConnectionPool struct {
    clients   map[string]*pooledClient
    configs   map[string]ClientConfig
    maxIdle   time.Duration
}
```

## MCP Server

### Server Core

The server exposes local tools via the MCP protocol:

```go
type Server struct {
    config   ServerConfig
    router   *Router
    codec    *protocol.Codec
    registry tools.Registry
}
```

**Capabilities:**
- Stdio-based serving
- Request routing and handling
- Tool list exposure
- Tool execution
- Graceful shutdown

### Request Handlers

Handlers implement the MCP protocol methods:

1. **InitializeHandler**: Server capability negotiation
2. **ToolsListHandler**: Returns available tools
3. **ToolsCallHandler**: Executes tool with arguments
4. **PingHandler**: Health check endpoint

### Method Routing

The router dispatches requests to appropriate handlers:

```go
type Router struct {
    handlers map[string]Handler
}
```

**Built-in Methods:**
- `initialize`: Handshake and capability exchange
- `tools/list`: List available tools
- `tools/call`: Execute a tool
- `ping`: Health check

## MCP Bridge

The bridge provides a unified interface for MCP operations:

### Features

1. **Multi-Server Management**: Connect to multiple MCP servers simultaneously
2. **Auto-Discovery**: Automatically discover and register tools on connection
3. **Connection Pooling**: Efficient connection reuse
4. **Tool Routing**: Route tool calls to appropriate servers

### Configuration

```go
type BridgeConfig struct {
    Servers      []ServerEntry
    AutoDiscover bool
}

type ServerEntry struct {
    ID      string
    Command string
    Args    []string
    Env     []string
}
```

### Usage Example

```go
// Create bridge
bridge := mcp.NewMCPBridge(registry, mcp.BridgeConfig{
    Servers: []mcp.ServerEntry{
        {
            ID:      "filesystem",
            Command: "mcp-server-filesystem",
            Args:    []string{"--root", "/data"},
        },
    },
    AutoDiscover: true,
})

// Initialize (connect and discover tools)
bridge.Initialize(ctx)

// Call a tool
result, err := bridge.CallTool(ctx, "filesystem", "read_file", 
    map[string]any{"path": "/data/file.txt"})
```

## Tool Registry Integration

### Tool Sources

Tools can come from multiple sources:

```go
type ToolSource string

const (
    SourceLocal ToolSource = "Local"
    SourceMCP   ToolSource = "MCP"
)
```

### MCP Tool Wrapper

Remote MCP tools are wrapped to integrate with the local registry:

```go
type MCPToolWrapper struct {
    client   *Client
    toolName string
}

func (w *MCPToolWrapper) Execute(ctx context.Context, args map[string]any) (any, error) {
    result, err := w.client.CallTool(ctx, w.toolName, args)
    // Extract and return content
}
```

### Registry Extensions

The tool registry was extended to support MCP tools:

**New Methods:**
- `ListBySource(source ToolSource)`: Filter tools by source
- `UnregisterByPrefix(prefix string)`: Bulk removal for server reconnection

## Protocol Flow

### Client Connection Flow

```
1. Create Transport (spawn server process)
2. Create Session
3. Send initialize request
   → Server responds with capabilities
4. Send initialized notification
5. Session state = Connected
6. Discover tools (optional)
```

### Tool Call Flow

```
1. Client generates unique request ID
2. Encode ToolCallParams as JSON-RPC request
3. Send via transport
4. Register pending response channel
5. Wait for response (with timeout)
6. Decode response
7. Extract content blocks
8. Return result
```

### Server Request Flow

```
1. Read line from stdin
2. Decode JSON-RPC request
3. Route to handler based on method
4. Execute handler
5. Encode response
6. Write to stdout
7. Flush output
```

## Error Handling

### Error Types

1. **Transport Errors**: Connection failures, pipe errors
2. **Protocol Errors**: Malformed JSON-RPC messages
3. **Method Errors**: Unknown methods, invalid parameters
4. **Tool Errors**: Tool execution failures

### Error Propagation

```
Tool Error
  → ToolCallResult.IsError = true
  → Client receives error content
  → Returns wrapped error to caller
```

## Testing Strategy

### Unit Tests

- **Protocol Layer**: JSON-RPC encoding/decoding, transport operations
- **Client**: Connection management, tool calls, caching
- **Server**: Request handling, routing, tool execution
- **Registry**: Tool registration, source filtering, prefix removal

### Integration Tests

- **End-to-End**: Client-Server communication via pipes
- **Tool Discovery**: Full discovery and registration flow
- **Connection Pool**: Multi-client scenarios
- **Bridge**: Multi-server coordination

## Performance Considerations

### Connection Pooling

- Reuse connections to avoid repeated handshakes
- Cleanup idle connections to release resources
- Monitor connection health

### Request Handling

- Concurrent request processing
- Timeout management to prevent hangs
- Efficient message framing

### Memory Management

- Bounded buffers for large messages (1MB default)
- Tool cache to avoid repeated discovery
- Cleanup of pending requests on shutdown

## Security Considerations

### Process Isolation

- MCP servers run in separate processes
- Communication only via stdin/stdout
- No direct filesystem or network access by default

### Input Validation

- JSON-RPC message validation
- Parameter type checking
- Schema validation for tool arguments

### Resource Limits

- Message size limits (1MB)
- Request timeout enforcement
- Connection count limits via pool

## Future Enhancements

### Protocol Extensions

1. **SSE Transport**: For browser-based MCP clients
2. **WebSocket Transport**: For bidirectional streaming
3. **Resource Support**: Implement resources/list and resources/read
4. **Prompt Support**: Implement prompts/list and prompts/get

### Advanced Features

1. **Tool Caching**: Cache tool results for identical calls
2. **Batch Requests**: Execute multiple tools in one call
3. **Streaming Results**: Support large result streaming
4. **Authentication**: Add authentication layer for secure connections

### Monitoring

1. **Metrics**: Request latency, error rates, connection stats
2. **Tracing**: Distributed tracing for tool call chains
3. **Logging**: Structured logging for debugging

## Configuration

### Server Configuration

```yaml
mcp:
  servers:
    - id: filesystem
      command: mcp-server-filesystem
      args: ["--root", "/data"]
      env: []
    - id: github
      command: mcp-server-github
      args: ["--token", "${GITHUB_TOKEN}"]
      env: []
  auto_discover: true
  pool:
    max_idle_time: 5m
    cleanup_interval: 1m
```

### Client Configuration

```go
cfg := client.ClientConfig{
    ServerCommand: "mcp-server-tool",
    ServerArgs:    []string{"--config", "config.json"},
    ServerEnv:     []string{"API_KEY=secret"},
    Timeout:       30 * time.Second,
    AutoReconnect: true,
}
```

## Conclusion

The MCP protocol implementation provides a robust, extensible foundation for integrating external tools into KubeStack AI. The modular design allows for easy addition of new transports, handlers, and protocol features while maintaining clean separation of concerns.

The implementation follows MCP specification closely while adding practical features like connection pooling, auto-discovery, and multi-server management that are essential for production use.
