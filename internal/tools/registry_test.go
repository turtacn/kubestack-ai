package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("Expected registry to be created")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	tool := &Tool{
		Name:        "test-tool",
		Description: "A test tool",
		Source:      SourceLocal,
		Schema:      json.RawMessage(`{"type":"object"}`),
	}

	if err := registry.Register(tool); err != nil {
		t.Errorf("Failed to register tool: %v", err)
	}

	// Register again should succeed (overwrite)
	if err := registry.Register(tool); err != nil {
		t.Errorf("Failed to register tool again: %v", err)
	}
}

func TestRegistry_RegisterNil(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(nil); err == nil {
		t.Error("Expected error when registering nil tool")
	}
}

func TestRegistry_RegisterEmptyName(t *testing.T) {
	registry := NewRegistry()

	tool := &Tool{
		Name:   "",
		Source: SourceLocal,
	}

	if err := registry.Register(tool); err == nil {
		t.Error("Expected error when registering tool with empty name")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	tool := &Tool{
		Name:        "test-tool",
		Description: "A test tool",
		Source:      SourceLocal,
	}

	registry.Register(tool)

	retrieved, err := registry.Get("test-tool")
	if err != nil {
		t.Errorf("Failed to get tool: %v", err)
	}

	if retrieved.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", retrieved.Name)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent tool")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	tool := &Tool{
		Name:   "test-tool",
		Source: SourceLocal,
	}

	registry.Register(tool)

	if err := registry.Unregister("test-tool"); err != nil {
		t.Errorf("Failed to unregister tool: %v", err)
	}

	_, err := registry.Get("test-tool")
	if err == nil {
		t.Error("Expected tool to be removed")
	}
}

func TestRegistry_UnregisterNotFound(t *testing.T) {
	registry := NewRegistry()

	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering nonexistent tool")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	tool1 := &Tool{Name: "tool1", Source: SourceLocal}
	tool2 := &Tool{Name: "tool2", Source: SourceLocal}

	registry.Register(tool1)
	registry.Register(tool2)

	tools := registry.List()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestRegistry_ListEmpty(t *testing.T) {
	registry := NewRegistry()

	tools := registry.List()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}
}

func TestRegistry_ListBySource(t *testing.T) {
	registry := NewRegistry()

	localTool := &Tool{Name: "local", Source: SourceLocal}
	mcpTool1 := &Tool{Name: "mcp1", Source: SourceMCP}
	mcpTool2 := &Tool{Name: "mcp2", Source: SourceMCP}

	registry.Register(localTool)
	registry.Register(mcpTool1)
	registry.Register(mcpTool2)

	localTools := registry.ListBySource(SourceLocal)
	if len(localTools) != 1 {
		t.Errorf("Expected 1 local tool, got %d", len(localTools))
	}

	mcpTools := registry.ListBySource(SourceMCP)
	if len(mcpTools) != 2 {
		t.Errorf("Expected 2 MCP tools, got %d", len(mcpTools))
	}
}

func TestRegistry_UnregisterByPrefix(t *testing.T) {
	registry := NewRegistry()

	tool1 := &Tool{Name: "mcp:server1:tool1", Source: SourceMCP}
	tool2 := &Tool{Name: "mcp:server1:tool2", Source: SourceMCP}
	tool3 := &Tool{Name: "mcp:server2:tool3", Source: SourceMCP}
	tool4 := &Tool{Name: "local-tool", Source: SourceLocal}

	registry.Register(tool1)
	registry.Register(tool2)
	registry.Register(tool3)
	registry.Register(tool4)

	count := registry.UnregisterByPrefix("mcp:server1:")
	if count != 2 {
		t.Errorf("Expected to unregister 2 tools, got %d", count)
	}

	// Verify removal
	_, err := registry.Get("mcp:server1:tool1")
	if err == nil {
		t.Error("Expected tool1 to be removed")
	}

	_, err = registry.Get("mcp:server1:tool2")
	if err == nil {
		t.Error("Expected tool2 to be removed")
	}

	// Verify others still exist
	_, err = registry.Get("mcp:server2:tool3")
	if err != nil {
		t.Error("Expected tool3 to still exist")
	}

	_, err = registry.Get("local-tool")
	if err != nil {
		t.Error("Expected local-tool to still exist")
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()

	called := false
	tool := &Tool{
		Name:   "test-tool",
		Source: SourceLocal,
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			called = true
			return "result", nil
		},
	}

	registry.Register(tool)

	ctx := context.Background()
	result, err := registry.Execute(ctx, "test-tool", nil)
	if err != nil {
		t.Errorf("Failed to execute tool: %v", err)
	}

	if !called {
		t.Error("Expected handler to be called")
	}

	if result != "result" {
		t.Errorf("Expected result 'result', got %v", result)
	}
}

func TestRegistry_ExecuteNotFound(t *testing.T) {
	registry := NewRegistry()

	ctx := context.Background()
	_, err := registry.Execute(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("Expected error when executing nonexistent tool")
	}
}

func TestRegistry_ExecuteNoHandler(t *testing.T) {
	registry := NewRegistry()

	tool := &Tool{
		Name:    "test-tool",
		Source:  SourceLocal,
		Handler: nil,
	}

	registry.Register(tool)

	ctx := context.Background()
	_, err := registry.Execute(ctx, "test-tool", nil)
	if err == nil {
		t.Error("Expected error when executing tool without handler")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	// Register initial tools
	for i := 0; i < 10; i++ {
		tool := &Tool{
			Name:   string(rune('A' + i)),
			Source: SourceLocal,
		}
		registry.Register(tool)
	}

	// Concurrent operations
	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				registry.List()
				registry.ListBySource(SourceLocal)
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				tool := &Tool{
					Name:   string(rune('a' + n)),
					Source: SourceMCP,
				}
				registry.Register(tool)
			}
			done <- true
		}(i)
	}

	// Wait for completion
	for i := 0; i < 10; i++ {
		<-done
	}
}
