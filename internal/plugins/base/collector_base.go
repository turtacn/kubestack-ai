// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import (
	"context"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// Collector defines a generic interface for a component that collects a specific
// piece of data. While the `interfaces.MiddlewarePlugin` defines high-level collection
// methods (e.g., `CollectMetrics`), this can be used for more granular, internal
// collector components within a plugin.
type Collector interface {
	// Collect performs the data collection and returns the result as a generic interface{}.
	Collect(ctx context.Context) (interface{}, error)
}

// CollectorConfig holds common configuration parameters for data collectors, such
// as timeout and retry settings, promoting consistent behavior across different collectors.
type CollectorConfig struct {
	// Timeout is the maximum time to allow for a single collection attempt.
	Timeout time.Duration
	// RetryCount is the number of times to retry a failed collection attempt.
	RetryCount int
	// RetryDelay is the duration to wait between retry attempts.
	RetryDelay time.Duration
}

// CollectorBase is a foundational struct that provides common utilities and
// configuration for data collection tasks. It is designed to be embedded into
// specific collector implementations within a plugin (e.g., a Redis slowlog
// collector) to handle shared concerns like logging and resilient data fetching.
type CollectorBase struct {
	// Log is a contextualized logger for the collector.
	Log logger.Logger
	// Config holds common settings like timeouts and retries.
	Config *CollectorConfig
}

// NewCollectorBase creates and initializes a new CollectorBase with a logger and
// configuration. If no configuration is provided, it applies a set of sane defaults.
//
// Parameters:
//   log (logger.Logger): The logger to be used by the collector.
//   cfg (*CollectorConfig): The configuration for the collector. Can be nil.
//
// Returns:
//   *CollectorBase: A pointer to a new, initialized CollectorBase.
func NewCollectorBase(log logger.Logger, cfg *CollectorConfig) *CollectorBase {
	// Provide default config values if none are given
	if cfg == nil {
		cfg = &CollectorConfig{
			Timeout:    30 * time.Second,
			RetryCount: 2,
			RetryDelay: 5 * time.Second,
		}
	}
	return &CollectorBase{
		Log:    log,
		Config: cfg,
	}
}

// Retry is a higher-order function that executes a given collection function
// with a retry mechanism based on the collector's configuration. This is a key
// utility for making data collection more resilient to transient network errors or
// other temporary failures.
//
// Parameters:
//   operationName (string): A descriptive name for the operation being attempted, used for logging.
//   collectFunc (func() (interface{}, error)): The function to be executed and retried upon failure.
//
// Returns:
//   interface{}: The data returned by a successful execution of `collectFunc`.
//   error: The last error returned by `collectFunc` after all retry attempts have been exhausted.
func (b *CollectorBase) Retry(operationName string, collectFunc func() (interface{}, error)) (interface{}, error) {
	var data interface{}
	var err error

	for i := 0; i <= b.Config.RetryCount; i++ {
		if i > 0 {
			b.Log.Warnf("Retrying operation '%s' (attempt %d/%d) after delay of %s...",
				operationName, i, b.Config.RetryCount, b.Config.RetryDelay)
			time.Sleep(b.Config.RetryDelay)
		}

		data, err = collectFunc()
		if err == nil {
			return data, nil // Success
		}

		b.Log.Errorf("Operation '%s' failed on attempt %d: %v", operationName, i, err)
	}

	return nil, err // Return the last error after all retries have been exhausted
}

// Connect provides a placeholder for establishing a connection to a data source.
// Specific collectors (e.g., for Redis, MySQL) should override this method with
// their actual connection logic. This base method can be used to encapsulate
// common logic like validating host and port, setting up TLS, or implementing timeouts.
//
// Parameters:
//   host (string): The hostname or IP address of the data source.
//   port (int): The port number of the data source.
//
// Returns:
//   error: An error if the connection fails (nil in this placeholder implementation).
func (b *CollectorBase) Connect(host string, port int) error {
	b.Log.Infof("Base connect method called for %s:%d. This should be overridden by a specific collector.", host, port)
	// Placeholder for common connection logic. For example:
	// - Validating host and port.
	// - Setting up a common TLS configuration.
	// - Implementing a connection timeout.
	return nil
}

// TODO: Implement other common collector utilities as needed.
// - Batching: A helper to group data points before sending them to an analysis engine.
// - RateLimiting: A helper, possibly using a token bucket algorithm, to ensure the collector doesn't overload the target system.
// - Caching: A helper to temporarily store collected data to avoid redundant collection.

//Personal.AI order the ending
