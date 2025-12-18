package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
)

// Router handles method routing
type Router struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]Handler),
	}
}

// Register registers a handler for a method
func (r *Router) Register(method string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[method] = handler
}

// Route finds a handler for the given method
func (r *Router) Route(method string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[method]
	return handler, exists
}

// Handle processes a request using the appropriate handler
func (r *Router) Handle(ctx context.Context, req *protocol.Request) *protocol.Response {
	handler, exists := r.Route(req.Method)
	if !exists {
		return &protocol.Response{
			JSONRPC: protocol.JSONRPC20,
			ID:      req.ID,
			Error: &protocol.RPCError{
				Code:    protocol.MethodNotFound,
				Message: "Method not found",
				Data:    fmt.Sprintf("method '%s' is not supported", req.Method),
			},
		}
	}

	result, err := handler.Handle(ctx, req.Params)
	if err != nil {
		return &protocol.Response{
			JSONRPC: protocol.JSONRPC20,
			ID:      req.ID,
			Error: &protocol.RPCError{
				Code:    protocol.InternalError,
				Message: "Internal error",
				Data:    err.Error(),
			},
		}
	}

	return &protocol.Response{
		JSONRPC: protocol.JSONRPC20,
		ID:      req.ID,
		Result:  result,
	}
}
