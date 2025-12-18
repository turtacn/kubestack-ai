package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

// Handler defines the interface for request handlers
type Handler interface {
	Handle(ctx context.Context, params any) (any, error)
}

// InitializeHandler handles initialize requests
type InitializeHandler struct {
	serverInfo   protocol.ServerInfo
	capabilities protocol.ServerCapabilities
}

// NewInitializeHandler creates a new initialize handler
func NewInitializeHandler(info protocol.ServerInfo, caps protocol.ServerCapabilities) *InitializeHandler {
	return &InitializeHandler{
		serverInfo:   info,
		capabilities: caps,
	}
}

// Handle processes the initialize request
func (h *InitializeHandler) Handle(ctx context.Context, params any) (any, error) {
	// Parse params
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	var initParams protocol.InitializeParams
	if err := json.Unmarshal(paramsBytes, &initParams); err != nil {
		return nil, fmt.Errorf("failed to parse initialize params: %w", err)
	}

	// Validate protocol version
	if initParams.ProtocolVersion != "2024-11-05" {
		// Accept it anyway but log warning
	}

	// Return initialize result
	result := protocol.InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    h.capabilities,
		ServerInfo:      h.serverInfo,
	}

	return result, nil
}

// ToolsListHandler handles tools/list requests
type ToolsListHandler struct {
	registry tools.Registry
}

// NewToolsListHandler creates a new tools/list handler
func NewToolsListHandler(registry tools.Registry) *ToolsListHandler {
	return &ToolsListHandler{
		registry: registry,
	}
}

// Handle processes the tools/list request
func (h *ToolsListHandler) Handle(ctx context.Context, params any) (any, error) {
	// Get all local tools
	allTools := h.registry.ListBySource(tools.SourceLocal)

	// Convert to protocol format
	toolDefs := make([]protocol.ToolDefinition, 0, len(allTools))
	for _, tool := range allTools {
		def := protocol.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Schema,
		}
		toolDefs = append(toolDefs, def)
	}

	result := protocol.ToolsListResult{
		Tools: toolDefs,
	}

	return result, nil
}

// ToolsCallHandler handles tools/call requests
type ToolsCallHandler struct {
	registry tools.Registry
}

// NewToolsCallHandler creates a new tools/call handler
func NewToolsCallHandler(registry tools.Registry) *ToolsCallHandler {
	return &ToolsCallHandler{
		registry: registry,
	}
}

// Handle processes the tools/call request
func (h *ToolsCallHandler) Handle(ctx context.Context, params any) (any, error) {
	// Parse params
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	var callParams protocol.ToolCallParams
	if err := json.Unmarshal(paramsBytes, &callParams); err != nil {
		return nil, fmt.Errorf("failed to parse call params: %w", err)
	}

	// Execute tool
	result, err := h.registry.Execute(ctx, callParams.Name, callParams.Arguments)
	if err != nil {
		// Return error as content
		return protocol.ToolCallResult{
			Content: []protocol.ContentBlock{
				{
					Type: "text",
					Text: err.Error(),
				},
			},
			IsError: true,
		}, nil
	}

	// Convert result to content blocks
	content := []protocol.ContentBlock{
		{
			Type: "text",
			Text: fmt.Sprintf("%v", result),
		},
	}

	return protocol.ToolCallResult{
		Content: content,
		IsError: false,
	}, nil
}

// PingHandler handles ping requests
type PingHandler struct{}

// NewPingHandler creates a new ping handler
func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

// Handle processes the ping request
func (h *PingHandler) Handle(ctx context.Context, params any) (any, error) {
	return protocol.PingResult{}, nil
}
