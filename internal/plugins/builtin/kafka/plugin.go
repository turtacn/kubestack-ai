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

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
	"github.com/segmentio/kafka-go"
)

const (
	IssueTitleConsumerLag   = "High Consumer Lag"
	IssueTitleUnderReplicated = "Under Replicated Partitions"
)

// kafkaPlugin is the concrete implementation of the MiddlewarePlugin for Kafka.
type kafkaPlugin struct {
	base.Plugin
	conn      *kafka.Conn
	collector *collector
	analyzer  *analyzer
	fixer     *base.FixExecutor
}

func New() (interfaces.MiddlewarePlugin, error) {
	p := &kafkaPlugin{}
	p.Plugin.Init("kafka", "0.1.0", "Provides diagnostics for Apache Kafka clusters.")

	brokers := []string{"localhost:9092"}
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka broker %s: %w", brokers[0], err)
	}

	p.conn = conn
	p.collector = newCollector(conn, p.Log)
	p.analyzer = newAnalyzer(p.Log)
	p.fixer = base.NewFixExecutor(p.Log)

	p.Log.Info("Kafka plugin initialized successfully.")
	return p, nil
}

// Init shadows base.Plugin.Init to satisfy interface
func (p *kafkaPlugin) Init(cfg *config.PluginConfig) error {
	// In real world, update brokers from config
	return nil
}

func (p *kafkaPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting Kafka diagnosis.")
	metadata, err := p.collector.CollectMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect kafka metadata: %w", err)
	}
	issues := p.analyzer.Analyze(metadata)
	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("kafka-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("Kafka diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}
	return result, nil
}

// --- Interface Method Implementations ---

func (p *kafkaPlugin) Ping(ctx context.Context, target string) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	p.conn.SetReadDeadline(deadline)
	_, err := p.conn.ReadPartitions()
	p.conn.SetReadDeadline(time.Time{})
	return err
}

func (p *kafkaPlugin) HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error) {
	if err := p.Ping(ctx, target); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to get metadata from Kafka: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "Kafka is responsive."}, nil
}

func (p *kafkaPlugin) CollectConfig(ctx context.Context, target string) (*models.ConfigData, error) {
	p.Log.Info("Kafka configuration collection via client is a placeholder.")
	return &models.ConfigData{Data: make(map[string]string)}, nil
}

func (p *kafkaPlugin) CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

func (p *kafkaPlugin) CollectLogs(ctx context.Context, target string, _ *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("Kafka log collection is a placeholder.")
	return &models.LogData{Entries: []string{}}, nil
}

func (p *kafkaPlugin) Shutdown() error {
	p.Log.Info("Shutting down Kafka plugin and closing connection.")
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// --- Fix Capabilities ---

func (p *kafkaPlugin) CanAutoFix(issue *models.Issue) (bool, *models.FixAction) {
	switch issue.Title {
	case IssueTitleConsumerLag:
		// Example fix: Reset offset to latest (Data Loss Risk! But serves as example)
		return true, &models.FixAction{
			ID:          "fix-kafka-reset-offset",
			Description: "Reset consumer group offset to latest",
			Command:     "KAFKA_RESET_OFFSET",
			Parameters:  map[string]string{"group": "my-group", "topic": "my-topic"}, // In reality extracted from issue
		}
	case IssueTitleUnderReplicated:
		// Example fix: Trigger partition reassignment (simplified)
		return true, &models.FixAction{
			ID:          "fix-kafka-reassign-partition",
			Description: "Trigger partition reassignment for under-replicated partitions",
			Command:     "KAFKA_REASSIGN_PARTITION",
			Parameters:  map[string]string{"topic": "my-topic", "partition": "0"},
		}
	}
	return false, nil
}

func (p *kafkaPlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return p.fixer.Execute(ctx, fix, func(ctx context.Context) error {
		if fix.Command == "KAFKA_RESET_OFFSET" {
			// Mock implementation as kafka-go requires more complex setup for offset reset
			p.Log.Info("Simulating Kafka offset reset to latest.")
			return nil
		} else if fix.Command == "KAFKA_REASSIGN_PARTITION" {
			// Mock implementation for partition reassignment
			p.Log.Infof("Simulating partition reassignment for topic %s", fix.Parameters["topic"])
			return nil
		}
		return fmt.Errorf("unknown command: %s", fix.Command)
	}, nil)
}

func (p *kafkaPlugin) ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error) {
	return true, "Assumed success", nil
}
