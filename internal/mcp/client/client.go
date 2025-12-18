package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
)

// ClientConfig holds configuration for an MCP client
type ClientConfig struct {
	ServerCommand string
	ServerArgs    []string
	ServerEnv     []string
	Timeout       time.Duration
	AutoReconnect bool
}

// Client represents an MCP client
type Client struct {
	config  ClientConfig
	session *protocol.Session
	tools   []protocol.ToolDefinition
	mu      sync.RWMutex
}

// NewClient creates a new MCP client
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		config: cfg,
	}
}

// Connect establishes a connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create stdio transport
	transport, err := protocol.NewStdioTransport(
		c.config.ServerCommand,
		c.config.ServerArgs,
		c.config.ServerEnv,
	)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	// Create session
	c.session = protocol.NewSession(transport)

	// Initialize session
	clientInfo := protocol.ClientInfo{
		Name:    "kubestack-ai",
		Version: "1.0.0",
	}

	capabilities := protocol.ClientCapabilities{
		Roots: &protocol.RootsCapability{
			ListChanged: true,
		},
	}

	if err := c.session.Initialize(clientInfo, capabilities); err != nil {
		c.session.Close()
		c.session = nil
		return fmt.Errorf("failed to initialize session: %w", err)
	}

	// Discover tools
	if err := c.discoverTools(ctx); err != nil {
		// Log warning but don't fail connection
		// Tools can be discovered later
	}

	return nil
}

// Disconnect closes the connection to the MCP server
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == nil {
		return nil
	}

	err := c.session.Close()
	c.session = nil
	c.tools = nil
	return err
}

// CallTool calls a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*protocol.ToolCallResult, error) {
	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	params := protocol.ToolCallParams{
		Name:      name,
		Arguments: args,
	}

	result, err := session.Call(protocol.MethodToolsCall, params)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	// Parse result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var toolResult protocol.ToolCallResult
	if err := json.Unmarshal(resultBytes, &toolResult); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if toolResult.IsError {
		// Extract error message from content
		errMsg := "tool execution failed"
		if len(toolResult.Content) > 0 && toolResult.Content[0].Text != "" {
			errMsg = toolResult.Content[0].Text
		}
		return &toolResult, fmt.Errorf("%s", errMsg)
	}

	return &toolResult, nil
}

// ListTools retrieves the list of available tools from the server
func (c *Client) ListTools(ctx context.Context) ([]protocol.ToolDefinition, error) {
	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	result, err := session.Call(protocol.MethodToolsList, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Parse result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var listResult protocol.ToolsListResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	c.mu.Lock()
	c.tools = listResult.Tools
	c.mu.Unlock()

	return listResult.Tools, nil
}

// GetCachedTools returns the cached list of tools
func (c *Client) GetCachedTools() []protocol.ToolDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions
	tools := make([]protocol.ToolDefinition, len(c.tools))
	copy(tools, c.tools)
	return tools
}

// Ping sends a ping request to verify the connection
func (c *Client) Ping(ctx context.Context) error {
	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("client is not connected")
	}

	_, err := session.Call(protocol.MethodPing, nil)
	return err
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return false
	}

	return c.session.State == protocol.StateConnected
}

// discoverTools discovers tools after connection
func (c *Client) discoverTools(ctx context.Context) error {
	_, err := c.ListTools(ctx)
	return err
}

// GetServerInfo returns information about the connected server
func (c *Client) GetServerInfo() *protocol.ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return nil
	}

	return c.session.ServerInfo
}

// GetServerCapabilities returns the capabilities of the connected server
func (c *Client) GetServerCapabilities() *protocol.ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return nil
	}

	return c.session.Capabilities
}
