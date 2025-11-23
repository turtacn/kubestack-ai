package storage

import (
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// InMemoryTaskStore implements TaskStore using a map in memory.
// Suitable for development and testing.
type InMemoryTaskStore struct {
	mu      sync.RWMutex
	tasks   map[string]*TaskStatus
	results map[string]*models.DiagnosisResult
}

func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		tasks:   make(map[string]*TaskStatus),
		results: make(map[string]*models.DiagnosisResult),
	}
}

func (s *InMemoryTaskStore) CreateTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[taskID] = &TaskStatus{
		TaskID:    taskID,
		State:     TaskStatePending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func (s *InMemoryTaskStore) UpdateStatus(taskID string, state TaskState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	task.State = state
	task.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryTaskStore) SaveResult(taskID string, result *models.DiagnosisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[taskID]; !ok {
		return ErrTaskNotFound
	}

	s.results[taskID] = result
	// Assuming SaveResult is called when task is completed
	s.tasks[taskID].State = TaskStateCompleted
	s.tasks[taskID].UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryTaskStore) SaveError(taskID string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	task.State = TaskStateFailed
	task.Error = err.Error()
	task.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryTaskStore) GetStatus(taskID string) (*TaskStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}

	// Return a copy to prevent race conditions
	status := *task
	return &status, nil
}

func (s *InMemoryTaskStore) GetResult(taskID string, ) (*models.DiagnosisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.results[taskID]
	if !ok {
		// If task exists but no result, it might not be completed yet
		if _, taskOk := s.tasks[taskID]; taskOk {
			return nil, nil // Or specific error like ErrResultNotReady
		}
		return nil, ErrTaskNotFound
	}

	// Return a deep copy ideally, but pointer is fine for now as results are typically immutable once saved
	// To be safe, we could marshal/unmarshal
	// For now just return the pointer
	return result, nil
}
