package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/IBM/sarama"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

type KafkaPlugin struct {
	config      *KafkaConfig
	adminClient sarama.ClusterAdmin
	client      sarama.Client
	logger      *zap.Logger
}

type KafkaConfig struct {
	Brokers      []string
	SASL         *SASLConfig
	TLS          *tls.Config
	JMXEndpoints []string // JMX HTTP endpoints (e.g., Jolokia)
}

type SASLConfig struct {
	User     string
	Password string
}

func (p *KafkaPlugin) Name() string { return "kafka" }
func (p *KafkaPlugin) Version() string { return "1.0.0" }
func (p *KafkaPlugin) Description() string { return "Kafka diagnostic plugin" }
func (p *KafkaPlugin) SupportedMiddlewareVersions() []string {
	return []string{"2.x", "3.x"}
}

func (p *KafkaPlugin) Initialize(config *plugin.PluginConfig) error {
	p.logger = zap.L().With(zap.String("plugin", "kafka"))

	var kafkaConf KafkaConfig
	if err := mapstructure.Decode(config.Settings, &kafkaConf); err != nil {
		return err
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0
	if kafkaConf.SASL != nil {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = kafkaConf.SASL.User
		saramaConfig.Net.SASL.Password = kafkaConf.SASL.Password
	}

	// Create generic client for offset queries
	client, err := sarama.NewClient(kafkaConf.Brokers, saramaConfig)
	if err != nil {
		return fmt.Errorf("kafka client creation failed: %w", err)
	}
	p.client = client

	// Create admin client for metadata
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("kafka admin client creation failed: %w", err)
	}
	p.adminClient = admin
	p.config = &kafkaConf
	return nil
}

