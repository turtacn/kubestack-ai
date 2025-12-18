package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

// Discovery handles tool discovery from MCP servers
type Discovery struct {
	client   *Client
	registry tools.Registry
	serverID string
}

// NewDiscovery creates a new discovery service
func NewDiscovery(client *Client, registry tools.Registry, serverID string) *Discovery {
	return &Discovery{
		client:   client,
		registry: registry,
		serverID: serverID,
	}
}

// DiscoverAndRegister discovers tools from the MCP server and registers them
func (d *Discovery) DiscoverAndRegister(ctx context.Context) (int, error) {
	if !d.client.IsConnected() {
		return 0, fmt.Errorf("client is not connected")
	}

	// List tools from server
	toolDefs, err := d.client.ListTools(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list tools: %w", err)
	}

	count := 0
	for _, def := range toolDefs {
		tool, err := d.convertToLocalTool(def)
		if err != nil {
			// Log error but continue with other tools
			continue
		}

		if err := d.registry.Register(tool); err != nil {
			// Log error but continue
			continue
		}

		count++
	}

	return count, nil
}

// convertToLocalTool converts an MCP tool definition to a local tool
func (d *Discovery) convertToLocalTool(def protocol.ToolDefinition) (*tools.Tool, error) {
	// Create tool wrapper
	wrapper := &MCPToolWrapper{
		client:   d.client,
		toolName: def.Name,
	}

	tool := &tools.Tool{
		Name:        "mcp:" + d.serverID + ":" + def.Name,
		Description: def.Description,
		Source:      tools.SourceMCP,
		ServerID:    d.serverID,
		Schema:      def.InputSchema,
		Handler:     wrapper.Execute,
	}

	return tool, nil
}

// RefreshTools clears old MCP tools and re-discovers them
func (d *Discovery) RefreshTools(ctx context.Context) error {
	// Remove old tools for this server
	prefix := "mcp:" + d.serverID + ":"
	d.registry.UnregisterByPrefix(prefix)

	// Re-discover
	_, err := d.DiscoverAndRegister(ctx)
	return err
}

// MCPToolWrapper wraps MCP tool calls
type MCPToolWrapper struct {
	client   *Client
	toolName string
}

// Execute executes the MCP tool
func (w *MCPToolWrapper) Execute(ctx context.Context, args map[string]any) (any, error) {
	result, err := w.client.CallTool(ctx, w.toolName, args)
	if err != nil {
		return nil, err
	}

	// Extract text content from result
	if len(result.Content) == 0 {
		return nil, nil
	}

	// If single text content, return just the text
	if len(result.Content) == 1 && result.Content[0].Type == "text" {
		return result.Content[0].Text, nil
	}

	// Otherwise return the full content structure
	return result.Content, nil
}

// GetToolName returns the original tool name (without prefix)
func (w *MCPToolWrapper) GetToolName() string {
	return w.toolName
}

// ParseMCPToolName extracts serverID and tool name from a prefixed tool name
func ParseMCPToolName(fullName string) (serverID, toolName string, isMCP bool) {
	if !strings.HasPrefix(fullName, "mcp:") {
		return "", "", false
	}

	parts := strings.SplitN(fullName[4:], ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}
