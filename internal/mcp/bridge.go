package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/client"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

// BridgeConfig holds configuration for the MCP bridge
type BridgeConfig struct {
	Servers      []ServerEntry
	AutoDiscover bool
}

// ServerEntry represents an MCP server configuration
type ServerEntry struct {
	ID      string
	Command string
	Args    []string
	Env     []string
}

// MCPBridge bridges between MCP servers and local tools
type MCPBridge struct {
	pool      *client.ConnectionPool
	registry  tools.Registry
	config    BridgeConfig
	discoveries map[string]*client.Discovery
	mu        sync.RWMutex
}

// NewMCPBridge creates a new MCP bridge
func NewMCPBridge(registry tools.Registry, cfg BridgeConfig) (*MCPBridge, error) {
	pool := client.NewConnectionPool(0) // Use default idle timeout

	// Add servers to pool
	for _, server := range cfg.Servers {
		clientCfg := client.ClientConfig{
			ServerCommand: server.Command,
			ServerArgs:    server.Args,
			ServerEnv:     server.Env,
		}
		pool.AddServer(server.ID, clientCfg)
	}

	bridge := &MCPBridge{
		pool:        pool,
		registry:    registry,
		config:      cfg,
		discoveries: make(map[string]*client.Discovery),
	}

	return bridge, nil
}

// Initialize connects to all configured servers and discovers tools
func (b *MCPBridge) Initialize(ctx context.Context) error {
	if !b.config.AutoDiscover {
		return nil
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(b.config.Servers))

	for _, server := range b.config.Servers {
		wg.Add(1)
		go func(srv ServerEntry) {
			defer wg.Done()

			if err := b.ConnectAndDiscover(ctx, srv.ID); err != nil {
				errors <- fmt.Errorf("failed to connect to server %s: %w", srv.ID, err)
			}
		}(server)
	}

	wg.Wait()
	close(errors)

	// Collect errors
	var initErrors []error
	for err := range errors {
		initErrors = append(initErrors, err)
	}

	if len(initErrors) > 0 {
		return fmt.Errorf("failed to initialize some servers: %v", initErrors)
	}

	return nil
}

// ConnectAndDiscover connects to a server and discovers its tools
func (b *MCPBridge) ConnectAndDiscover(ctx context.Context, serverID string) error {
	// Get or create client
	mcpClient, err := b.pool.GetClient(ctx, serverID)
	if err != nil {
		return err
	}

	// Create discovery service
	discovery := client.NewDiscovery(mcpClient, b.registry, serverID)

	b.mu.Lock()
	b.discoveries[serverID] = discovery
	b.mu.Unlock()

	// Discover and register tools
	count, err := discovery.DiscoverAndRegister(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	fmt.Printf("Discovered %d tools from server %s\n", count, serverID)
	return nil
}

// CallTool calls a tool on a specific MCP server
func (b *MCPBridge) CallTool(ctx context.Context, serverID, toolName string, args map[string]any) (any, error) {
	mcpClient, err := b.pool.GetClient(ctx, serverID)
	if err != nil {
		return nil, err
	}

	result, err := mcpClient.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}

	// Extract content
	if len(result.Content) == 0 {
		return nil, nil
	}

	if len(result.Content) == 1 && result.Content[0].Type == "text" {
		return result.Content[0].Text, nil
	}

	return result.Content, nil
}

// RefreshTools refreshes tool discovery for a server
func (b *MCPBridge) RefreshTools(ctx context.Context, serverID string) error {
	b.mu.RLock()
	discovery, exists := b.discoveries[serverID]
	b.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %s not found", serverID)
	}

	return discovery.RefreshTools(ctx)
}

// Close closes all connections
func (b *MCPBridge) Close() error {
	return b.pool.CloseAll()
}

// GetPoolStats returns connection pool statistics
func (b *MCPBridge) GetPoolStats() client.PoolStats {
	return b.pool.GetStats()
}

// ListServers returns the list of configured servers
func (b *MCPBridge) ListServers() []ServerEntry {
	return b.config.Servers
}

// GetServerClient returns the client for a specific server
func (b *MCPBridge) GetServerClient(ctx context.Context, serverID string) (*client.Client, error) {
	return b.pool.GetClient(ctx, serverID)
}
