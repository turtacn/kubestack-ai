package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/mcp/protocol"
	"github.com/kubestack-ai/kubestack-ai/internal/tools"
)

// ServerConfig holds configuration for an MCP server
type ServerConfig struct {
	Name         string
	Version      string
	Capabilities protocol.ServerCapabilities
}

// Server represents an MCP server
type Server struct {
	config   ServerConfig
	router   *Router
	codec    *protocol.Codec
	registry tools.Registry
	mu       sync.RWMutex
	shutdown bool
}

// NewServer creates a new MCP server
func NewServer(cfg ServerConfig, registry tools.Registry) *Server {
	server := &Server{
		config:   cfg,
		router:   NewRouter(),
		codec:    protocol.NewCodec(),
		registry: registry,
	}

	// Register default handlers
	serverInfo := protocol.ServerInfo{
		Name:    cfg.Name,
		Version: cfg.Version,
	}

	server.router.Register(protocol.MethodInitialize, NewInitializeHandler(serverInfo, cfg.Capabilities))
	server.router.Register(protocol.MethodToolsList, NewToolsListHandler(registry))
	server.router.Register(protocol.MethodToolsCall, NewToolsCallHandler(registry))
	server.router.Register(protocol.MethodPing, NewPingHandler())

	return server
}

// RegisterHandler registers a custom handler
func (s *Server) RegisterHandler(method string, handler Handler) {
	s.router.Register(method, handler)
}

// ServeStdio serves MCP protocol over stdio
func (s *Server) ServeStdio(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	// Set larger buffer for large messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.mu.RLock()
		if s.shutdown {
			s.mu.RUnlock()
			return nil
		}
		s.mu.RUnlock()

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("scanner error: %w", err)
			}
			return nil
		}

		data := scanner.Bytes()
		if len(data) == 0 {
			continue
		}

		// Process request
		response := s.handleRequest(ctx, data)
		if response != nil {
			// Encode response
			respData, err := s.encodeResponse(response)
			if err != nil {
				// Log error but continue
				continue
			}

			// Write response to stdout
			if _, err := os.Stdout.Write(respData); err != nil {
				return fmt.Errorf("failed to write response: %w", err)
			}
		}
	}
}

// handleRequest processes a single request
func (s *Server) handleRequest(ctx context.Context, data []byte) *protocol.Response {
	// Decode request
	req, err := s.codec.DecodeRequest(data)
	if err != nil {
		// Return error response
		return &protocol.Response{
			JSONRPC: protocol.JSONRPC20,
			ID:      nil,
			Error: &protocol.RPCError{
				Code:    protocol.ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	// Skip notifications (no response needed)
	if req.IsNotification() {
		return nil
	}

	// Route and handle request
	return s.router.Handle(ctx, req)
}

// encodeResponse encodes a response
func (s *Server) encodeResponse(resp *protocol.Response) ([]byte, error) {
	if resp.Error != nil {
		return s.codec.EncodeError(resp.ID, resp.Error.Code, resp.Error.Message, resp.Error.Data)
	}
	return s.codec.EncodeResponse(resp.ID, resp.Result)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shutdown = true
	return nil
}
