package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

func init() {
	plugin.RegisterPluginFactory("kafka", func() plugin.DiagnosticPlugin {
		return &KafkaPlugin{}
	})
}

type KafkaPlugin struct {
	client sarama.Client
	admin  sarama.ClusterAdmin
	// Configurable consumer group to monitor
	targetGroup string
}

func (p *KafkaPlugin) Name() string {
	return "kafka"
}

func (p *KafkaPlugin) SupportedTypes() []string {
	return []string{"kafka"}
}

func (p *KafkaPlugin) Version() string {
	return "1.0.0"
}

func (p *KafkaPlugin) Init(config map[string]interface{}) error {
	brokersInterface, ok := config["brokers"].([]interface{})
	if !ok {
		return fmt.Errorf("config 'brokers' is required and must be a list of strings")
	}
	var brokers []string
	for _, b := range brokersInterface {
		brokers = append(brokers, b.(string))
	}

	if group, ok := config["consumer_group"].(string); ok {
		p.targetGroup = group
	}

	conf := sarama.NewConfig()
	conf.Version = sarama.V2_8_0_0 // Default version, can be made configurable

	var err error
	p.client, err = sarama.NewClient(brokers, conf)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %w", err)
	}

	p.admin, err = sarama.NewClusterAdmin(brokers, conf)
	if err != nil {
		p.client.Close()
		return fmt.Errorf("failed to create Kafka admin: %w", err)
	}

	return nil
}

func (p *KafkaPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	result := &models.DiagnosisResult{
		Issues: []*models.Issue{},
	}

	// Check Broker Health
	if len(p.client.Brokers()) < 1 {
		result.Issues = append(result.Issues, &models.Issue{
			Title:       "Kafka Broker Connection Issue",
			Severity:    enum.SeverityCritical,
			Description: "No active brokers found",
			Source:      "KafkaPlugin",
		})
	}

	// Check Consumer Lag if group is configured
	if p.targetGroup != "" {
		issue := p.checkConsumerLag(p.targetGroup)
		if issue != nil {
			result.Issues = append(result.Issues, issue)
		}
	} else {
		// List all groups and check them (limit to first 5 to avoid overload)
		groups, err := p.admin.ListConsumerGroups()
		if err == nil {
			count := 0
			for group := range groups {
				if count >= 5 {
					break
				}
				issue := p.checkConsumerLag(group)
				if issue != nil {
					result.Issues = append(result.Issues, issue)
				}
				count++
			}
		}
	}

	return result, nil
}

func (p *KafkaPlugin) checkConsumerLag(group string) *models.Issue {
	offsets, err := p.admin.ListConsumerGroupOffsets(group, nil)
	if err != nil {
		return nil // Unable to fetch offsets, maybe group is dead or empty
	}

	var totalLag int64
	for topic, partitions := range offsets.Blocks {
		for partition, block := range partitions {
			if block.Offset == -1 {
				continue // No offset committed
			}

			// Get latest offset
			latestOffset, err := p.client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				continue
			}

			lag := latestOffset - block.Offset
			if lag > 0 {
				totalLag += lag
			}
		}
	}

	if totalLag > 1000 {
		return &models.Issue{
			Title:       fmt.Sprintf("High Consumer Lag for group %s", group),
			Severity:    enum.SeverityHigh,
			Description: fmt.Sprintf("Total lag across all partitions is %d messages", totalLag),
			Source:      "KafkaPlugin",
		}
	}

	return nil
}

func (p *KafkaPlugin) Shutdown() error {
	if p.admin != nil {
		p.admin.Close()
	}
	if p.client != nil {
		p.client.Close()
	}
	return nil
}
