package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
)

// MockNotifier
type MockNotifier struct {
	NotifiedCount int
	LastResult    *models.DiagnosisResult
}

func (m *MockNotifier) Notify(ctx context.Context, result *models.DiagnosisResult) error {
	m.NotifiedCount++
	m.LastResult = result
	return nil
}

// MockDiagnosisManager
type MockDiagnosisManager struct{}

func (m *MockDiagnosisManager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progress chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	close(progress)
	return &models.DiagnosisResult{
		ID:        "test-task-1",
		Status:    enum.StatusCritical,
		Issues: []*models.Issue{
			{Title: "CPU High", Severity: enum.SeverityCritical},
		},
	}, nil
}
func (m *MockDiagnosisManager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, data *models.CollectedData) ([]*models.Issue, error) {
	return nil, nil
}
func (m *MockDiagnosisManager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	return "", nil
}
func (m *MockDiagnosisManager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	return nil, nil
}
func (m *MockDiagnosisManager) GetKnowledgeBase() *knowledge.KnowledgeBase {
	return nil
}

// MockTaskStore
type MockTaskStore struct{}

func (m *MockTaskStore) CreateTask(taskID string) error { return nil }
func (m *MockTaskStore) UpdateStatus(taskID string, state storage.TaskState) error { return nil }
func (m *MockTaskStore) SaveResult(taskID string, result *models.DiagnosisResult) error { return nil }
func (m *MockTaskStore) SaveError(taskID string, err error) error { return nil }
func (m *MockTaskStore) GetStatus(taskID string) (*storage.TaskStatus, error) { return nil, nil }
func (m *MockTaskStore) GetResult(taskID string) (*models.DiagnosisResult, error) { return nil, nil }

// LocalMockQueue
type LocalMockQueue struct {
	tasks []*task.Task
}

func (q *LocalMockQueue) Enqueue(ctx context.Context, t *task.Task) error {
	q.tasks = append(q.tasks, t)
	return nil
}

func (q *LocalMockQueue) Dequeue(ctx context.Context) (*task.Task, error) {
	if len(q.tasks) == 0 {
		return nil, nil // causes worker to loop
	}
	t := q.tasks[0]
	q.tasks = q.tasks[1:]
	return t, nil
}

func (q *LocalMockQueue) Close() error { return nil }

func TestWorker_Triggers_Notification(t *testing.T) {
	q := &LocalMockQueue{}
	notifier := &MockNotifier{}
	diagManager := &MockDiagnosisManager{}
	store := &MockTaskStore{}
	notifCfg := config.NotificationConfig{
		AlertSeverity: "Warning",
	}

	worker := task.NewWorker(q, diagManager, store, notifier, notifCfg)

	testTask := &task.Task{
		ID:   "test-task-1",
		Type: "diagnosis",
		Payload: map[string]interface{}{
			"target": "all",
		},
	}
	q.Enqueue(context.Background(), testTask)

	worker.Start()
	defer worker.Stop()

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	if notifier.NotifiedCount != 1 {
		t.Errorf("Expected 1 notification, got %d", notifier.NotifiedCount)
	}
	if notifier.LastResult != nil && notifier.LastResult.Status != enum.StatusCritical {
		t.Errorf("Expected Critical status, got %s", notifier.LastResult.Status)
	}
}
