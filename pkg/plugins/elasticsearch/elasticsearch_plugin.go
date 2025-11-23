package elasticsearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

type ElasticsearchPlugin struct {
	config *ESConfig
	client *elasticsearch.Client
	logger *zap.Logger
}

type ESConfig struct {
	Addresses []string
	Username  string
	Password  string
	TLS       *tls.Config
}

func (p *ElasticsearchPlugin) Name() string { return "elasticsearch" }
func (p *ElasticsearchPlugin) Version() string { return "1.0.0" }
func (p *ElasticsearchPlugin) Description() string { return "Elasticsearch diagnostic plugin" }
func (p *ElasticsearchPlugin) SupportedMiddlewareVersions() []string {
	return []string{"7.x", "8.x"}
}

func (p *ElasticsearchPlugin) Initialize(config *plugin.PluginConfig) error {
	p.logger = zap.L().With(zap.String("plugin", "elasticsearch"))
	var esConf ESConfig
	if err := mapstructure.Decode(config.Settings, &esConf); err != nil {
		return err
	}

	cfg := elasticsearch.Config{
		Addresses: esConf.Addresses,
		Username:  esConf.Username,
		Password:  esConf.Password,
		Transport: &http.Transport{TLSClientConfig: esConf.TLS},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("ES client creation failed: %w", err)
	}

	res, err := client.Ping()
	if err != nil {
		return fmt.Errorf("ES ping failed: %w", err)
	}
	if res.IsError() {
		return fmt.Errorf("ES ping returned error: %s", res.String())
	}

	p.client = client
	p.config = &esConf
	return nil
}

func (p *ElasticsearchPlugin) Shutdown() error {
	return nil
}

func (p *ElasticsearchPlugin) Collector() plugin.DataCollector {
	return &ElasticsearchDataCollector{plugin: p}
}

func (p *ElasticsearchPlugin) Parser() plugin.MetricParser {
	return &ElasticsearchMetricParser{plugin: p}
}

func (p *ElasticsearchPlugin) HealthChecker() plugin.HealthChecker {
	return &ElasticsearchHealthChecker{plugin: p}
}

// ElasticsearchDataCollector Implementation
type ElasticsearchDataCollector struct {
	plugin *ElasticsearchPlugin
}

