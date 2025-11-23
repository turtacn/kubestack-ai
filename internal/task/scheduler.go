package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
)

// Scheduler handles task submission and ID generation.
type Scheduler struct {
	queue     TaskQueue
	taskStore storage.TaskStore
}

// NewScheduler creates a new Scheduler.
func NewScheduler(queue TaskQueue, store storage.TaskStore) *Scheduler {
	return &Scheduler{
		queue:     queue,
		taskStore: store,
	}
}

// SubmitDiagnosisTask creates a new diagnosis task and submits it to the queue.
func (s *Scheduler) SubmitDiagnosisTask(req *models.DiagnosisRequest) (string, error) {
	taskID := uuid.New().String()

	// Create initial task state in storage
	if err := s.taskStore.CreateTask(taskID); err != nil {
		return "", fmt.Errorf("failed to create task in storage: %w", err)
	}

	task := &Task{
		ID:        taskID,
		Type:      "diagnosis",
		Payload:   req,
		CreatedAt: time.Now(),
	}

	// Enqueue the task
	// Use background context for enqueueing to ensure it happens even if request context cancels?
	// But Enqueue might respect context cancellation.
	// Let's use a timeout context for enqueue.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queue.Enqueue(ctx, task); err != nil {
		// Attempt to update status to failed if enqueue fails
		_ = s.taskStore.SaveError(taskID, fmt.Errorf("failed to enqueue: %w", err))
		return "", err
	}

	return taskID, nil
}