func (p *KafkaPlugin) Shutdown() error {
	var errs []error
	if p.adminClient != nil {
		if err := p.adminClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.client != nil {
		if err := p.client.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

func (p *KafkaPlugin) Collector() plugin.DataCollector {
	return &KafkaDataCollector{plugin: p}
}

func (p *KafkaPlugin) Parser() plugin.MetricParser {
	return &KafkaMetricParser{plugin: p}
}

func (p *KafkaPlugin) HealthChecker() plugin.HealthChecker {
	return &KafkaHealthChecker{plugin: p}
}

// KafkaDataCollector Implementation
type KafkaDataCollector struct {
	plugin *KafkaPlugin
}

func (c *KafkaDataCollector) Collect(ctx context.Context, target *plugin.Target) (*plugin.CollectedData, error) {
	// 1. Topic Metadata
	topics, err := c.plugin.adminClient.ListTopics()
	if err != nil {
		return nil, err
	}

	var topicNames []string
	for topic := range topics {
		topicNames = append(topicNames, topic)
	}

	topicDetails := make(map[string]*sarama.TopicMetadata)
	if len(topicNames) > 0 {
		metadata, err := c.plugin.adminClient.DescribeTopics(topicNames)
		if err == nil {
			for _, m := range metadata {
				topicDetails[m.Name] = m
			}
		}
	}

	// 2. Consumer Group Offsets & High Watermarks
	groups, _ := c.plugin.adminClient.ListConsumerGroups()

	// Map[group]Map[topic]Map[partition]Offset
	committedOffsets := make(map[string]map[string]map[int32]int64)

	for group := range groups {
		offsets, err := c.plugin.adminClient.ListConsumerGroupOffsets(group, nil)
		if err != nil {
			continue
		}

		if committedOffsets[group] == nil {
			committedOffsets[group] = make(map[string]map[int32]int64)
		}

		for topic, parts := range offsets.Blocks {
			if committedOffsets[group][topic] == nil {
				committedOffsets[group][topic] = make(map[int32]int64)
			}
			for partID, block := range parts {
				if block.Offset != -1 {
					committedOffsets[group][topic][partID] = block.Offset
				}
			}
		}
	}

	// Get High Watermarks (Log End Offsets) for all topics/partitions found in groups
	// Map[topic]Map[partition]LogEndOffset
	topicEndOffsets := make(map[string]map[int32]int64)

	// Collect all needed topic-partitions
	for group := range committedOffsets {
		for topic, parts := range committedOffsets[group] {
			if topicEndOffsets[topic] == nil {
				topicEndOffsets[topic] = make(map[int32]int64)
			}
			for partID := range parts {
				// Fetch latest offset
				offset, err := c.plugin.client.GetOffset(topic, partID, sarama.OffsetNewest)
				if err == nil {
					topicEndOffsets[topic][partID] = offset
				}
			}
		}
	}

	// 3. JMX Metrics
	jmxMetrics := c.collectJMXMetrics(ctx)

	return &plugin.CollectedData{
		PluginName: "kafka",
		Target:     target,
		Timestamp:  time.Now(),
		RawData: map[string]interface{}{
			"topics":            topicDetails,
			"committed_offsets": committedOffsets,
			"topic_end_offsets": topicEndOffsets,
			"jmx":               jmxMetrics,
		},
	}, nil
}

func (c *KafkaDataCollector) collectJMXMetrics(ctx context.Context) map[string]interface{} {
	metrics := make(map[string]interface{})

	// If JMX endpoints configured, try to fetch
	if len(c.plugin.config.JMXEndpoints) > 0 {
		for _, endpoint := range c.plugin.config.JMXEndpoints {
			// Example Jolokia query: http://host:port/jolokia/read/kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec
			// We'll try to fetch a few standard metrics

			// Simple implementation: just one endpoint, assume it's Jolokia
			// Fetch MessagesInPerSec
			val, err := fetchJolokiaMetric(ctx, endpoint, "kafka.server", "BrokerTopicMetrics", "MessagesInPerSec", "Count")
			if err == nil {
				metrics["MessagesInPerSec"] = val
			}

			// Fetch UnderReplicatedPartitions
			val, err = fetchJolokiaMetric(ctx, endpoint, "kafka.server", "ReplicaManager", "UnderReplicatedPartitions", "Value")
			if err == nil {
				metrics["UnderReplicatedPartitions"] = val
			}

			// Break after first successful endpoint to avoid duplication for now
			if len(metrics) > 0 {
				break
			}
		}
	}
	return metrics
}

func fetchJolokiaMetric(ctx context.Context, base, domain, typeName, name, attr string) (float64, error) {
	url := fmt.Sprintf("%s/read/%s:type=%s,name=%s/%s", base, domain, typeName, name, attr)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Value interface{} `json:"value"`
		Status int `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Status != 200 {
		return 0, fmt.Errorf("bad status: %d", result.Status)
	}

	switch v := result.Value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unknown type")
	}
}

func (c *KafkaDataCollector) SupportedDataSources() []plugin.DataSourceType {
	return []plugin.DataSourceType{plugin.DataSourceAPI}
}

// KafkaMetricParser Implementation
type KafkaMetricParser struct {
	plugin *KafkaPlugin
}

func (p *KafkaMetricParser) Parse(ctx context.Context, data *plugin.CollectedData) (*plugin.ParsedMetrics, error) {
	// 1. Calculate Lag
	committedOffsets := data.RawData["committed_offsets"].(map[string]map[string]map[int32]int64)
	topicEndOffsets := data.RawData["topic_end_offsets"].(map[string]map[int32]int64)

	totalLag := int64(0)

	for _, topics := range committedOffsets {
		for topic, partitions := range topics {
			for partID, committed := range partitions {
				if endOffset, ok := topicEndOffsets[topic][partID]; ok {
					if endOffset > committed {
						totalLag += (endOffset - committed)
					}
				}
			}
		}
	}

	// 2. Parse JMX
	jmx := data.RawData["jmx"].(map[string]interface{})
	messagesInPerSec, _ := jmx["MessagesInPerSec"].(float64)
	underReplicated, _ := jmx["UnderReplicatedPartitions"].(float64)

	metrics := map[string]*plugin.MetricValue{
		"consumer_lag_total":          {Value: totalLag, Unit: "messages"},
		"messages_in_per_sec":         {Value: messagesInPerSec, Unit: "msg/s"},
		"under_replicated_partitions": {Value: int(underReplicated), Unit: "count"},
	}

	return &plugin.ParsedMetrics{
		PluginName: "kafka",
		Timestamp:  time.Now(),
		Metrics:    metrics,
	}, nil
}

func (p *KafkaMetricParser) AvailableMetrics() []plugin.MetricDefinition {
	return []plugin.MetricDefinition{
		{Name: "consumer_lag_total"},
		{Name: "messages_in_per_sec"},
		{Name: "under_replicated_partitions"},
	}
}

// KafkaHealthChecker Implementation
type KafkaHealthChecker struct {
	plugin *KafkaPlugin
}

func (c *KafkaHealthChecker) Check(ctx context.Context, target *plugin.Target) (*plugin.HealthStatus, error) {
	// 1. Check Brokers
	brokers, controllerID, err := c.plugin.adminClient.DescribeCluster()
	brokerResult := &plugin.HealthCheckResult{Name: "brokers"}

	if err != nil {
		brokerResult.Status = plugin.UnhealthyLevel
		brokerResult.Message = err.Error()
	} else {
		if len(brokers) == 0 {
			brokerResult.Status = plugin.UnhealthyLevel
			brokerResult.Message = "No brokers found"
		} else {
			brokerResult.Status = plugin.HealthyLevel
		}
	}

	// 2. Check Controller
	controllerResult := c.checkController(controllerID)

	// 3. Check ISR (In-Sync Replicas)
	// We need to fetch topic metadata for this.
	// To be efficient, we might want to pass data from Collect, but HealthCheck is usually standalone.
	// We will fetch a list of topics and check a sample or all.
	isrResult := c.checkISR()

	items := []*plugin.HealthCheckResult{brokerResult, controllerResult, isrResult}
	overall := c.calculateOverallHealth(items)

	return &plugin.HealthStatus{
		PluginName: "kafka",
		Overall:    overall,
		Items:      items,
		Timestamp:  time.Now(),
	}, nil
}

func (c *KafkaHealthChecker) checkController(controllerID int32) *plugin.HealthCheckResult {
	if controllerID == -1 {
		return &plugin.HealthCheckResult{Name: "controller", Status: plugin.UnhealthyLevel, Message: "No active controller"}
	}

	return &plugin.HealthCheckResult{Name: "controller", Status: plugin.HealthyLevel}
}

func (c *KafkaHealthChecker) checkISR() *plugin.HealthCheckResult {
	// List topics first
	topics, err := c.plugin.adminClient.ListTopics()
	if err != nil {
		return &plugin.HealthCheckResult{Name: "isr", Status: plugin.UnhealthyLevel, Message: "Failed to list topics"}
	}

	var topicNames []string
	for t := range topics {
		topicNames = append(topicNames, t)
	}

	// Describe all topics to check partitions
	meta, err := c.plugin.adminClient.DescribeTopics(topicNames)
	if err != nil {
		return &plugin.HealthCheckResult{Name: "isr", Status: plugin.UnhealthyLevel, Message: "Failed to describe topics"}
	}

	underReplicatedCount := 0
	for _, m := range meta {
		for _, p := range m.Partitions {
			if len(p.Isr) < len(p.Replicas) {
				underReplicatedCount++
			}
		}
	}

	if underReplicatedCount > 0 {
		return &plugin.HealthCheckResult{
			Name: "isr",
			Status: plugin.DegradedLevel,
			Message: fmt.Sprintf("%d partitions under-replicated", underReplicatedCount),
		}
	}

	return &plugin.HealthCheckResult{Name: "isr", Status: plugin.HealthyLevel}
}

func (c *KafkaHealthChecker) calculateOverallHealth(items []*plugin.HealthCheckResult) plugin.HealthLevel {
	overall := plugin.HealthyLevel
	for _, item := range items {
		if item.Status > overall {
			overall = item.Status
		}
	}
	return overall
}

func (c *KafkaHealthChecker) CheckItems() []plugin.HealthCheckItem {
	return []plugin.HealthCheckItem{
		{Name: "brokers"},
		{Name: "controller"},
		{Name: "isr"},
	}
}

func init() {
	plugin.RegisterPlugin(&KafkaPluginFactory{})
}

type KafkaPluginFactory struct{}

func (f *KafkaPluginFactory) Create() plugin.Plugin {
	return &KafkaPlugin{}
}

func (f *KafkaPluginFactory) Metadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:       "kafka",
		Version:    "1.0.0",
		APIVersion: "v1",
		Description: "Kafka plugin",
	}
}
