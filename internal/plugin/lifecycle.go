package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// PluginState defines the state of the plugin
type PluginState int

const (
	StateUninitialized PluginState = iota
	StateInitializing
	StateRunning
	StateStopping
	StateStopped
	StateError
)

// PluginLifecycle manages the lifecycle of a plugin
type PluginLifecycle struct {
	plugin      MiddlewarePlugin
	state       PluginState
	config      *PluginConfig
	healthCheck *HealthChecker
	mu          sync.RWMutex
	log         logger.Logger

	// Event callbacks
	onStateChange func(from, to PluginState)
}

// NewPluginLifecycle creates a new lifecycle manager
func NewPluginLifecycle(plugin MiddlewarePlugin, config *PluginConfig) *PluginLifecycle {
	l := &PluginLifecycle{
		plugin: plugin,
		state:  StateUninitialized,
		config: config,
		log:    logger.NewLogger("PluginLifecycle"),
	}
	l.healthCheck = &HealthChecker{
		interval: 30 * time.Second, // Default interval
		timeout:  5 * time.Second,
		retries:  3,
		plugin:   plugin,
		onUnhealthy: func(err error) {
			l.log.Error("Plugin became unhealthy", "plugin", plugin.Name(), "error", err)
			l.mu.Lock()
			if l.state == StateRunning {
				l.setState(StateError)
			}
			l.mu.Unlock()
		},
	}
	return l
}

// Start starts the plugin
func (l *PluginLifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state != StateUninitialized && l.state != StateStopped && l.state != StateError {
		return fmt.Errorf("invalid state for start: %v", l.state)
	}

	l.setState(StateInitializing)

	// 1. Connect if not connected
	if !l.plugin.IsConnected() {
		if err := l.plugin.Connect(ctx, l.config.Connection); err != nil {
			l.setState(StateError)
			return fmt.Errorf("connect failed: %w", err)
		}
	}

	// 2. Health check
	if err := l.plugin.Ping(ctx); err != nil {
		l.plugin.Disconnect(ctx)
		l.setState(StateError)
		return fmt.Errorf("health check failed: %w", err)
	}

	// 3. Start background health check
	l.healthCheck.Start()

	l.setState(StateRunning)
	return nil
}

// Stop stops the plugin
func (l *PluginLifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state != StateRunning && l.state != StateError {
		return nil
	}

	l.setState(StateStopping)

	// 1. Stop health check
	l.healthCheck.Stop()

	// 2. Disconnect
	if err := l.plugin.Disconnect(ctx); err != nil {
		l.log.Warn("disconnect error", "error", err)
	}

	l.setState(StateStopped)
	return nil
}

// Restart restarts the plugin
func (l *PluginLifecycle) Restart(ctx context.Context) error {
	if err := l.Stop(ctx); err != nil {
		return err
	}
	return l.Start(ctx)
}

// State returns the current state
func (l *PluginLifecycle) State() PluginState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

func (l *PluginLifecycle) setState(state PluginState) {
	oldState := l.state
	l.state = state
	if l.onStateChange != nil {
		l.onStateChange(oldState, state)
	}
}

// HealthChecker checks plugin health periodically
type HealthChecker struct {
	interval    time.Duration
	timeout     time.Duration
	retries     int
	stopCh      chan struct{}
	plugin      MiddlewarePlugin
	onUnhealthy func(err error)
	wg          sync.WaitGroup
}

func (h *HealthChecker) Start() {
	h.stopCh = make(chan struct{})
	h.wg.Add(1)
	go h.run()
}

func (h *HealthChecker) Stop() {
	if h.stopCh != nil {
		close(h.stopCh)
		h.wg.Wait()
		h.stopCh = nil
	}
}

func (h *HealthChecker) run() {
	defer h.wg.Done()
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	failCount := 0

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
			err := h.plugin.Ping(ctx)
			cancel()

			if err != nil {
				failCount++
				if failCount >= h.retries {
					h.onUnhealthy(err)
					failCount = 0
				}
			} else {
				failCount = 0
			}
		}
	}
}
