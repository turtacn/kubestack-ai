package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/client"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

func TestMCPBridge_NewAndClose(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := BridgeConfig{
		Servers: []ServerEntry{
			{
				ID:      "test1",
				Command: "echo",
				Args:    []string{},
			},
		},
		AutoDiscover: false,
	}

	bridge, err := NewMCPBridge(registry, cfg)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}

	if bridge == nil {
		t.Fatal("Expected bridge to be created")
	}

	if err := bridge.Close(); err != nil {
		t.Errorf("Failed to close bridge: %v", err)
	}
}

func TestMCPBridge_ListServers(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := BridgeConfig{
		Servers: []ServerEntry{
			{ID: "server1", Command: "cmd1"},
			{ID: "server2", Command: "cmd2"},
		},
		AutoDiscover: false,
	}

	bridge, err := NewMCPBridge(registry, cfg)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}
	defer bridge.Close()

	servers := bridge.ListServers()
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}
}

func TestMCPBridge_GetPoolStats(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := BridgeConfig{
		Servers: []ServerEntry{
			{ID: "server1", Command: "cmd1"},
		},
		AutoDiscover: false,
	}

	bridge, err := NewMCPBridge(registry, cfg)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}
	defer bridge.Close()

	stats := bridge.GetPoolStats()
	if stats.TotalServers != 1 {
		t.Errorf("Expected 1 total server, got %d", stats.TotalServers)
	}
}

func TestMCPBridge_CallToolNotConnected(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := BridgeConfig{
		Servers:      []ServerEntry{},
		AutoDiscover: false,
	}

	bridge, err := NewMCPBridge(registry, cfg)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}
	defer bridge.Close()

	ctx := context.Background()
	_, err = bridge.CallTool(ctx, "nonexistent", "tool", nil)
	if err == nil {
		t.Error("Expected error when calling tool on nonexistent server")
	}
}

func TestMCPBridge_RefreshToolsNotFound(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := BridgeConfig{
		Servers:      []ServerEntry{},
		AutoDiscover: false,
	}

	bridge, err := NewMCPBridge(registry, cfg)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}
	defer bridge.Close()

	ctx := context.Background()
	err = bridge.RefreshTools(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when refreshing tools for nonexistent server")
	}
}

func TestConnectionPool_AddAndGet(t *testing.T) {
	pool := client.NewConnectionPool(5 * time.Minute)
	defer pool.CloseAll()

	cfg := client.ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	pool.AddServer("test1", cfg)

	stats := pool.GetStats()
	if stats.TotalServers != 1 {
		t.Errorf("Expected 1 server, got %d", stats.TotalServers)
	}
}

func TestConnectionPool_RemoveServer(t *testing.T) {
	pool := client.NewConnectionPool(5 * time.Minute)
	defer pool.CloseAll()

	cfg := client.ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	pool.AddServer("test1", cfg)
	
	if err := pool.RemoveServer("test1"); err != nil {
		t.Errorf("Failed to remove server: %v", err)
	}

	stats := pool.GetStats()
	if stats.TotalServers != 0 {
		t.Errorf("Expected 0 servers after removal, got %d", stats.TotalServers)
	}
}

func TestConnectionPool_CloseAll(t *testing.T) {
	pool := client.NewConnectionPool(5 * time.Minute)

	cfg := client.ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	pool.AddServer("test1", cfg)
	pool.AddServer("test2", cfg)

	if err := pool.CloseAll(); err != nil {
		t.Errorf("Failed to close all: %v", err)
	}
}

func TestToolsRegistry_MCPIntegration(t *testing.T) {
	registry := tools.NewRegistry()

	// Register local tool
	localTool := &tools.Tool{
		Name:        "local-tool",
		Description: "A local tool",
		Source:      tools.SourceLocal,
		Schema:      json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return "local result", nil
		},
	}
	registry.Register(localTool)

	// Register MCP tool
	mcpTool := &tools.Tool{
		Name:        "mcp:server1:remote-tool",
		Description: "A remote tool",
		Source:      tools.SourceMCP,
		ServerID:    "server1",
		Schema:      json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return "remote result", nil
		},
	}
	registry.Register(mcpTool)

	// List by source
	localTools := registry.ListBySource(tools.SourceLocal)
	if len(localTools) != 1 {
		t.Errorf("Expected 1 local tool, got %d", len(localTools))
	}

	mcpTools := registry.ListBySource(tools.SourceMCP)
	if len(mcpTools) != 1 {
		t.Errorf("Expected 1 MCP tool, got %d", len(mcpTools))
	}

	// Unregister by prefix
	count := registry.UnregisterByPrefix("mcp:server1:")
	if count != 1 {
		t.Errorf("Expected to unregister 1 tool, got %d", count)
	}

	// Verify removal
	mcpTools = registry.ListBySource(tools.SourceMCP)
	if len(mcpTools) != 0 {
		t.Errorf("Expected 0 MCP tools after unregister, got %d", len(mcpTools))
	}
}

func TestMCPBridge_MultipleServers(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := BridgeConfig{
		Servers: []ServerEntry{
			{ID: "server1", Command: "echo", Args: []string{}},
			{ID: "server2", Command: "cat", Args: []string{}},
		},
		AutoDiscover: false,
	}

	bridge, err := NewMCPBridge(registry, cfg)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}
	defer bridge.Close()

	servers := bridge.ListServers()
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	stats := bridge.GetPoolStats()
	if stats.TotalServers != 2 {
		t.Errorf("Expected 2 total servers in pool, got %d", stats.TotalServers)
	}
}
