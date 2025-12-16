package kafka

import (
	"context"
	"fmt"
	"strings"

	"github.com/IBM/sarama"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// KafkaPlugin implementation
type KafkaPlugin struct {
	client      sarama.Client
	adminClient sarama.ClusterAdmin
	config      *plugin.ConnectionConfig
	connected   bool
	collector   *MetricsCollector
	executor    *CommandExecutor
}

func NewKafkaPlugin(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) {
	return &KafkaPlugin{
		collector: NewMetricsCollector(),
		executor:  NewCommandExecutor(),
	}, nil
}

// === Basic Information ===

func (p *KafkaPlugin) Name() string { return "Kafka Plugin" }
func (p *KafkaPlugin) Type() plugin.MiddlewareType { return plugin.MiddlewareKafka }
func (p *KafkaPlugin) Version() string { return "1.0.0" }

// === Connection Management ===

func (p *KafkaPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	brokers := []string{fmt.Sprintf("%s:%d", config.Host, config.Port)}
	if extraBrokers, ok := config.Extra["brokers"]; ok {
		brokers = strings.Split(extraBrokers, ",")
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0 // Use a reasonably new version
	saramaConfig.Net.DialTimeout = config.Timeout
	saramaConfig.Net.ReadTimeout = config.Timeout
	saramaConfig.Net.WriteTimeout = config.Timeout

	if config.Username != "" {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = config.Username
		saramaConfig.Net.SASL.Password = config.Password
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if config.TLS != nil {
		saramaConfig.Net.TLS.Enable = true
		saramaConfig.Net.TLS.Config = config.TLS.ToTLSConfig()
	}

	client, err := sarama.NewClient(brokers, saramaConfig)
	if err != nil {
		return fmt.Errorf("kafka client failed: %w", err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("kafka admin failed: %w", err)
	}

	p.client = client
	p.adminClient = admin
	p.config = config
	p.connected = true
	p.collector.SetClient(client, admin)
	p.executor.SetClient(client, admin)

	return nil
}

func (p *KafkaPlugin) Disconnect(ctx context.Context) error {
	p.connected = false
	if p.adminClient != nil {
		p.adminClient.Close()
	}
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *KafkaPlugin) Ping(ctx context.Context) error {
	if p.client == nil || p.client.Closed() {
		return fmt.Errorf("not connected")
	}
	if len(p.client.Brokers()) == 0 {
		return fmt.Errorf("no brokers available")
	}
	return nil
}

func (p *KafkaPlugin) IsConnected() bool {
	return p.connected && !p.client.Closed()
}

// === Metrics Collection ===

func (p *KafkaPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	return p.collector.Collect(ctx)
}

func (p *KafkaPlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	return p.collector.CollectSpecific(ctx, metricName)
}

// === Command Execution ===

func (p *KafkaPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	return p.executor.Execute(ctx, cmd)
}

func (p *KafkaPlugin) SupportedCommands() []plugin.CommandSpec {
	return []plugin.CommandSpec{
		{Name: "LIST TOPICS", Description: "List topics", RiskLevel: 1},
		{Name: "DESCRIBE TOPIC", Description: "Describe topic", RiskLevel: 1},
		{Name: "LIST GROUPS", Description: "List consumer groups", RiskLevel: 1},
		{Name: "DELETE TOPIC", Description: "Delete topic", RiskLevel: 5},
	}
}

// === Diagnosis Support ===

func (p *KafkaPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	data := &plugin.DiagnosticData{
		Extra: make(map[string]interface{}),
	}

	metrics, err := p.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}
	data.Metrics = metrics

	// Extended data
	data.Extra["topics"], _ = p.collector.GetTopicDetails(ctx)
	data.Extra["brokers"], _ = p.collector.GetBrokerDetails(ctx)
	data.Extra["consumer_lags"], _ = p.collector.GetAllConsumerLags(ctx)

	return data, nil
}

func (p *KafkaPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	return kafkaBuiltinRules
}

var kafkaBuiltinRules = []plugin.DiagnosisRule{
	{
		ID:          "kafka-consumer-lag-high",
		Name:        "High Consumer Lag",
		Severity:    plugin.SeverityWarning,
		Condition:   "any(extra.consumer_lags, .lag > 10000)",
		Message:     "Consumer group {{.group}} lag is {{.lag}}",
		Suggestion:  "Increase consumers or optimize processing",
	},
}
