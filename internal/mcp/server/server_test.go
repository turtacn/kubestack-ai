package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

func TestNewServer(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
		Capabilities: protocol.ServerCapabilities{
			Tools: &protocol.ToolsCapability{},
		},
	}

	server := NewServer(cfg, registry)
	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config.Name != "test-server" {
		t.Errorf("Expected name 'test-server', got %s", server.config.Name)
	}
}

func TestServer_HandleRequest_Initialize(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
		Capabilities: protocol.ServerCapabilities{
			Tools: &protocol.ToolsCapability{},
		},
	}

	server := NewServer(cfg, registry)

	initParams := protocol.InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	paramsJSON, _ := json.Marshal(initParams)
	reqJSON := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` + string(paramsJSON) + `}`

	ctx := context.Background()
	resp := server.handleRequest(ctx, []byte(reqJSON))

	if resp == nil {
		t.Fatal("Expected response")
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("Expected result")
	}
}

func TestServer_HandleRequest_Ping(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server := NewServer(cfg, registry)

	reqJSON := `{"jsonrpc":"2.0","id":1,"method":"ping"}`

	ctx := context.Background()
	resp := server.handleRequest(ctx, []byte(reqJSON))

	if resp == nil {
		t.Fatal("Expected response")
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}
}

func TestServer_HandleRequest_UnknownMethod(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server := NewServer(cfg, registry)

	reqJSON := `{"jsonrpc":"2.0","id":1,"method":"unknown"}`

	ctx := context.Background()
	resp := server.handleRequest(ctx, []byte(reqJSON))

	if resp == nil {
		t.Fatal("Expected response")
	}

	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if resp.Error.Code != protocol.MethodNotFound {
		t.Errorf("Expected MethodNotFound error, got code %d", resp.Error.Code)
	}
}

func TestServer_HandleRequest_ParseError(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server := NewServer(cfg, registry)

	reqJSON := `{invalid json}`

	ctx := context.Background()
	resp := server.handleRequest(ctx, []byte(reqJSON))

	if resp == nil {
		t.Fatal("Expected response")
	}

	if resp.Error == nil {
		t.Error("Expected parse error")
	}

	if resp.Error.Code != protocol.ParseError {
		t.Errorf("Expected ParseError code, got %d", resp.Error.Code)
	}
}

func TestServer_HandleRequest_ToolsList(t *testing.T) {
	registry := tools.NewRegistry()

	// Register a test tool
	testTool := &tools.Tool{
		Name:        "test-tool",
		Description: "A test tool",
		Source:      tools.SourceLocal,
		Schema:      json.RawMessage(`{"type":"object"}`),
	}
	registry.Register(testTool)

	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server := NewServer(cfg, registry)

	reqJSON := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	ctx := context.Background()
	resp := server.handleRequest(ctx, []byte(reqJSON))

	if resp == nil {
		t.Fatal("Expected response")
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	if resp.Result == nil {
		t.Fatal("Expected result")
	}

	// Parse result
	resultJSON, _ := json.Marshal(resp.Result)
	var listResult protocol.ToolsListResult
	if err := json.Unmarshal(resultJSON, &listResult); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if len(listResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(listResult.Tools))
	}
}

func TestServer_HandleRequest_Notification(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server := NewServer(cfg, registry)

	// Notification has no ID
	reqJSON := `{"jsonrpc":"2.0","method":"notifications/initialized"}`

	ctx := context.Background()
	resp := server.handleRequest(ctx, []byte(reqJSON))

	// Notifications should not return a response
	if resp != nil {
		t.Error("Expected no response for notification")
	}
}

func TestServer_Shutdown(t *testing.T) {
	registry := tools.NewRegistry()
	cfg := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	server := NewServer(cfg, registry)

	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if !server.shutdown {
		t.Error("Expected shutdown flag to be set")
	}
}
