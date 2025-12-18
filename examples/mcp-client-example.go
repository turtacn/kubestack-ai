package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/client"
)

func main() {
	// Example 1: Basic client usage
	basicExample()

	// Example 2: Using connection pool
	// poolExample()

	// Example 3: Tool discovery
	// discoveryExample()
}

func basicExample() {
	fmt.Println("=== Basic MCP Client Example ===")

	// Configure client to connect to example server
	cfg := client.ClientConfig{
		ServerCommand: "go",
		ServerArgs:    []string{"run", "examples/mcp-server-example.go"},
		Timeout:       30 * time.Second,
	}

	// Create and connect client
	mcpClient := client.NewClient(cfg)
	ctx := context.Background()

	fmt.Println("Connecting to server...")
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer mcpClient.Disconnect()

	// Get server info
	info := mcpClient.GetServerInfo()
	if info != nil {
		fmt.Printf("Connected to: %s v%s\n", info.Name, info.Version)
	}

	// List available tools
	fmt.Println("\nListing available tools...")
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Call echo tool
	fmt.Println("\nCalling 'echo' tool...")
	result, err := mcpClient.CallTool(ctx, "echo", map[string]any{
		"message": "Hello, MCP!",
	})
	if err != nil {
		log.Fatal(err)
	}
	printToolResult("echo", result)

	// Call add tool
	fmt.Println("\nCalling 'add' tool...")
	result, err = mcpClient.CallTool(ctx, "add", map[string]any{
		"a": 42.0,
		"b": 58.0,
	})
	if err != nil {
		log.Fatal(err)
	}
	printToolResult("add", result)

	// Call uppercase tool
	fmt.Println("\nCalling 'uppercase' tool...")
	result, err = mcpClient.CallTool(ctx, "uppercase", map[string]any{
		"text": "hello world",
	})
	if err != nil {
		log.Fatal(err)
	}
	printToolResult("uppercase", result)

	// Test ping
	fmt.Println("\nPinging server...")
	if err := mcpClient.Ping(ctx); err != nil {
		log.Printf("Ping failed: %v", err)
	} else {
		fmt.Println("Ping successful!")
	}

	fmt.Println("\n=== Example completed ===")
}

func poolExample() {
	fmt.Println("=== Connection Pool Example ===")

	// Create connection pool
	pool := client.NewConnectionPool(5 * time.Minute)
	defer pool.CloseAll()

	// Add server configuration
	cfg := client.ClientConfig{
		ServerCommand: "go",
		ServerArgs:    []string{"run", "examples/mcp-server-example.go"},
		Timeout:       30 * time.Second,
	}
	pool.AddServer("example-server", cfg)

	ctx := context.Background()

	// Get client from pool (creates connection)
	fmt.Println("Getting client from pool...")
	cli, err := pool.GetClient(ctx, "example-server")
	if err != nil {
		log.Fatal(err)
	}

	// Use client
	result, err := cli.CallTool(ctx, "echo", map[string]any{
		"message": "Hello from pool!",
	})
	if err != nil {
		log.Fatal(err)
	}
	printToolResult("echo", result)

	// Release client back to pool
	pool.ReleaseClient("example-server")

	// Get client again (reuses existing connection)
	fmt.Println("\nReusing connection from pool...")
	cli, err = pool.GetClient(ctx, "example-server")
	if err != nil {
		log.Fatal(err)
	}

	result, err = cli.CallTool(ctx, "add", map[string]any{
		"a": 10.0,
		"b": 20.0,
	})
	if err != nil {
		log.Fatal(err)
	}
	printToolResult("add", result)

	// Get pool statistics
	stats := pool.GetStats()
	fmt.Printf("\nPool statistics:\n")
	fmt.Printf("  Total servers: %d\n", stats.TotalServers)
	fmt.Printf("  Active connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  Idle connections: %d\n", stats.IdleConnections)

	fmt.Println("\n=== Example completed ===")
}

func discoveryExample() {
	fmt.Println("=== Tool Discovery Example ===")

	// This example would demonstrate the discovery mechanism
	// It requires a registry and bridge setup
	fmt.Println("See guide-mcp-tools.md for discovery examples")

	fmt.Println("\n=== Example completed ===")
}

func printToolResult(toolName string, result any) {
	fmt.Printf("Result from '%s':\n", toolName)
	fmt.Printf("  %+v\n", result)
}
