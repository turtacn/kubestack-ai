package client

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConnectionPool manages a pool of MCP client connections
type ConnectionPool struct {
	clients   map[string]*pooledClient
	configs   map[string]ClientConfig
	mu        sync.RWMutex
	maxIdle   time.Duration
	closeChan chan struct{}
}

// pooledClient wraps a client with metadata
type pooledClient struct {
	client   *Client
	lastUsed time.Time
	mu       sync.Mutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxIdle time.Duration) *ConnectionPool {
	if maxIdle == 0 {
		maxIdle = 5 * time.Minute
	}

	pool := &ConnectionPool{
		clients:   make(map[string]*pooledClient),
		configs:   make(map[string]ClientConfig),
		maxIdle:   maxIdle,
		closeChan: make(chan struct{}),
	}

	// Start cleanup loop
	go pool.cleanupLoop()

	return pool
}

// AddServer adds a server configuration to the pool
func (p *ConnectionPool) AddServer(id string, cfg ClientConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.configs[id] = cfg
}

// GetClient retrieves or creates a client for the given server
func (p *ConnectionPool) GetClient(ctx context.Context, serverID string) (*Client, error) {
	p.mu.RLock()
	cfg, exists := p.configs[serverID]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not configured", serverID)
	}

	p.mu.Lock()
	pooled, exists := p.clients[serverID]
	
	if exists && pooled.client.IsConnected() {
		pooled.lastUsed = time.Now()
		p.mu.Unlock()
		return pooled.client, nil
	}
	p.mu.Unlock()

	// Need to create new connection
	client := NewClient(cfg)
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to server %s: %w", serverID, err)
	}

	pooled = &pooledClient{
		client:   client,
		lastUsed: time.Now(),
	}

	p.mu.Lock()
	p.clients[serverID] = pooled
	p.mu.Unlock()

	return client, nil
}

// ReleaseClient marks a client as available (no-op in this implementation)
func (p *ConnectionPool) ReleaseClient(serverID string) {
	p.mu.RLock()
	pooled, exists := p.clients[serverID]
	p.mu.RUnlock()

	if exists {
		pooled.mu.Lock()
		pooled.lastUsed = time.Now()
		pooled.mu.Unlock()
	}
}

// RemoveServer removes a server and closes its connection
func (p *ConnectionPool) RemoveServer(serverID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove config
	delete(p.configs, serverID)

	// Close and remove client
	if pooled, exists := p.clients[serverID]; exists {
		delete(p.clients, serverID)
		return pooled.client.Disconnect()
	}

	return nil
}

// CloseAll closes all connections in the pool
func (p *ConnectionPool) CloseAll() error {
	close(p.closeChan)

	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for id, pooled := range p.clients {
		if err := pooled.client.Disconnect(); err != nil {
			lastErr = err
		}
		delete(p.clients, id)
	}

	return lastErr
}

// cleanupLoop periodically cleans up idle connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(p.maxIdle / 2)
	defer ticker.Stop()

	for {
		select {
		case <-p.closeChan:
			return
		case <-ticker.C:
			p.cleanup()
		}
	}
}

// cleanup removes connections that have been idle for too long
func (p *ConnectionPool) cleanup() {
	now := time.Now()

	p.mu.Lock()
	defer p.mu.Unlock()

	for id, pooled := range p.clients {
		pooled.mu.Lock()
		idle := now.Sub(pooled.lastUsed)
		pooled.mu.Unlock()

		if idle > p.maxIdle {
			pooled.client.Disconnect()
			delete(p.clients, id)
		}
	}
}

// GetStats returns statistics about the connection pool
func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalServers:      len(p.configs),
		ActiveConnections: 0,
		IdleConnections:   0,
	}

	now := time.Now()
	for _, pooled := range p.clients {
		pooled.mu.Lock()
		idle := now.Sub(pooled.lastUsed)
		pooled.mu.Unlock()

		if pooled.client.IsConnected() {
			stats.ActiveConnections++
			if idle > time.Minute {
				stats.IdleConnections++
			}
		}
	}

	return stats
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	TotalServers      int
	ActiveConnections int
	IdleConnections   int
}
