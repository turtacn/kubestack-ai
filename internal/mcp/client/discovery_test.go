package client

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

func TestParseMCPToolName(t *testing.T) {
	tests := []struct {
		name         string
		fullName     string
		wantServerID string
		wantToolName string
		wantIsMCP    bool
	}{
		{
			name:         "valid MCP tool name",
			fullName:     "mcp:server1:tool1",
			wantServerID: "server1",
			wantToolName: "tool1",
			wantIsMCP:    true,
		},
		{
			name:         "valid MCP tool with colons in name",
			fullName:     "mcp:server1:tool:subcommand",
			wantServerID: "server1",
			wantToolName: "tool:subcommand",
			wantIsMCP:    true,
		},
		{
			name:         "local tool",
			fullName:     "local-tool",
			wantServerID: "",
			wantToolName: "",
			wantIsMCP:    false,
		},
		{
			name:         "invalid MCP format",
			fullName:     "mcp:server1",
			wantServerID: "",
			wantToolName: "",
			wantIsMCP:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverID, toolName, isMCP := ParseMCPToolName(tt.fullName)

			if serverID != tt.wantServerID {
				t.Errorf("serverID = %v, want %v", serverID, tt.wantServerID)
			}
			if toolName != tt.wantToolName {
				t.Errorf("toolName = %v, want %v", toolName, tt.wantToolName)
			}
			if isMCP != tt.wantIsMCP {
				t.Errorf("isMCP = %v, want %v", isMCP, tt.wantIsMCP)
			}
		})
	}
}

func TestDiscovery_convertToLocalTool(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ClientConfig{
		ServerCommand: "test",
	}
	client := NewClient(cfg)
	discovery := NewDiscovery(client, registry, "test-server")

	schema := json.RawMessage(`{"type":"object","properties":{"arg1":{"type":"string"}}}`)

	def := protocol.ToolDefinition{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: schema,
	}

	tool, err := discovery.convertToLocalTool(def)
	if err != nil {
		t.Fatalf("Failed to convert tool: %v", err)
	}

	if tool.Name != "mcp:test-server:test-tool" {
		t.Errorf("Expected name 'mcp:test-server:test-tool', got %s", tool.Name)
	}

	if tool.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got %s", tool.Description)
	}

	if tool.Source != tools.SourceMCP {
		t.Errorf("Expected source MCP, got %s", tool.Source)
	}

	if tool.ServerID != "test-server" {
		t.Errorf("Expected serverID 'test-server', got %s", tool.ServerID)
	}

	if tool.Handler == nil {
		t.Error("Expected handler to be set")
	}
}

func TestMCPToolWrapper_GetToolName(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName: "original-name",
	}

	if wrapper.GetToolName() != "original-name" {
		t.Errorf("Expected 'original-name', got %s", wrapper.GetToolName())
	}
}

func TestDiscovery_NotConnected(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ClientConfig{
		ServerCommand: "test",
	}
	client := NewClient(cfg)
	discovery := NewDiscovery(client, registry, "test-server")

	ctx := context.Background()
	_, err := discovery.DiscoverAndRegister(ctx)
	if err == nil {
		t.Error("Expected error when client is not connected")
	}
}

func TestNewDiscovery(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ClientConfig{
		ServerCommand: "test",
	}
	client := NewClient(cfg)

	discovery := NewDiscovery(client, registry, "test-server")
	if discovery == nil {
		t.Fatal("Expected discovery to be created")
	}

	if discovery.client != client {
		t.Error("Client not set correctly")
	}

	if discovery.registry != registry {
		t.Error("Registry not set correctly")
	}

	if discovery.serverID != "test-server" {
		t.Error("ServerID not set correctly")
	}
}
