package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRedisClient is a mock of redis.UniversalClient
type MockRedisClient struct {
	mock.Mock
	redis.UniversalClient
}

func (m *MockRedisClient) Info(ctx context.Context, section ...string) *redis.StringCmd {
	args := m.Called(ctx, section)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	args := m.Called(ctx)
	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClient) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	callArgs := m.Called(ctx, args)
	return callArgs.Get(0).(*redis.Cmd)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRedisPlugin_Lifecycle(t *testing.T) {
	// Since Initialize tries to connect to real Redis, we can't easily unit test it without
	// mocking the client creation or having a real Redis.
	// However, we can test the components individually if we inject the mock client.

	p := &RedisPlugin{}
	mockClient := new(MockRedisClient)
	p.client = mockClient

	// Test Shutdown
	mockClient.On("Close").Return(nil)
	err := p.Shutdown()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestRedisHealthChecker_Check(t *testing.T) {
	p := &RedisPlugin{}
	mockClient := new(MockRedisClient)
	p.client = mockClient
	checker := &RedisHealthChecker{plugin: p}
	ctx := context.Background()

	// Mock PING
	mockClient.On("Ping", ctx).Return(redis.NewStatusResult("PONG", nil))

	// Mock INFO replication
	mockClient.On("Info", ctx, []string{"replication"}).Return(
		redis.NewStringResult("role:master\r\nconnected_slaves:0", nil),
	)

	// Mock INFO memory
	mockClient.On("Info", ctx, []string{"memory"}).Return(
		redis.NewStringResult("used_memory:1000\r\nmaxmemory:0", nil),
	)

	status, err := checker.Check(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, plugin.HealthyLevel, status.Overall)

	// Verify PING called
	mockClient.AssertCalled(t, "Ping", ctx)
}

func TestRedisMetricParser_Parse(t *testing.T) {
	p := &RedisPlugin{}
	parser := &RedisMetricParser{plugin: p}

	data := &plugin.CollectedData{
		RawData: map[string]interface{}{
			"info": "used_memory:1024\r\nconnected_clients:10\r\nkeyspace_hits:100\r\nkeyspace_misses:20",
		},
	}

	metrics, err := parser.Parse(context.Background(), data)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	assert.Equal(t, int64(1024), metrics.Metrics["memory_used_bytes"].Value)
	assert.Equal(t, int64(10), metrics.Metrics["connected_clients"].Value)
	assert.InDelta(t, 0.833, metrics.Metrics["hit_rate"].Value, 0.001)
}

func TestRedisHealthChecker_ReplicationSlave(t *testing.T) {
	p := &RedisPlugin{}
	mockClient := new(MockRedisClient)
	p.client = mockClient
	checker := &RedisHealthChecker{plugin: p}
	ctx := context.Background()

	// Mock INFO replication for lagging slave
	mockClient.On("Info", ctx, []string{"replication"}).Return(
		redis.NewStringResult("role:slave\r\nmaster_link_status:up\r\nmaster_last_io_seconds_ago:20", nil),
	)

	result := checker.checkReplication(ctx)
	assert.Equal(t, plugin.DegradedLevel, result.Status)
	assert.Contains(t, result.Message, "High replication lag")
}