func (c *ElasticsearchDataCollector) Collect(ctx context.Context, target *plugin.Target) (*plugin.CollectedData, error) {
	clusterStats, err := c.plugin.client.Cluster.Stats(c.plugin.client.Cluster.Stats.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	clusterData := c.parseJSONResponse(clusterStats.Body)

	nodesStats, err := c.plugin.client.Nodes.Stats(c.plugin.client.Nodes.Stats.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	nodesData := c.parseJSONResponse(nodesStats.Body)

	health, err := c.plugin.client.Cluster.Health(c.plugin.client.Cluster.Health.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	healthData := c.parseJSONResponse(health.Body)

	return &plugin.CollectedData{
		PluginName: "elasticsearch",
		Target:     target,
		Timestamp:  time.Now(),
		RawData: map[string]interface{}{
			"cluster_stats": clusterData,
			"nodes_stats":   nodesData,
			"health":        healthData,
		},
	}, nil
}

func (c *ElasticsearchDataCollector) parseJSONResponse(body io.ReadCloser) map[string]interface{} {
	defer body.Close()
	var data map[string]interface{}
	_ = json.NewDecoder(body).Decode(&data)
	return data
}

func (c *ElasticsearchDataCollector) SupportedDataSources() []plugin.DataSourceType {
	return []plugin.DataSourceType{plugin.DataSourceAPI}
}

// ElasticsearchMetricParser Implementation
type ElasticsearchMetricParser struct {
	plugin *ElasticsearchPlugin
}

func (p *ElasticsearchMetricParser) Parse(ctx context.Context, data *plugin.CollectedData) (*plugin.ParsedMetrics, error) {
	health := data.RawData["health"].(map[string]interface{})
	clusterStatus, _ := health["status"].(string)

	activeShards, _ := health["active_shards_percent_as_number"].(float64)

	// Additional metrics from nodes_stats if needed (e.g. latency)
	// Simplified here

	metrics := map[string]*plugin.MetricValue{
		"cluster_status":        {Value: clusterStatus, Unit: "enum"},
		"active_shards_percent": {Value: activeShards, Unit: "percent"},
	}

	return &plugin.ParsedMetrics{
		PluginName: "elasticsearch",
		Timestamp:  time.Now(),
		Metrics:    metrics,
	}, nil
}

func (p *ElasticsearchMetricParser) AvailableMetrics() []plugin.MetricDefinition {
	return []plugin.MetricDefinition{
		{Name: "cluster_status"},
		{Name: "active_shards_percent"},
	}
}

// ElasticsearchHealthChecker Implementation
type ElasticsearchHealthChecker struct {
	plugin *ElasticsearchPlugin
}

func (c *ElasticsearchHealthChecker) Check(ctx context.Context, target *plugin.Target) (*plugin.HealthStatus, error) {
	healthResult := c.checkClusterHealth(ctx)
	diskResult := c.checkDiskWatermark(ctx)

	items := []*plugin.HealthCheckResult{healthResult, diskResult}
	overall := c.calculateOverallHealth(items)

	return &plugin.HealthStatus{
		PluginName: "elasticsearch",
		Overall:    overall,
		Items:      items,
		Timestamp:  time.Now(),
	}, nil
}

func (c *ElasticsearchHealthChecker) checkClusterHealth(ctx context.Context) *plugin.HealthCheckResult {
	res, err := c.plugin.client.Cluster.Health(c.plugin.client.Cluster.Health.WithContext(ctx))
	if err != nil {
		return &plugin.HealthCheckResult{Name: "health", Status: plugin.UnhealthyLevel, Message: err.Error()}
	}
	defer res.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return &plugin.HealthCheckResult{Name: "health", Status: plugin.UnhealthyLevel, Message: "Parse error"}
	}

	status, _ := health["status"].(string)

	var level plugin.HealthLevel
	switch status {
	case "green":
		level = plugin.HealthyLevel
	case "yellow":
		level = plugin.DegradedLevel
	case "red":
		level = plugin.UnhealthyLevel
	default:
		level = plugin.UnhealthyLevel
	}

	return &plugin.HealthCheckResult{Name: "health", Status: level, Message: status}
}

func (c *ElasticsearchHealthChecker) checkDiskWatermark(ctx context.Context) *plugin.HealthCheckResult {
	// Call Nodes Stats for fs info
	res, err := c.plugin.client.Nodes.Stats(
		c.plugin.client.Nodes.Stats.WithContext(ctx),
		c.plugin.client.Nodes.Stats.WithMetric("fs"),
	)
	if err != nil {
		return &plugin.HealthCheckResult{Name: "disk", Status: plugin.UnhealthyLevel, Message: err.Error()}
	}
	defer res.Body.Close()

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return &plugin.HealthCheckResult{Name: "disk", Status: plugin.UnhealthyLevel, Message: "Parse error"}
	}

	nodes, ok := stats["nodes"].(map[string]interface{})
	if !ok {
		return &plugin.HealthCheckResult{Name: "disk", Status: plugin.UnhealthyLevel, Message: "No nodes data"}
	}

	for nodeID, nodeData := range nodes {
		node := nodeData.(map[string]interface{})
		fs, ok := node["fs"].(map[string]interface{})
		if !ok {
			continue
		}
		total, ok1 := fs["total"].(map[string]interface{})
		if !ok1 {
			continue
		}

		totalBytes, _ := total["total_in_bytes"].(float64)
		availBytes, _ := total["available_in_bytes"].(float64)

		if totalBytes > 0 {
			usage := 1.0 - (availBytes / totalBytes)
			if usage > 0.90 {
				return &plugin.HealthCheckResult{
					Name: "disk",
					Status: plugin.UnhealthyLevel,
					Message: fmt.Sprintf("Node %s disk usage critical: %.2f%%", nodeID, usage*100),
				}
			} else if usage > 0.85 {
				return &plugin.HealthCheckResult{
					Name: "disk",
					Status: plugin.DegradedLevel,
					Message: fmt.Sprintf("Node %s disk usage high: %.2f%%", nodeID, usage*100),
				}
			}
		}
	}

	return &plugin.HealthCheckResult{Name: "disk", Status: plugin.HealthyLevel}
}

func (c *ElasticsearchHealthChecker) calculateOverallHealth(items []*plugin.HealthCheckResult) plugin.HealthLevel {
	overall := plugin.HealthyLevel
	for _, item := range items {
		if item.Status > overall {
			overall = item.Status
		}
	}
	return overall
}

func (c *ElasticsearchHealthChecker) CheckItems() []plugin.HealthCheckItem {
	return []plugin.HealthCheckItem{
		{Name: "health"},
		{Name: "disk"},
	}
}

func init() {
	plugin.RegisterPlugin(&ElasticsearchPluginFactory{})
}

type ElasticsearchPluginFactory struct{}

func (f *ElasticsearchPluginFactory) Create() plugin.Plugin {
	return &ElasticsearchPlugin{}
}

func (f *ElasticsearchPluginFactory) Metadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:       "elasticsearch",
		Version:    "1.0.0",
		APIVersion: "v1",
		Description: "Elasticsearch plugin",
	}
}
