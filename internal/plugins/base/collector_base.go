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

// Collector is a generic interface for a component that collects data.
// While the MiddlewarePlugin interface defines specific collection methods, this can be
// used for more generic, internal collector components.
type Collector interface {
	Collect(ctx context.Context) (interface{}, error)
}

// CollectorConfig holds common configuration parameters for data collectors,
// such as timeout and retry settings.
type CollectorConfig struct {
	Timeout    time.Duration
	RetryCount int
	RetryDelay time.Duration
}

// CollectorBase provides common utilities and configuration for data collection tasks.
// It can be embedded into specific collectors within a plugin (e.g., a Redis slowlog collector)
// to handle common concerns like connection management and resilient data fetching.
type CollectorBase struct {
	Log    logger.Logger
	Config *CollectorConfig
}

// NewCollectorBase creates and initializes a new CollectorBase with a logger and configuration.
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

// Retry is a high-order function that executes a given collection function with a retry mechanism.
// This is a key utility for making data collection more resilient to transient network errors.
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

// Connect is a placeholder for establishing a connection to a data source.
// Each specific collector (e.g., for Redis, MySQL) will implement its own connection logic.
// This base method can be used to encapsulate common logic like setting up TLS.
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
