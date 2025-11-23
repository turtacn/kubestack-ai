package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTimeoutMiddleware(t *testing.T) {
	logger := zap.NewNop()
	middleware := NewTimeoutMiddleware(100*time.Millisecond, logger)

	// Case 1: Success within timeout
	handler := middleware.Wrap(func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		return &PluginResponse{Data: "ok"}, nil
	})
	resp, err := handler(context.Background(), &PluginRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Data)

	// Case 2: Timeout
	handler = middleware.Wrap(func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		time.Sleep(200 * time.Millisecond)
		return &PluginResponse{Data: "ok"}, nil
	})
	_, err = handler(context.Background(), &PluginRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestRetryMiddleware(t *testing.T) {
	logger := zap.NewNop()
	middleware := NewRetryMiddleware(2, 10*time.Millisecond, logger)

	// Case 1: Success on first try
	calls := 0
	handler := middleware.Wrap(func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		calls++
		return &PluginResponse{Data: "ok"}, nil
	})
	resp, err := handler(context.Background(), &PluginRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Data)
	assert.Equal(t, 1, calls)

	// Case 2: Success on retry
	calls = 0
	handler = middleware.Wrap(func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		calls++
		if calls < 2 {
			return nil, errors.New("temp error")
		}
		return &PluginResponse{Data: "ok"}, nil
	})
	resp, err = handler(context.Background(), &PluginRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Data)
	assert.Equal(t, 2, calls)

	// Case 3: Fail after retries
	calls = 0
	handler = middleware.Wrap(func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		calls++
		return nil, errors.New("persistent error")
	})
	_, err = handler(context.Background(), &PluginRequest{})
	assert.Error(t, err)
	assert.Equal(t, 3, calls) // Initial + 2 retries
}

func TestMiddlewareChain(t *testing.T) {
	logger := zap.NewNop()
	timeout := NewTimeoutMiddleware(100*time.Millisecond, logger)
	retry := NewRetryMiddleware(1, 10*time.Millisecond, logger)
	chain := NewMiddlewareChain(retry, timeout) // Retry(Timeout(Handler))

	calls := 0
	handler := func(ctx context.Context, req *PluginRequest) (*PluginResponse, error) {
		calls++
		if calls == 1 {
			// Fail first call immediately
			return nil, errors.New("temp error")
		}
		return &PluginResponse{Data: "ok"}, nil
	}

	resp, err := chain.Execute(context.Background(), &PluginRequest{}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Data)
	assert.Equal(t, 2, calls)
}
