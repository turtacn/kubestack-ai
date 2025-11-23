package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/notification"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueue
type MockQueue struct {
	tasks []*task.Task
}

func (q *MockQueue) Enqueue(ctx context.Context, t *task.Task) error {
	q.tasks = append(q.tasks, t)
	return nil
}

func (q *MockQueue) Dequeue(ctx context.Context) (*task.Task, error) {
	if len(q.tasks) == 0 {
		// Block or return error to simulate timeout/empty
		time.Sleep(10 * time.Millisecond)
		return nil, context.DeadlineExceeded
	}
	t := q.tasks[0]
	q.tasks = q.tasks[1:]
	return t, nil
}
func (q *MockQueue) Close() error { return nil }

// MockManager
type MockDiagnosisManager struct {
	mock.Mock
}

func (m *MockDiagnosisManager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	args := m.Called(ctx, req, progressChan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiagnosisResult), args.Error(1)
}
func (m *MockDiagnosisManager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, collectedData *models.CollectedData) ([]*models.Issue, error) {
	return nil, nil
}
func (m *MockDiagnosisManager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	return "", nil
}
func (m *MockDiagnosisManager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	return nil, nil
}

// MockNotifier
type MockNotifier struct {
	mock.Mock
}
func (m *MockNotifier) Notify(payload *notification.NotificationPayload) error {
	m.Called(payload)
	return nil
}

func TestWorkerExecution(t *testing.T) {
	// Setup
	queue := &MockQueue{}
	store := storage.NewInMemoryTaskStore()
	manager := new(MockDiagnosisManager)
	notifier := new(MockNotifier)

	worker := task.NewWorker(queue, manager, store, notifier)

	// Prepare task
	taskID := "task-worker-1"
	store.CreateTask(taskID)
	req := &models.DiagnosisRequest{Instance: "redis-1"}
	taskObj := &task.Task{
		ID: taskID,
		Type: "diagnosis",
		Payload: req,
	}
	queue.Enqueue(context.Background(), taskObj)

	// Expectations
	manager.On("RunDiagnosis", mock.Anything, mock.Anything, mock.Anything).Return(&models.DiagnosisResult{ID: "res-1", Summary: "Ok"}, nil)
	notifier.On("Notify", mock.MatchedBy(func(p *notification.NotificationPayload) bool {
		return p.TaskID == taskID && p.Status == "COMPLETED"
	})).Return(nil)

	// Action
	go worker.Start()
	time.Sleep(100 * time.Millisecond) // Let worker process
	worker.Stop()

	// Assert
	status, err := store.GetStatus(taskID)
	assert.NoError(t, err)
	assert.Equal(t, storage.TaskStateCompleted, status.State)

	manager.AssertExpectations(t)
	notifier.AssertExpectations(t)
}
