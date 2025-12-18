package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// SessionState represents the state of a session
type SessionState int

const (
	StateDisconnected SessionState = iota
	StateConnecting
	StateConnected
	StateError
)

func (s SessionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Session represents an MCP session
type Session struct {
	ID              string
	Transport       Transport
	Codec           *Codec
	ServerInfo      *ServerInfo
	Capabilities    *ServerCapabilities
	State           SessionState
	pendingRequests map[any]chan *Response
	nextID          int64
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// NewSession creates a new MCP session
func NewSession(transport Transport) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	session := &Session{
		ID:              generateSessionID(),
		Transport:       transport,
		Codec:           NewCodec(),
		State:           StateDisconnected,
		pendingRequests: make(map[any]chan *Response),
		nextID:          0,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start receive loop
	session.wg.Add(1)
	go session.receiveLoop()

	return session
}

// Initialize performs the MCP initialization handshake
func (s *Session) Initialize(clientInfo ClientInfo, caps ClientCapabilities) error {
	s.setState(StateConnecting)

	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    caps,
		ClientInfo:      clientInfo,
	}

	result, err := s.Call(MethodInitialize, params)
	if err != nil {
		s.setState(StateError)
		return fmt.Errorf("initialize failed: %w", err)
	}

	// Parse initialize result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		s.setState(StateError)
		return fmt.Errorf("failed to marshal initialize result: %w", err)
	}

	var initResult InitializeResult
	if err := json.Unmarshal(resultBytes, &initResult); err != nil {
		s.setState(StateError)
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	s.ServerInfo = &initResult.ServerInfo
	s.Capabilities = &initResult.Capabilities

	// Send initialized notification
	if err := s.Notify(MethodInitialized, nil); err != nil {
		s.setState(StateError)
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	s.setState(StateConnected)
	return nil
}

// Call makes a synchronous RPC call
func (s *Session) Call(method string, params any) (any, error) {
	return s.CallWithTimeout(method, params, 30*time.Second)
}

// CallWithTimeout makes a synchronous RPC call with timeout
func (s *Session) CallWithTimeout(method string, params any, timeout time.Duration) (any, error) {
	if s.State == StateDisconnected || s.State == StateError {
		return nil, fmt.Errorf("session is not connected")
	}

	// Generate unique ID
	id := atomic.AddInt64(&s.nextID, 1)

	// Create response channel
	respChan := make(chan *Response, 1)

	s.mu.Lock()
	s.pendingRequests[id] = respChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pendingRequests, id)
		s.mu.Unlock()
		close(respChan)
	}()

	// Encode and send request
	data, err := s.Codec.EncodeRequest(id, method, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	if err := s.Transport.Send(data); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response with timeout
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		if s.ctx.Err() != nil {
			return nil, fmt.Errorf("session closed")
		}
		return nil, fmt.Errorf("request timeout after %v", timeout)
	}
}

// Notify sends a notification (no response expected)
func (s *Session) Notify(method string, params any) error {
	if s.State == StateDisconnected || s.State == StateError {
		return fmt.Errorf("session is not connected")
	}

	// Encode and send notification (ID is nil)
	data, err := s.Codec.EncodeRequest(nil, method, params)
	if err != nil {
		return fmt.Errorf("failed to encode notification: %w", err)
	}

	if err := s.Transport.Send(data); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

// receiveLoop continuously receives and processes messages
func (s *Session) receiveLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		data, err := s.Transport.Receive()
		if err != nil {
			if err.Error() != "EOF" && !s.isClosed() {
				s.setState(StateError)
			}
			return
		}

		// Decode response
		resp, err := s.Codec.DecodeResponse(data)
		if err != nil {
			// Log error but continue
			continue
		}

		// Handle response
		if resp.ID != nil {
			s.mu.RLock()
			respChan, exists := s.pendingRequests[resp.ID]
			s.mu.RUnlock()

			if exists {
				select {
				case respChan <- resp:
				case <-s.ctx.Done():
					return
				}
			}
		}
		// Note: Notifications from server could be handled here
	}
}

// Close closes the session
func (s *Session) Close() error {
	s.setState(StateDisconnected)
	s.cancel()

	// Close all pending requests
	s.mu.Lock()
	for _, ch := range s.pendingRequests {
		close(ch)
	}
	s.pendingRequests = make(map[any]chan *Response)
	s.mu.Unlock()

	// Wait for receive loop to finish
	s.wg.Wait()

	// Close transport
	return s.Transport.Close()
}

// setState safely updates the session state
func (s *Session) setState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
}

// isClosed checks if the session is closed
func (s *Session) isClosed() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}
