package kafka

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IBM/sarama"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSaramaAdmin is a mock for sarama.ClusterAdmin
type MockSaramaAdmin struct {
	mock.Mock
	sarama.ClusterAdmin // Embed interface to satisfy it without implementing all methods
}

func (m *MockSaramaAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	args := m.Called()
	return args.Get(0).(map[string]sarama.TopicDetail), args.Error(1)
}
func (m *MockSaramaAdmin) DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error) {
	args := m.Called(topics)
	return args.Get(0).([]*sarama.TopicMetadata), args.Error(1)
}
func (m *MockSaramaAdmin) ListConsumerGroups() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}
func (m *MockSaramaAdmin) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	args := m.Called(group, topicPartitions)
	return args.Get(0).(*sarama.OffsetFetchResponse), args.Error(1)
}
func (m *MockSaramaAdmin) DescribeCluster() (brokers []*sarama.Broker, controllerID int32, err error) {
	args := m.Called()
	return args.Get(0).([]*sarama.Broker), int32(args.Int(1)), args.Error(2)
}
func (m *MockSaramaAdmin) Close() error {
	return m.Called().Error(0)
}

func TestKafkaMetricParser_Parse(t *testing.T) {
	p := &KafkaPlugin{}
	parser := &KafkaMetricParser{plugin: p}

	// Mock data
	committedOffsets := map[string]map[string]map[int32]int64{
		"group1": {
			"topic1": {
				0: 100, // Committed
			},
		},
	}
	topicEndOffsets := map[string]map[int32]int64{
		"topic1": {
			0: 150, // End
		},
	}
	jmx := map[string]interface{}{
		"MessagesInPerSec": 50.0,
		"UnderReplicatedPartitions": 0.0,
	}

	data := &plugin.CollectedData{
		RawData: map[string]interface{}{
			"committed_offsets": committedOffsets,
			"topic_end_offsets": topicEndOffsets,
			"jmx":               jmx,
		},
	}

	metrics, err := parser.Parse(context.Background(), data)
	assert.NoError(t, err)

	// Lag = 150 - 100 = 50
	assert.Equal(t, int64(50), metrics.Metrics["consumer_lag_total"].Value)
	assert.Equal(t, 50.0, metrics.Metrics["messages_in_per_sec"].Value)
}

func TestKafkaHealthChecker_Check(t *testing.T) {
	p := &KafkaPlugin{}
	mockAdmin := new(MockSaramaAdmin)
	p.adminClient = mockAdmin
	checker := &KafkaHealthChecker{plugin: p}

	// Mock DescribeCluster
	// We use new(sarama.Broker) as a placeholder since fields are private.
	// We only check len(brokers) > 0.
	brokers := []*sarama.Broker{new(sarama.Broker)}
	mockAdmin.On("DescribeCluster").Return(brokers, 1, nil)

	// Mock ListTopics
	mockAdmin.On("ListTopics").Return(map[string]sarama.TopicDetail{"topic1": {}}, nil)

	// Mock DescribeTopics (ISR Check)
	// Partition 0: Replicas[1,2], Isr[1,2] -> Healthy
	mockAdmin.On("DescribeTopics", []string{"topic1"}).Return([]*sarama.TopicMetadata{
		{
			Name: "topic1",
			Partitions: []*sarama.PartitionMetadata{
				{ID: 0, Replicas: []int32{1, 2}, Isr: []int32{1, 2}},
			},
		},
	}, nil)

	status, err := checker.Check(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, plugin.HealthyLevel, status.Overall)
}

func TestKafkaHealthChecker_Check_UnderReplicated(t *testing.T) {
	p := &KafkaPlugin{}
	mockAdmin := new(MockSaramaAdmin)
	p.adminClient = mockAdmin
	checker := &KafkaHealthChecker{plugin: p}

	mockAdmin.On("DescribeCluster").Return([]*sarama.Broker{new(sarama.Broker)}, 1, nil)
	mockAdmin.On("ListTopics").Return(map[string]sarama.TopicDetail{"topic1": {}}, nil)

	// Partition 0: Replicas[1,2], Isr[1] -> UnderReplicated
	mockAdmin.On("DescribeTopics", []string{"topic1"}).Return([]*sarama.TopicMetadata{
		{
			Name: "topic1",
			Partitions: []*sarama.PartitionMetadata{
				{ID: 0, Replicas: []int32{1, 2}, Isr: []int32{1}},
			},
		},
	}, nil)

	status, err := checker.Check(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, plugin.DegradedLevel, status.Overall)
	assert.Contains(t, status.Items[2].Message, "under-replicated")
}

func TestKafkaJMXCollection(t *testing.T) {
	// Start Mock JMX Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond like Jolokia
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": 200,
			"value":  123.45,
		})
	}))
	defer server.Close()

	p := &KafkaPlugin{
		config: &KafkaConfig{
			JMXEndpoints: []string{server.URL},
		},
	}
	collector := &KafkaDataCollector{plugin: p}

	metrics := collector.collectJMXMetrics(context.Background())
	assert.Equal(t, 123.45, metrics["MessagesInPerSec"])
}
