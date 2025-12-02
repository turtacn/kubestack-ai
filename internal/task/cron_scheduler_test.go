package task

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// MockTaskQueue for testing (redefined here since it's not exported from other test file if running separately)
type MockTaskQueueCron struct {
	tasks []*Task
}

func (q *MockTaskQueueCron) Enqueue(ctx context.Context, task *Task) error {
	q.tasks = append(q.tasks, task)
	return nil
}

func (q *MockTaskQueueCron) Dequeue(ctx context.Context) (*Task, error) {
	if len(q.tasks) == 0 {
		return nil, nil // Or error
	}
	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return task, nil
}

func (q *MockTaskQueueCron) Close() error {
	return nil
}

func TestCronScheduler_Start(t *testing.T) {
	cfg := config.CronConfig{
		Enabled:            true,
		InspectionSchedule: "@every 1s", // robfig cron supports this
	}

	queue := &MockTaskQueueCron{}
	logger := logger.NewLogger("test-scheduler")
	scheduler := NewCronScheduler(cfg, queue, logger)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// Wait for 2 seconds
	time.Sleep(2100 * time.Millisecond)

	if len(queue.tasks) == 0 {
		t.Errorf("Expected tasks to be enqueued, got 0")
	}
	if len(queue.tasks) < 2 {
		t.Logf("Expected at least 2 tasks, got %d", len(queue.tasks))
	}
}
