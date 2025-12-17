package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTimeout = errors.New("execution timeout")
	ErrPanic   = errors.New("execution panic")
)

// SandboxOptions configures the sandbox environment
type SandboxOptions struct {
	Timeout           time.Duration
	MemoryLimit       int64
	CPULimit          float64
	AllowedOperations []string
}

// Sandbox provides isolated execution environment for plugins
type Sandbox struct {
	timeout    time.Duration
	memLimit   int64
	cpuLimit   float64
	allowedOps []string
}

// NewSandbox creates a new sandbox with the given options
func NewSandbox(opts SandboxOptions) *Sandbox {
	return &Sandbox{
		timeout:    opts.Timeout,
		memLimit:   opts.MemoryLimit,
		cpuLimit:   opts.CPULimit,
		allowedOps: opts.AllowedOperations,
	}
}

// Execute executes a function in the sandbox with timeout and panic recovery
func (s *Sandbox) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (result interface{}, err error) {
	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	
	// Create channel for result
	done := make(chan struct{})
	
	// Execute with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%w: %v", ErrPanic, r)
			}
			close(done)
		}()
		
		result, err = fn(execCtx)
	}()
	
	// Wait for completion or timeout
	select {
	case <-done:
		return result, err
	case <-execCtx.Done():
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, execCtx.Err()
	}
}

// ExecutePlugin executes a plugin action in the sandbox
func (s *Sandbox) ExecutePlugin(ctx context.Context, plugin Plugin, action string, params map[string]interface{}) (interface{}, error) {
	// Check if action is allowed
	if len(s.allowedOps) > 0 && !s.isAllowed(action) {
		return nil, fmt.Errorf("action not allowed: %s", action)
	}
	
	// Execute the action
	return s.Execute(ctx, func(execCtx context.Context) (interface{}, error) {
		// Cast to EnhancedMiddlewarePlugin and execute
		if mwPlugin, ok := plugin.(EnhancedMiddlewarePlugin); ok {
			return mwPlugin.Execute(execCtx, action, params)
		}
		return nil, fmt.Errorf("plugin does not support Execute method")
	})
}

// isAllowed checks if an operation is in the allowed list
func (s *Sandbox) isAllowed(operation string) bool {
	for _, allowed := range s.allowedOps {
		if allowed == operation {
			return true
		}
	}
	return false
}

// DefaultSandbox creates a sandbox with default settings
func DefaultSandbox() *Sandbox {
	return NewSandbox(SandboxOptions{
		Timeout:     5 * time.Minute,
		MemoryLimit: 256 * 1024 * 1024, // 256MB
		CPULimit:    1.0,
		AllowedOperations: []string{
			"diagnose",
			"get-metrics",
			"health-check",
			"get-slow-logs",
			"get-client-list",
			"get-config",
		},
	})
}
