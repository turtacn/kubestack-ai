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
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// analyzer is responsible for analyzing collected Kafka data to identify issues.
type analyzer struct {
	log logger.Logger
}

// newAnalyzer creates a new Kafka data analyzer.
func newAnalyzer(log logger.Logger) *analyzer {
	return &analyzer{log: log}
}

// Analyze is the main entry point for the analyzer. It orchestrates various specialized
// analysis functions and aggregates the issues they find.
func (a *analyzer) Analyze(metadata *Metadata) []*models.Issue {
	var issues []*models.Issue
	a.log.Info("Analyzing collected Kafka cluster metadata.")

	issues = append(issues, a.analyzeClusterState(metadata)...)
	issues = append(issues, a.analyzePartitionHealth(metadata)...)
	issues = append(issues, a.analyzeTopicConfiguration(metadata)...)
	// TODO: Add calls to other analyzers, e.g., for partition balance, leader skew, etc.

	return issues
}

// analyzeClusterState checks for high-level cluster issues like a missing controller broker.
func (a *analyzer) analyzeClusterState(metadata *Metadata) []*models.Issue {
	var issues []*models.Issue

	if metadata.Controller.ID == -1 {
		issues = append(issues, &models.Issue{
			Title:    "No Active Controller Broker",
			Severity: enum.SeverityCritical,
			Evidence: "The cluster metadata reports no active controller broker (controller ID is -1).",
			Recommendations: []*models.Recommendation{{
				Description: "The Kafka cluster is missing a controller, which is essential for managing cluster state, leader elections, and metadata. This indicates a major problem, possibly a ZooKeeper connectivity issue or severe network partitioning. Check the logs on all Kafka brokers for controller election failures.",
			}},
		})
	}

	return issues
}

// analyzePartitionHealth checks for under-replicated or offline partitions, which are critical health indicators.
func (a *analyzer) analyzePartitionHealth(metadata *Metadata) []*models.Issue {
	var issues []*models.Issue
	var underReplicatedPartitions []string
	var offlinePartitions []string

	for _, topic := range metadata.Topics {
		// Internal topics are often handled differently, so we might want to ignore them for some checks.
		if topic.Name == "__consumer_offsets" {
			continue
		}
		for _, partition := range topic.Partitions {
			// Check for under-replicated partitions: when the number of in-sync replicas is less than the total number of replicas.
			if len(partition.Isr) < len(partition.Replicas) {
				partitionID := fmt.Sprintf("%s-%d", topic.Name, partition.ID)
				underReplicatedPartitions = append(underReplicatedPartitions, partitionID)
			}
			// Check for offline partitions: when a partition has no leader.
			if partition.Leader.ID == -1 {
				partitionID := fmt.Sprintf("%s-%d", topic.Name, partition.ID)
				offlinePartitions = append(offlinePartitions, partitionID)
			}
		}
	}

	if len(underReplicatedPartitions) > 0 {
		issues = append(issues, &models.Issue{
			Title:    "Under-Replicated Partitions Detected",
			Severity: enum.SeverityHigh,
			Evidence: fmt.Sprintf("Found %d under-replicated partitions. Example: %s. This means some replicas are not in sync with the leader.", len(underReplicatedPartitions), underReplicatedPartitions[0]),
			Recommendations: []*models.Recommendation{{
				Description: "Under-replicated partitions reduce fault tolerance and can lead to data loss if the leader fails. Check the health and logs of the brokers that are supposed to be hosting the out-of-sync replicas for these partitions (brokers in 'Replicas' but not in 'ISR').",
			}},
		})
	}

	if len(offlinePartitions) > 0 {
		issues = append(issues, &models.Issue{
			Title:    "Offline Partitions Detected",
			Severity: enum.SeverityCritical,
			Evidence: fmt.Sprintf("Found %d offline partitions (with no leader). Example: %s.", len(offlinePartitions), offlinePartitions[0]),
			Recommendations: []*models.Recommendation{{
				Description: "Offline partitions are unavailable for both reads and writes. This is a critical issue, often caused by all replicas for a partition being down. Check the health and connectivity of all brokers in the cluster immediately.",
			}},
		})
	}

	return issues
}

// analyzeTopicConfiguration checks for risky topic settings, like a replication factor of 1.
func (a *analyzer) analyzeTopicConfiguration(metadata *Metadata) []*models.Issue {
	var issues []*models.Issue
	var singleReplicaTopics []string

	for _, topic := range metadata.Topics {
		if topic.Name == "__consumer_offsets" {
			continue
		}
		if len(topic.Partitions) > 0 && len(topic.Partitions[0].Replicas) == 1 {
			singleReplicaTopics = append(singleReplicaTopics, topic.Name)
		}
	}

	if len(singleReplicaTopics) > 0 {
		issues = append(issues, &models.Issue{
			Title:    "Topics with Replication Factor of 1",
			Severity: enum.SeverityHigh,
			Evidence: fmt.Sprintf("Found %d topic(s) with a replication factor of 1. Example: '%s'.", len(singleReplicaTopics), singleReplicaTopics[0]),
			Recommendations: []*models.Recommendation{{
				Description: "A replication factor of 1 means there is no data redundancy. If the broker hosting the only replica for a partition fails, it will cause permanent data loss. It is highly recommended to increase the replication factor for these topics (e.g., to 3).",
			}},
		})
	}

	return issues
}

//Personal.AI order the ending
