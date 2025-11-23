package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type PluginMiddleware interface {
	Wrap(next PluginCallFunc) PluginCallFunc
}

type PluginCallFunc func(ctx context.Context, req *PluginRequest) (*PluginResponse, error)

type PluginRequest struct {
	PluginName string
	Method     string
	Params     interface{}
}

type PluginResponse struct {
	Data interface{}
}

// TimeoutMiddleware 超时控制
type TimeoutMiddleware struct {
	timeout time.Duration
	logger  *zap.Logger
}

func NewTimeoutMiddleware(timeout time.Duration, logger *zap.Logger) *TimeoutMiddleware {
	return &TimeoutMiddleware{timeout: timeout, logger: logger}
}

func (m *TimeoutMiddleware) Wrap(next PluginCallFunc) PluginCallFunc {
	return func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, m.timeout)
		defer cancel()

		type result struct {
			resp *PluginResponse
			err  error
		}
		resultCh := make(chan result, 1)

		go func() {
			resp, err := next(ctx, req)
			resultCh <- result{resp, err}
		}()

		select {
		case res := <-resultCh:
			return res.resp, res.err
		case <-ctx.Done():
			m.logger.Warn("plugin call timeout",
				zap.String("plugin", req.PluginName),
				zap.String("method", req.Method),
			)
			return nil, fmt.Errorf("plugin call timeout after %v", m.timeout)
		}
	}
}

// RetryMiddleware 重试逻辑
type RetryMiddleware struct {
	maxRetries int
	backoff    time.Duration
	logger     *zap.Logger
}

func NewRetryMiddleware(maxRetries int, backoff time.Duration, logger *zap.Logger) *RetryMiddleware {
	return &RetryMiddleware{
		maxRetries: maxRetries,
		backoff:    backoff,
		logger:     logger,
	}
}

func (m *RetryMiddleware) Wrap(next PluginCallFunc) PluginCallFunc {
	return func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		var lastErr error
		for attempt := 0; attempt <= m.maxRetries; attempt++ {
			resp, err := next(ctx, req)
			if err == nil {
				return resp, nil
			}

			lastErr = err

			if !isRetriableError(err) {
				return nil, err
			}

			if attempt < m.maxRetries {
				backoffDuration := m.backoff * time.Duration(1<<uint(attempt))
				m.logger.Info("retrying plugin call",
					zap.String("plugin", req.PluginName),
					zap.Int("attempt", attempt+1),
					zap.Duration("backoff", backoffDuration),
				)

				select {
				case <-time.After(backoffDuration):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}
		return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
	}
}

func isRetriableError(err error) bool {
	// Simple heuristic: retriable unless explicitly stated otherwise or canceled
	// In production, we should check error types more strictly.
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	return true
}

// CircuitBreakerMiddleware 熔断器
type CircuitBreakerMiddleware struct {
	breaker *gobreaker.CircuitBreaker
	logger  *zap.Logger
}

type CircuitBreakerConfig struct {
	Name        string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

func NewCircuitBreakerMiddleware(config CircuitBreakerConfig, logger *zap.Logger) *CircuitBreakerMiddleware {
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("circuit breaker state change",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}
	breaker := gobreaker.NewCircuitBreaker(settings)

	return &CircuitBreakerMiddleware{
		breaker: breaker,
		logger:  logger,
	}
}

func (m *CircuitBreakerMiddleware) Wrap(next PluginCallFunc) PluginCallFunc {
	return func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		result, err := m.breaker.Execute(func() (interface{}, error) {
			return next(ctx, req)
		})

		if err != nil {
			if err == gobreaker.ErrOpenState {
				m.logger.Warn("circuit breaker open, request rejected",
					zap.String("plugin", req.PluginName),
				)
			}
			return nil, err
		}

		return result.(*PluginResponse), nil
	}
}

// RateLimitMiddleware 限流
type RateLimitMiddleware struct {
	limiter *rate.Limiter
	logger  *zap.Logger
}

func NewRateLimitMiddleware(rps int, burst int, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		logger:  logger,
	}
}

func (m *RateLimitMiddleware) Wrap(next PluginCallFunc) PluginCallFunc {
	return func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		if !m.limiter.Allow() {
			m.logger.Warn("rate limit exceeded",
				zap.String("plugin", req.PluginName),
			)
			return nil, fmt.Errorf("rate limit exceeded")
		}

		return next(ctx, req)
	}
}

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []PluginMiddleware
}

func NewMiddlewareChain(middlewares ...PluginMiddleware) *MiddlewareChain {
	return &MiddlewareChain{middlewares: middlewares}
}

func (c *MiddlewareChain) Execute(ctx context.Context, req *PluginRequest, handler PluginCallFunc) (*PluginResponse, error) {
	finalHandler := handler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		finalHandler = c.middlewares[i].Wrap(finalHandler)
	}

	return finalHandler(ctx, req)
}
