package client

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
)

func TestNewClient(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	client := NewClient(cfg)
	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config.Timeout == 0 {
		t.Error("Expected default timeout to be set")
	}
}

func TestClient_IsConnected(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	client := NewClient(cfg)

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}
}

func TestClient_GetCachedTools(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	client := NewClient(cfg)

	// Should return empty slice initially
	tools := client.GetCachedTools()
	if tools == nil {
		t.Error("Expected non-nil slice")
	}

	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}

	// Add some test tools
	client.mu.Lock()
	client.tools = []protocol.ToolDefinition{
		{Name: "tool1", Description: "Test tool 1"},
		{Name: "tool2", Description: "Test tool 2"},
	}
	client.mu.Unlock()

	tools = client.GetCachedTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestClient_DisconnectWithoutConnect(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	client := NewClient(cfg)

	// Should not panic
	if err := client.Disconnect(); err != nil {
		t.Errorf("Disconnect should not error: %v", err)
	}
}

func TestClient_CallToolNotConnected(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	client := NewClient(cfg)
	ctx := context.Background()

	_, err := client.CallTool(ctx, "test", nil)
	if err == nil {
		t.Error("Expected error when calling tool without connection")
	}
}

func TestClient_PingNotConnected(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
		ServerArgs:    []string{},
	}

	client := NewClient(cfg)
	ctx := context.Background()

	err := client.Ping(ctx)
	if err == nil {
		t.Error("Expected error when pinging without connection")
	}
}

func TestClientConfig_DefaultTimeout(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "test",
	}

	client := NewClient(cfg)

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", client.config.Timeout)
	}
}

func TestClient_GetServerInfo(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
	}

	client := NewClient(cfg)

	info := client.GetServerInfo()
	if info != nil {
		t.Error("Expected nil server info when not connected")
	}
}

func TestClient_GetServerCapabilities(t *testing.T) {
	cfg := ClientConfig{
		ServerCommand: "echo",
	}

	client := NewClient(cfg)

	caps := client.GetServerCapabilities()
	if caps != nil {
		t.Error("Expected nil capabilities when not connected")
	}
}
