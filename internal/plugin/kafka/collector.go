package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// MetricsCollector Kafka metrics collector
type MetricsCollector struct {
	client sarama.Client
	admin  sarama.ClusterAdmin
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (c *MetricsCollector) SetClient(client sarama.Client, admin sarama.ClusterAdmin) {
	c.client = client
	c.admin = admin
}

func (c *MetricsCollector) Collect(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	snapshot := &plugin.MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]plugin.MetricValue),
		RawData:   make(map[string]interface{}),
	}

	brokers := c.client.Brokers()
	snapshot.Metrics["brokers_count"] = plugin.MetricValue{
		Name:      "brokers_count",
		Value:     float64(len(brokers)),
		Timestamp: snapshot.Timestamp,
	}

	topics, err := c.client.Topics()
	if err == nil {
		snapshot.Metrics["topics_count"] = plugin.MetricValue{
			Name:      "topics_count",
			Value:     float64(len(topics)),
			Timestamp: snapshot.Timestamp,
		}
	}

	// More metrics like under-replicated partitions require deeper scan
	// For simplicity, we skip heavy scanning in CollectMetrics

	return snapshot, nil
}

func (c *MetricsCollector) CollectSpecific(ctx context.Context, metricName string) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

// Helpers for extended data

func (c *MetricsCollector) GetTopicDetails(ctx context.Context) (map[string]interface{}, error) {
	topics, err := c.client.Topics()
	if err != nil {
		return nil, err
	}
	details := make(map[string]interface{})
	for _, t := range topics {
		parts, _ := c.client.Partitions(t)
		details[t] = map[string]interface{}{
			"partition_count": len(parts),
		}
	}
	return details, nil
}

func (c *MetricsCollector) GetBrokerDetails(ctx context.Context) ([]interface{}, error) {
	var details []interface{}
	for _, b := range c.client.Brokers() {
		details = append(details, map[string]interface{}{
			"id":   b.ID(),
			"addr": b.Addr(),
		})
	}
	return details, nil
}

func (c *MetricsCollector) GetAllConsumerLags(ctx context.Context) ([]map[string]interface{}, error) {
	groups, err := c.admin.ListConsumerGroups()
	if err != nil {
		return nil, err
	}

	var lags []map[string]interface{}
	// This can be slow, iterating all groups
	for group := range groups {
		// Mock logic for lag calculation as it involves offsets
		// Real implementation needs to fetch committed offsets vs high watermarks
		lags = append(lags, map[string]interface{}{
			"group": group,
			"lag":   0, // Placeholder
		})
	}
	return lags, nil
}
