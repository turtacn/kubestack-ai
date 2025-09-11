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

package kafka

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/segmentio/kafka-go"
)

// collector is responsible for gathering data from a Kafka cluster.
type collector struct {
	conn *kafka.Conn
	log  logger.Logger
}

// newCollector creates a new Kafka data collector.
func newCollector(conn *kafka.Conn, log logger.Logger) *collector {
	return &collector{conn: conn, log: log}
}

// Metadata is a custom struct to hold the rich metadata fetched from Kafka.
type Metadata struct {
	Topics      []kafka.Topic
	Brokers     []kafka.Broker
	Controller  kafka.Broker
}

// CollectMetadata reads the cluster metadata, which includes topics, partitions, and brokers.
// This is the primary source of information for diagnosing cluster state.
func (c *collector) CollectMetadata(_ context.Context) (*Metadata, error) {
	c.log.Info("Collecting Kafka cluster metadata.")

	partitions, err := c.conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("failed to read partitions: %w", err)
	}

	// The kafka-go library returns a flat list of partitions. We need to group them by topic for easier analysis.
	topicMap := make(map[string][]kafka.Partition)
	for _, p := range partitions {
		topicMap[p.Topic] = append(topicMap[p.Topic], p)
	}

	topics := make([]kafka.Topic, 0, len(topicMap))
	for name, parts := range topicMap {
		topics = append(topics, kafka.Topic{Name: name, Partitions: parts})
	}

	brokers, err := c.conn.Brokers()
	if err != nil {
		return nil, fmt.Errorf("failed to read broker list: %w", err)
	}

	controller, err := c.conn.Controller()
	if err != nil {
		return nil, fmt.Errorf("failed to read controller broker: %w", err)
	}

	return &Metadata{
		Topics:     topics,
		Brokers:    brokers,
		Controller: controller,
	}, nil
}

// CollectMetrics derives simple metrics from the cluster metadata.
func (c *collector) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	c.log.Info("Collecting and deriving Kafka metrics.")
	metadata, err := c.CollectMetadata(ctx)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})
	metrics["broker_count"] = float64(len(metadata.Brokers))
	metrics["topic_count"] = float64(len(metadata.Topics))

	partitionCount := 0
	underReplicatedPartitions := 0
	for _, t := range metadata.Topics {
		partitionCount += len(t.Partitions)
		for _, p := range t.Partitions {
			// Check for under-replicated partitions (ISR count < replica count)
			if len(p.Isr) < len(p.Replicas) {
				underReplicatedPartitions++
			}
		}
	}
	metrics["partition_count"] = float64(partitionCount)
	metrics["under_replicated_partitions_count"] = float64(underReplicatedPartitions)

	// Note: Real performance metrics (throughput, latency, request rates) require a dedicated monitoring system
	// like Prometheus with JMX Exporter, as this data is not typically available via the standard client protocol.
	c.log.Info("Note: Throughput and latency metrics require a dedicated monitoring setup (e.g., JMX Exporter).")

	return &models.MetricsData{Data: metrics}, nil
}

// TODO: Implement CollectConsumerGroupLag. This is a critical metric for monitoring consumer health. It would involve:
// 1. Using `ListConsumerGroups` on an AdminClient.
// 2. For each group, using `DescribeConsumerGroups` to find members and their assignments.
// 3. For each topic partition assigned to a member, using `FetchOffset` to get the latest log-end-offset.
// 4. Comparing the latest offset with the group's committed offset to calculate the lag.

//Personal.AI order the ending
