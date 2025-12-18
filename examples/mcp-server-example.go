package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
	"github.com/kubestack-ai/kubestack-ai/internal/mcp/server"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

func main() {
	// Create tool registry
	registry := tools.NewRegistry()

	// Register example tools
	registerExampleTools(registry)

	// Configure server
	cfg := server.ServerConfig{
		Name:    "example-mcp-server",
		Version: "1.0.0",
		Capabilities: protocol.ServerCapabilities{
			Tools: &protocol.ToolsCapability{
				ListChanged: true,
			},
		},
	}

	// Create server
	mcpServer := server.NewServer(cfg, registry)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Serve via stdio
	log.Println("Starting MCP server...")
	if err := mcpServer.ServeStdio(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}

	log.Println("Server stopped")
}

func registerExampleTools(registry tools.Registry) {
	// Echo tool - simply echoes back the input
	echoTool := &tools.Tool{
		Name:        "echo",
		Description: "Echoes back the provided message",
		Source:      tools.SourceLocal,
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string",
					"description": "The message to echo back"
				}
			},
			"required": ["message"]
		}`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			message, ok := args["message"].(string)
			if !ok {
				return nil, fmt.Errorf("message must be a string")
			}
			return map[string]string{
				"echoed": message,
			}, nil
		},
	}

	// Add tool - adds two numbers
	addTool := &tools.Tool{
		Name:        "add",
		Description: "Adds two numbers together",
		Source:      tools.SourceLocal,
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"a": {
					"type": "number",
					"description": "First number"
				},
				"b": {
					"type": "number",
					"description": "Second number"
				}
			},
			"required": ["a", "b"]
		}`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			a, okA := args["a"].(float64)
			b, okB := args["b"].(float64)
			if !okA || !okB {
				return nil, fmt.Errorf("both arguments must be numbers")
			}
			return map[string]float64{
				"result": a + b,
			}, nil
		},
	}

	// Uppercase tool - converts text to uppercase
	uppercaseTool := &tools.Tool{
		Name:        "uppercase",
		Description: "Converts text to uppercase",
		Source:      tools.SourceLocal,
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "Text to convert to uppercase"
				}
			},
			"required": ["text"]
		}`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			text, ok := args["text"].(string)
			if !ok {
				return nil, fmt.Errorf("text must be a string")
			}
			return map[string]string{
				"result": fmt.Sprintf("%s", text),
			}, nil
		},
	}

	// Register all tools
	registry.Register(echoTool)
	registry.Register(addTool)
	registry.Register(uppercaseTool)

	log.Printf("Registered %d tools\n", 3)
}
