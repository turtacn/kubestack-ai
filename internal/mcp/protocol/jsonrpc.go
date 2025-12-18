package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	JSONRPC20 = "2.0"

	// Standard JSON-RPC 2.0 error codes
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface
func (e *RPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("JSON-RPC error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Codec handles encoding and decoding of JSON-RPC messages
type Codec struct{}

// NewCodec creates a new JSON-RPC codec
func NewCodec() *Codec {
	return &Codec{}
}

// EncodeRequest encodes a JSON-RPC request
func (c *Codec) EncodeRequest(id any, method string, params any) ([]byte, error) {
	req := Request{
		JSONRPC: JSONRPC20,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Add newline for stdio protocol
	data = append(data, '\n')
	return data, nil
}

// DecodeRequest decodes a JSON-RPC request
func (c *Codec) DecodeRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, &RPCError{
			Code:    ParseError,
			Message: "Parse error",
			Data:    err.Error(),
		}
	}

	if req.JSONRPC != JSONRPC20 {
		return nil, &RPCError{
			Code:    InvalidRequest,
			Message: "Invalid Request",
			Data:    fmt.Sprintf("jsonrpc field must be '2.0', got '%s'", req.JSONRPC),
		}
	}

	if req.Method == "" {
		return nil, &RPCError{
			Code:    InvalidRequest,
			Message: "Invalid Request",
			Data:    "method field is required",
		}
	}

	return &req, nil
}

// EncodeResponse encodes a successful JSON-RPC response
func (c *Codec) EncodeResponse(id any, result any) ([]byte, error) {
	resp := Response{
		JSONRPC: JSONRPC20,
		ID:      id,
		Result:  result,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Add newline for stdio protocol
	data = append(data, '\n')
	return data, nil
}

// EncodeError encodes an error JSON-RPC response
func (c *Codec) EncodeError(id any, code int, message string, data any) ([]byte, error) {
	resp := Response{
		JSONRPC: JSONRPC20,
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error response: %w", err)
	}

	// Add newline for stdio protocol
	encoded = append(encoded, '\n')
	return encoded, nil
}

// DecodeResponse decodes a JSON-RPC response
func (c *Codec) DecodeResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, &RPCError{
			Code:    ParseError,
			Message: "Parse error",
			Data:    err.Error(),
		}
	}

	if resp.JSONRPC != JSONRPC20 {
		return nil, &RPCError{
			Code:    InvalidRequest,
			Message: "Invalid Response",
			Data:    fmt.Sprintf("jsonrpc field must be '2.0', got '%s'", resp.JSONRPC),
		}
	}

	return &resp, nil
}

// IsNotification checks if a request is a notification (no ID)
func (r *Request) IsNotification() bool {
	return r.ID == nil
}
