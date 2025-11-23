package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
	"github.com/stretchr/testify/assert"
)

// TestEnqueueDequeue requires a running Redis instance.
// For unit tests without Redis, we should use miniredis or mock.
// Since we don't have miniredis in dependencies and I cannot install it easily,
// I will skip this test if Redis is not available or mock the Redis client if possible.
// Actually, `go-redis` interface is hard to mock without an interface wrapper.
// I'll assume for this environment we might not have redis.
// But the task asks for integration tests.
// I will write the test but check connection first.

func TestEnqueueDequeue(t *testing.T) {
	// Skip if no redis
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer client.Close()

	queueName := "test_queue"
	queue := task.NewRedisQueue("localhost:6379", "", 0, queueName)
	defer queue.Close()

	// Clear queue
	client.Del(ctx, queueName)

	// Action: Enqueue
	taskID := "task-1"
	t1 := &task.Task{ID: taskID, Type: "diagnosis", CreatedAt: time.Now()}
	err := queue.Enqueue(ctx, t1)
	assert.NoError(t, err)

	// Action: Dequeue
	receivedTask, err := queue.Dequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, taskID, receivedTask.ID)
}
