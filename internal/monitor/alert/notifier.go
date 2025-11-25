package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

// NotifierChannel defines the interface for notification channels
type NotifierChannel interface {
	Send(ctx context.Context, alert *types.Alert) error
	Name() string
}

// Notifier manages multiple notification channels
type Notifier struct {
	channels map[string]NotifierChannel
	mu       sync.RWMutex
	log      logger.Logger
}

// NewNotifier creates a new notifier manager
func NewNotifier(log logger.Logger) *Notifier {
	return &Notifier{
		channels: make(map[string]NotifierChannel),
		log:      log,
	}
}

// Register registers a notification channel
func (n *Notifier) Register(channel NotifierChannel) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.channels[channel.Name()] = channel
}

// Send sends an alert to specified channels
func (n *Notifier) Send(ctx context.Context, alert *types.Alert, channelNames []string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(channelNames))

	for _, name := range channelNames {
		channel, exists := n.channels[name]
		if !exists {
			n.log.Warnf("Notification channel not found: %s", name)
			continue
		}

		wg.Add(1)
		go func(ch NotifierChannel) {
			defer wg.Done()

			// Retry logic
			err := n.sendWithRetry(ctx, ch, alert, 3)
			if err != nil {
				errCh <- fmt.Errorf("[%s] Send failed: %w", ch.Name(), err)
			}
		}(channel)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("some channels failed to send: %v", errs)
	}

	return nil
}

// sendWithRetry attempts to send an alert with exponential backoff
func (n *Notifier) sendWithRetry(ctx context.Context, channel NotifierChannel, alert *types.Alert, maxRetries int) error {
	var lastErr error
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		err := channel.Send(ctx, alert)
		if err == nil {
			return nil
		}

		lastErr = err
		n.log.Warnf("[%s] Send failed (retry %d/%d): %v", channel.Name(), i+1, maxRetries, err)

		// Exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}
