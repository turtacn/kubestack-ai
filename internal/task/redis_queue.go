package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// RedisQueue implements TaskQueue using Redis.
type RedisQueue struct {
	client    *redis.Client
	queueName string
}

// NewRedisQueue creates a new RedisQueue.
func NewRedisQueue(addr, password string, db int, queueName string) *RedisQueue {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})

	return &RedisQueue{
		client:    rdb,
		queueName: queueName,
	}
}

// Enqueue adds a task to the queue.
func (q *RedisQueue) Enqueue(ctx context.Context, task *Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return q.client.RPush(ctx, q.queueName, data).Err()
}

// Dequeue retrieves a task from the queue. It blocks if the queue is empty.
func (q *RedisQueue) Dequeue(ctx context.Context) (*Task, error) {
	// BLPop returns a slice where result[0] is the key and result[1] is the value.
	// 0 timeout means block indefinitely.
	result, err := q.client.BLPop(ctx, 0, q.queueName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue task: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid response from redis")
	}

	var task Task
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// Close closes the Redis client.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}
