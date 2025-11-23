package task

import (
	"context"
	"time"
)

// Task represents a unit of work to be processed asynchronously.
type Task struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"` // e.g., "diagnosis"
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
}

// TaskQueue defines the interface for an asynchronous task queue.
type TaskQueue interface {
	// Enqueue adds a task to the queue.
	Enqueue(ctx context.Context, task *Task) error
	// Dequeue retrieves a task from the queue. It should block if the queue is empty.
	Dequeue(ctx context.Context) (*Task, error)
	// Close closes the queue connection.
	Close() error
}
