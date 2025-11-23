package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type PluginHealthChecker struct {
	manager  *PluginManager
	interval time.Duration
	logger   *zap.Logger
	stopCh   chan struct{}
}

func NewPluginHealthChecker(manager *PluginManager, interval time.Duration, logger *zap.Logger) *PluginHealthChecker {
	return &PluginHealthChecker{
		manager:  manager,
		interval: interval,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

func (h *PluginHealthChecker) Start(ctx context.Context) {
	go h.checkLoop(ctx)
}

func (h *PluginHealthChecker) Stop() {
	close(h.stopCh)
}

func (h *PluginHealthChecker) checkLoop(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.runChecks(ctx)
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		}
	}
}

func (h *PluginHealthChecker) runChecks(ctx context.Context) {
	plugins := h.manager.ListPlugins()
	var wg sync.WaitGroup

	for _, info := range plugins {
		if info.State != StateEnabled {
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := h.checkPlugin(ctx, name); err != nil {
				h.logger.Warn("plugin health check failed", zap.String("plugin", name), zap.Error(err))
			}
		}(info.Name)
	}

	wg.Wait()
}

func (h *PluginHealthChecker) checkPlugin(ctx context.Context, name string) error {
	p, err := h.manager.GetPlugin(name)
	if err != nil {
		return err
	}

	checker := p.HealthChecker()
	if checker == nil {
		return nil
	}

	// For general health check, we might need a dummy target or the plugin manages its own target?
	// The HealthChecker interface requires a Target.
	// Usually, a plugin is configured with a target in Initialize, or the Target is passed per request.
	// If the HealthCheck is for the configured target (which makes sense for a "connected" plugin),
	// we should probably pass nil or a default target derived from config.
	// HOWEVER, the interface definition `Check(ctx, target)` implies target-specific checks.
	// If the plugin maintains a connection pool (like Redis), it might check that connection.

	// Let's pass nil for now and assume the plugin checks its internal connection if target is nil,
	// or we need to look at how we want to use this.
	// If the plugin is designed to check *arbitrary* targets, then a periodic health check without a target doesn't make sense.
	// If the plugin represents a *connection* to a middleware (as implied by RedisPlugin struct having a client), then it has a default target.

	// The prompt's RedisPlugin implementation shows `p.client` is initialized in `Initialize`.
	// The `Check` method in RedisHealthChecker:
	// func (c *RedisHealthChecker) Check(ctx context.Context, target *Target) ...
	// It uses `c.plugin.client.Ping(ctx)`. It ignores `target` argument in the pseudo-code for PING check.
	// So passing nil might be fine for the internal connection check.

	status, err := checker.Check(ctx, nil)
	if err != nil {
		return fmt.Errorf("check execution failed: %w", err)
	}

	if status.Overall != HealthyLevel {
		h.logger.Warn("plugin unhealthy",
			zap.String("plugin", name),
			zap.String("status", status.Overall.String()),
			zap.String("summary", status.Summary),
		)
	}

	return nil
}
