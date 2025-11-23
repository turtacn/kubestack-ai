package storage

import (
	"errors"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// TaskState represents the current state of a task.
type TaskState string

const (
	TaskStatePending   TaskState = "PENDING"
	TaskStateRunning   TaskState = "RUNNING"
	TaskStateCompleted TaskState = "COMPLETED"
	TaskStateFailed    TaskState = "FAILED"
)

// TaskStatus holds status information about a task.
type TaskStatus struct {
	TaskID    string    `json:"task_id"`
	State     TaskState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

var (
	ErrTaskNotFound = errors.New("task not found")
)

// TaskStore defines the interface for persisting task status and results.
type TaskStore interface {
	// CreateTask creates a new task entry with initial status PENDING.
	CreateTask(taskID string) error

	// UpdateStatus updates the state of a task.
	UpdateStatus(taskID string, state TaskState) error

	// SaveResult saves the diagnosis result for a completed task.
	SaveResult(taskID string, result *models.DiagnosisResult) error

	// SaveError saves the error for a failed task.
	SaveError(taskID string, err error) error

	// GetStatus retrieves the current status of a task.
	GetStatus(taskID string) (*TaskStatus, error)

	// GetResult retrieves the diagnosis result of a completed task.
	GetResult(taskID string) (*models.DiagnosisResult, error)
}
