package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RedisTaskStore implements TaskStore using Redis.
type RedisTaskStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisTaskStore creates a new RedisTaskStore.
func NewRedisTaskStore(addr, password string, db int, ttl time.Duration) *RedisTaskStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisTaskStore{
		client: rdb,
		ttl:    ttl,
	}
}

func (s *RedisTaskStore) statusKey(taskID string) string {
	return fmt.Sprintf("task:status:%s", taskID)
}

func (s *RedisTaskStore) resultKey(taskID string) string {
	return fmt.Sprintf("task:result:%s", taskID)
}

func (s *RedisTaskStore) CreateTask(taskID string) error {
	ctx := context.Background()
	status := &TaskStatus{
		TaskID:    taskID,
		State:     TaskStatePending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, s.statusKey(taskID), data, s.ttl).Err()
}

func (s *RedisTaskStore) UpdateStatus(taskID string, state TaskState) error {
	ctx := context.Background()
	key := s.statusKey(taskID)

	// Optimistic locking or just get/update/set. For simplicity: get/update/set
	// But since we are only updating fields, maybe fetch first.
	// Actually, UpdateStatus might need to preserve CreatedAt.

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrTaskNotFound
		}
		return err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	status.State = state
	status.UpdatedAt = time.Now()

	newData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, newData, s.ttl).Err()
}

func (s *RedisTaskStore) SaveResult(taskID string, result *models.DiagnosisResult) error {
	ctx := context.Background()

	// Save result
	resData, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := s.client.Set(ctx, s.resultKey(taskID), resData, s.ttl).Err(); err != nil {
		return err
	}

	// Update status
	return s.UpdateStatus(taskID, TaskStateCompleted)
}

func (s *RedisTaskStore) SaveError(taskID string, errVal error) error {
	ctx := context.Background()
	key := s.statusKey(taskID)

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrTaskNotFound
		}
		return err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	status.State = TaskStateFailed
	status.Error = errVal.Error()
	status.UpdatedAt = time.Now()

	newData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, newData, s.ttl).Err()
}

func (s *RedisTaskStore) GetStatus(taskID string) (*TaskStatus, error) {
	ctx := context.Background()
	data, err := s.client.Get(ctx, s.statusKey(taskID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (s *RedisTaskStore) GetResult(taskID string) (*models.DiagnosisResult, error) {
	ctx := context.Background()
	data, err := s.client.Get(ctx, s.resultKey(taskID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Check if task exists
			if _, err := s.GetStatus(taskID); err == nil {
				return nil, nil // Task exists, result not ready
			}
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	var result models.DiagnosisResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
