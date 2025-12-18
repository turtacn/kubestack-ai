package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolSource represents where a tool comes from
type ToolSource string

const (
	SourceLocal ToolSource = "Local"
	SourceMCP   ToolSource = "MCP"
)

// Tool represents a tool definition
type Tool struct {
	Name        string
	Description string
	Source      ToolSource
	ServerID    string // MCP Server ID (when Source=MCP)
	Schema      json.RawMessage
	Handler     ToolHandler
}

// ToolHandler is a function that executes a tool
type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

// Registry manages tool registration and execution
type Registry interface {
	Register(tool *Tool) error
	Unregister(name string) error
	Get(name string) (*Tool, error)
	List() []*Tool
	ListBySource(source ToolSource) []*Tool
	UnregisterByPrefix(prefix string) int
	Execute(ctx context.Context, name string, args map[string]any) (any, error)
}

// DefaultRegistry is the default implementation of Registry
type DefaultRegistry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() Registry {
	return &DefaultRegistry{
		tools: make(map[string]*Tool),
	}
}

// Register registers a new tool
func (r *DefaultRegistry) Register(tool *Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[tool.Name] = tool
	return nil
}

// Unregister removes a tool from the registry
func (r *DefaultRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	return nil
}

// Get retrieves a tool by name
func (r *DefaultRegistry) Get(name string) (*Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool, nil
}

// List returns all registered tools
func (r *DefaultRegistry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ListBySource returns tools from a specific source
func (r *DefaultRegistry) ListBySource(source ToolSource) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0)
	for _, tool := range r.tools {
		if tool.Source == source {
			tools = append(tools, tool)
		}
	}

	return tools
}

// UnregisterByPrefix removes all tools with a given name prefix
func (r *DefaultRegistry) UnregisterByPrefix(prefix string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for name := range r.tools {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			delete(r.tools, name)
			count++
		}
	}

	return count
}

// Execute executes a tool by name
func (r *DefaultRegistry) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	if tool.Handler == nil {
		return nil, fmt.Errorf("tool %s has no handler", name)
	}

	return tool.Handler(ctx, args)
}
