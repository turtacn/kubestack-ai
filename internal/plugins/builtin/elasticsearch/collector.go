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

package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// collector is responsible for gathering data from an Elasticsearch cluster via its REST API.
type collector struct {
	client *elasticsearch.Client
	log    logger.Logger
}

// newCollector creates a new Elasticsearch data collector.
func newCollector(client *elasticsearch.Client, log logger.Logger) *collector {
	return &collector{client: client, log: log}
}

// apiToMap is a generic helper to perform an API call and decode the JSON response into a map.
func (c *collector) apiToMap(ctx context.Context, apiCall func() (*esapi.Response, error)) (map[string]interface{}, error) {
	res, err := apiCall()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("api call failed with status: %s", res.Status())
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}
	return data, nil
}

// CollectClusterHealth fetches the cluster health status (green, yellow, or red) and related metrics.
func (c *collector) CollectClusterHealth(ctx context.Context) (map[string]interface{}, error) {
	c.log.Info("Collecting Elasticsearch cluster health.")
	return c.apiToMap(ctx, func() (*esapi.Response, error) {
		return c.client.Cluster.Health(c.client.Cluster.Health.WithContext(ctx))
	})
}

// CollectNodesStats fetches detailed statistics for all nodes, including JVM, OS, and thread pool information.
func (c *collector) CollectNodesStats(ctx context.Context) (map[string]interface{}, error) {
	c.log.Info("Collecting Elasticsearch nodes stats.")
	// We can specify which metrics we want, e.g., jvm, os, process, thread_pool.
	return c.apiToMap(ctx, func() (*esapi.Response, error) {
		return c.client.Nodes.Stats(
			c.client.Nodes.Stats.WithContext(ctx),
			c.client.Nodes.Stats.WithMetric("jvm", "os", "process", "thread_pool"),
		)
	})
}

// CollectClusterSettings fetches the persistent and transient settings of the cluster.
func (c *collector) CollectClusterSettings(ctx context.Context) (*models.ConfigData, error) {
	c.log.Info("Collecting Elasticsearch cluster settings.")
	settingsMap, err := c.apiToMap(ctx, func() (*esapi.Response, error) {
		return c.client.Cluster.GetSettings(c.client.Cluster.GetSettings.WithContext(ctx))
	})
	if err != nil {
		return nil, err
	}

	// The settings are nested. We can flatten them for the ConfigData model.
	configMap := make(map[string]string)
	for group, settings := range settingsMap {
		if settingsGroup, ok := settings.(map[string]interface{}); ok {
			for key, value := range settingsGroup {
				configMap[fmt.Sprintf("%s.%s", group, key)] = fmt.Sprintf("%v", value)
			}
		}
	}
	return &models.ConfigData{Data: configMap}, nil
}

// CollectMetrics derives key metrics from the various stats endpoints.
func (c *collector) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	c.log.Info("Collecting and deriving Elasticsearch metrics.")
	health, err := c.CollectClusterHealth(ctx)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})

	// Health metrics
	if val, ok := health["number_of_nodes"]; ok {
		metrics["number_of_nodes"] = val
	}
	if val, ok := health["number_of_data_nodes"]; ok {
		metrics["number_of_data_nodes"] = val
	}
	if val, ok := health["active_shards"]; ok {
		metrics["active_shards"] = val
	}
	if val, ok := health["unassigned_shards"]; ok {
		metrics["unassigned_shards"] = val
	}
	if val, ok := health["relocating_shards"]; ok {
		metrics["relocating_shards"] = val
	}
	if val, ok := health["initializing_shards"]; ok {
		metrics["initializing_shards"] = val
	}

	// TODO: Extract key metrics from NodesStats, such as JVM heap usage, CPU load, and thread pool rejections.

	return &models.MetricsData{Data: metrics}, nil
}

// TODO: Implement CollectIndicesStats (`/_stats` or `/_cat/indices`) for index-level metrics like doc count, store size, and query/indexing rates.
// TODO: Implement slow log collection. This requires enabling and configuring the slow log on the cluster itself, then querying the relevant log files or system indices.

//Personal.AI order the ending
