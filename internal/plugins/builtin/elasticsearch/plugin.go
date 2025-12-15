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
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

const (
	IssueTitleIndexReadOnly = "Index Read-Only Block"
	IssueTitleCacheHigh     = "High Cache Usage"
)

// elasticsearchPlugin is the concrete implementation of the MiddlewarePlugin for Elasticsearch.
type elasticsearchPlugin struct {
	base.Plugin
	client    *elasticsearch.Client
	collector *collector
	analyzer  *analyzer
	fixer     *base.FixExecutor
}

func New() (interfaces.MiddlewarePlugin, error) {
	p := &elasticsearchPlugin{}
	p.Plugin.Init("elasticsearch", "0.1.0", "Provides diagnostics for Elasticsearch clusters.")

	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	p.client = esClient
	p.collector = newCollector(esClient, p.Log)
	p.analyzer = newAnalyzer(p.Log)
	p.fixer = base.NewFixExecutor(p.Log)

	p.Log.Info("Elasticsearch plugin initialized successfully.")
	return p, nil
}

// Init shadows base.Plugin.Init
func (p *elasticsearchPlugin) Init(cfg *config.PluginConfig) error {
	// Real world: use config to re-init client
	return nil
}

func (p *elasticsearchPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting Elasticsearch diagnosis.")
	health, err := p.collector.CollectClusterHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect elasticsearch cluster health: %w", err)
	}
	nodeStats, err := p.collector.CollectNodesStats(ctx)
	if err != nil {
		p.Log.Warnf("Failed to collect nodes stats: %v", err)
	}
	var issues []*models.Issue
	issues = append(issues, p.analyzer.AnalyzeClusterHealth(health)...)
	if nodeStats != nil {
		issues = append(issues, p.analyzer.AnalyzeNodesStats(nodeStats)...)
	}
	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("es-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("Elasticsearch diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}
	return result, nil
}

// --- Interface Method Implementations ---

func (p *elasticsearchPlugin) Ping(ctx context.Context, target string) error {
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

func (p *elasticsearchPlugin) HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error) {
	health, err := p.collector.CollectClusterHealth(ctx)
	if err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to get cluster health: %v", err)}, nil
	}
	status := health["status"].(string)
	isHealthy := status == "green"
	return &models.HealthStatus{IsHealthy: isHealthy, Message: fmt.Sprintf("Cluster health status is '%s'.", status)}, nil
}

func (p *elasticsearchPlugin) CollectConfig(ctx context.Context, target string) (*models.ConfigData, error) {
	return p.collector.CollectClusterSettings(ctx)
}

func (p *elasticsearchPlugin) CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

func (p *elasticsearchPlugin) CollectLogs(ctx context.Context, target string, _ *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("Elasticsearch log collection is a placeholder.")
	return &models.LogData{Entries: []string{}}, nil
}

// --- Fix Capabilities ---

func (p *elasticsearchPlugin) CanAutoFix(issue *models.Issue) (bool, *models.FixAction) {
	switch issue.Title {
	case IssueTitleIndexReadOnly:
		return true, &models.FixAction{
			ID:          "fix-es-unlock-index",
			Description: "Remove read-only block from indices",
			Command:     "ES_UNLOCK_INDEX",
			Parameters:  map[string]string{"index": "_all"},
		}
	case IssueTitleCacheHigh:
		return true, &models.FixAction{
			ID:          "fix-es-clear-cache",
			Description: "Clear indices cache",
			Command:     "ES_CLEAR_CACHE",
		}
	}
	return false, nil
}

func (p *elasticsearchPlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return p.fixer.Execute(ctx, fix, func(ctx context.Context) error {
		if fix.Command == "ES_UNLOCK_INDEX" {
			// PUT /<index>/_settings {"index.blocks.read_only_allow_delete": null}
			index := fix.Parameters["index"]
			if index == "" {
				index = "_all"
			}
			body := strings.NewReader(`{"index": {"blocks": {"read_only_allow_delete": null}}}`)
			res, err := p.client.Indices.PutSettings(body, p.client.Indices.PutSettings.WithIndex(index), p.client.Indices.PutSettings.WithContext(ctx))
			if err != nil {
				return err
			}
			defer res.Body.Close()
			if res.IsError() {
				bodyBytes, _ := io.ReadAll(res.Body)
				return fmt.Errorf("failed to unlock index: %s", string(bodyBytes))
			}
			return nil
		} else if fix.Command == "ES_CLEAR_CACHE" {
			res, err := p.client.Indices.ClearCache(p.client.Indices.ClearCache.WithContext(ctx))
			if err != nil {
				return err
			}
			defer res.Body.Close()
			if res.IsError() {
				return fmt.Errorf("failed to clear cache: %s", res.Status())
			}
			return nil
		}
		return fmt.Errorf("unknown command")
	}, nil)
}

func (p *elasticsearchPlugin) ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error) {
	return true, "Assumed success", nil
}
