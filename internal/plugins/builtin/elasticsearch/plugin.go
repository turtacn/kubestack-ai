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

// Package elasticsearch implements the built-in plugin for diagnosing Elasticsearch clusters.
package elasticsearch

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

// elasticsearchPlugin is the concrete implementation of the MiddlewarePlugin for Elasticsearch.
type elasticsearchPlugin struct {
	base.Plugin
	client    *elasticsearch.Client
	collector *collector
	analyzer  *analyzer
}

// New is the factory function that creates an instance of the Elasticsearch plugin.
// It initializes the base plugin, sets up the official Elasticsearch client, and
// wires together the specific collector and analyzer for this plugin.
//
// Returns:
//   interfaces.MiddlewarePlugin: A new, fully initialized Elasticsearch plugin.
//   error: An error if the Elasticsearch client fails to initialize.
func New() (interfaces.MiddlewarePlugin, error) {
	p := &elasticsearchPlugin{}
	p.Init("elasticsearch", "0.1.0", "Provides diagnostics for Elasticsearch clusters.")

	// In a real plugin, this configuration would come from a secure source.
	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		// Other options like username, password, cloud_id, etc. would go here.
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	p.client = esClient
	p.collector = newCollector(esClient, p.Log)
	p.analyzer = newAnalyzer(p.Log)

	p.Log.Info("Elasticsearch plugin initialized successfully.")
	return p, nil
}

// Diagnose orchestrates the diagnosis process for Elasticsearch by collecting
// data from various endpoints and then passing that data to the analyzer to
// identify potential issues.
func (p *elasticsearchPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting Elasticsearch diagnosis.")

	// 1. Collect data
	health, err := p.collector.CollectClusterHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect elasticsearch cluster health: %w", err)
	}
	nodeStats, err := p.collector.CollectNodesStats(ctx)
	if err != nil {
		p.Log.Warnf("Failed to collect nodes stats: %v", err)
	}

	// 2. Analyze data
	var issues []*models.Issue
	issues = append(issues, p.analyzer.AnalyzeClusterHealth(health)...)
	if nodeStats != nil {
		issues = append(issues, p.analyzer.AnalyzeNodesStats(nodeStats)...)
	}

	// 3. Assemble result
	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("es-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("Elasticsearch diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}

	return result, nil
}

// --- Interface Method Implementations ---

// Ping checks the connectivity to the Elasticsearch cluster by calling the Ping API.
func (p *elasticsearchPlugin) Ping(ctx context.Context) error {
	res, err := p.client.Ping(p.client.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ping failed with status: %s", res.Status())
	}
	return nil
}

// HealthCheck performs a detailed health check by fetching the cluster's health
// status and reports whether it is "green".
func (p *elasticsearchPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	health, err := p.collector.CollectClusterHealth(ctx)
	if err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to get cluster health: %v", err)}, nil
	}

	status := health["status"].(string)
	isHealthy := status == "green"
	return &models.HealthStatus{IsHealthy: isHealthy, Message: fmt.Sprintf("Cluster health status is '%s'.", status)}, nil
}

// GetConfiguration retrieves the cluster's settings.
func (p *elasticsearchPlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	return p.collector.CollectClusterSettings(ctx)
}

// CollectMetrics gathers key performance indicators for the cluster.
func (p *elasticsearchPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

// CollectLogs provides a placeholder implementation for log collection. A full
// implementation would involve querying the `_cat/tasks` endpoint for slow tasks
// or reading log files from each node.
func (p *elasticsearchPlugin) CollectLogs(_ context.Context, _ *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("Elasticsearch log collection is a placeholder.")
	return &models.LogData{Entries: []string{}}, nil
}

//Personal.AI order the ending
