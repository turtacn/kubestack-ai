package protocol

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Transport defines the interface for MCP transport mechanisms
type Transport interface {
	Send(data []byte) error
	Receive() ([]byte, error)
	Close() error
}

// StdioTransport implements Transport using stdin/stdout communication
type StdioTransport struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
	closed  bool
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args []string, env []string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	// Set a larger buffer size for large messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

	return &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: scanner,
	}, nil
}

// Send sends data through stdin
func (t *StdioTransport) Send(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Ensure data ends with newline
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	_, err := t.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}

	return nil
}

// Receive receives data from stdout
func (t *StdioTransport) Receive() ([]byte, error) {
	if t.closed {
		return nil, fmt.Errorf("transport is closed")
	}

	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner error: %w", err)
		}
		return nil, io.EOF
	}

	return t.scanner.Bytes(), nil
}

// Close closes the transport and terminates the process
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true

	// Close stdin to signal the process to exit gracefully
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Kill the process if it doesn't exit gracefully
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}

	// Wait for the process to exit
	if t.cmd != nil {
		t.cmd.Wait()
	}

	// Close remaining pipes
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.stderr != nil {
		t.stderr.Close()
	}

	return nil
}

// GetStderr returns the stderr reader for logging
func (t *StdioTransport) GetStderr() io.Reader {
	return t.stderr
}
