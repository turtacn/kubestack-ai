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

// Package kafka implements the built-in plugin for diagnosing Apache Kafka clusters.
package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
	"github.com/segmentio/kafka-go"
)

// kafkaPlugin is the concrete implementation of the MiddlewarePlugin for Kafka.
type kafkaPlugin struct {
	base.Plugin
	// A connection to a single broker is used for metadata queries.
	conn      *kafka.Conn
	collector *collector
	analyzer  *analyzer
}

// New is the factory function that creates an instance of the Kafka plugin. It
// initializes the base plugin, establishes a connection to a Kafka broker for
// metadata queries, and wires together the specific collector and analyzer for Kafka.
//
// Returns:
//   interfaces.MiddlewarePlugin: A new, fully initialized Kafka plugin.
//   error: An error if the initial connection to a Kafka broker fails.
func New() (interfaces.MiddlewarePlugin, error) {
	p := &kafkaPlugin{}
	p.Init("kafka", "0.1.0", "Provides diagnostics for Apache Kafka clusters.")

	// In a real plugin, broker addresses would come from a configuration source.
	brokers := []string{"localhost:9092"}

	// For metadata operations, we dial a single broker.
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka broker %s: %w", brokers[0], err)
	}

	p.conn = conn
	p.collector = newCollector(conn, p.Log)
	p.analyzer = newAnalyzer(p.Log)

	p.Log.Info("Kafka plugin initialized successfully.")
	return p, nil
}

// Diagnose orchestrates the diagnosis process for Kafka. It collects cluster
// metadata and then passes it to the analyzer to identify potential issues.
func (p *kafkaPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting Kafka diagnosis.")

	// 1. Collect data
	metadata, err := p.collector.CollectMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect kafka metadata: %w", err)
	}

	// 2. Analyze data
	issues := p.analyzer.Analyze(metadata)

	// 3. Assemble result
	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("kafka-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("Kafka diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}

	return result, nil
}

// --- Interface Method Implementations ---

// Ping checks connectivity to the Kafka cluster by attempting to read partition metadata.
func (p *kafkaPlugin) Ping(ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	p.conn.SetReadDeadline(deadline)
	_, err := p.conn.ReadPartitions()
	p.conn.SetReadDeadline(time.Time{}) // Reset deadline
	return err
}

// HealthCheck performs a basic health check by pinging the cluster.
func (p *kafkaPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	if err := p.Ping(ctx); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to get metadata from Kafka: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "Kafka is responsive."}, nil
}

// GetConfiguration provides a placeholder implementation. A full implementation would
// use the Kafka Admin API to describe cluster and broker configurations.
func (p *kafkaPlugin) GetConfiguration(_ context.Context) (*models.ConfigData, error) {
	p.Log.Info("Kafka configuration collection via client is a placeholder. Configuration is typically managed in server.properties files on each broker.")
	return &models.ConfigData{Data: make(map[string]string)}, nil
}

// CollectMetrics gathers key performance indicators for the cluster, derived from metadata.
func (p *kafkaPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

// CollectLogs provides a placeholder implementation. A full implementation would
// require reading log files from broker servers or connecting via JMX, which is
// beyond the scope of a simple client.
func (p *kafkaPlugin) CollectLogs(_ context.Context, _ *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("Kafka log collection is a placeholder. Logs are typically collected via other means (e.g., filebeat, JMX exporter).")
	return &models.LogData{Entries: []string{}}, nil
}

// Shutdown gracefully closes the underlying Kafka connection to prevent resource leaks.
func (p *kafkaPlugin) Shutdown() error {
	p.Log.Info("Shutting down Kafka plugin and closing connection.")
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

//Personal.AI order the ending
