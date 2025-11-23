package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/notification"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
)

// Worker consumes tasks from the queue and executes them.
type Worker struct {
	queue            TaskQueue
	diagnosisManager interfaces.DiagnosisManager
	taskStore        storage.TaskStore
	notifier         notification.Notifier
	stopChan         chan struct{}
}

// NewWorker creates a new Worker.
func NewWorker(queue TaskQueue, manager interfaces.DiagnosisManager, store storage.TaskStore, notifier notification.Notifier) *Worker {
	return &Worker{
		queue:            queue,
		diagnosisManager: manager,
		taskStore:        store,
		notifier:         notifier,
		stopChan:         make(chan struct{}),
	}
}

// Start starts the worker loop.
func (w *Worker) Start() {
	go w.run()
}

// Stop stops the worker loop.
func (w *Worker) Stop() {
	close(w.stopChan)
}

func (w *Worker) run() {
	for {
		select {
		case <-w.stopChan:
			return
		default:
			// Dequeue with timeout or use blocking dequeue with context check
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			task, err := w.queue.Dequeue(ctx)
			cancel()

			if err != nil {
				// Log error (timeout or connection issue) and retry/continue
				// In real impl, check if error is "empty queue" or actual error
				continue
			}

			w.processTask(task)
		}
	}
}

func (w *Worker) processTask(task *Task) {
	// Update status to RUNNING
	if err := w.taskStore.UpdateStatus(task.ID, storage.TaskStateRunning); err != nil {
		log.Printf("Failed to update task status to RUNNING: %v", err)
	}

	// Currently only supports "diagnosis" type
	if task.Type != "diagnosis" {
		err := fmt.Errorf("unknown task type: %s", task.Type)
		_ = w.taskStore.SaveError(task.ID, err)
		return
	}

	// Unmarshal payload
	// The payload in Task struct is interface{}, need to cast or unmarshal
	var req models.DiagnosisRequest

	// If payload is map (from JSON unmarshal to interface{}), we need to convert it back to struct
	// Or if it's already a struct (if in-memory queue).
	// For RedisQueue, it was unmarshaled from JSON to interface{}?
	// Wait, in RedisQueue.Dequeue, we unmarshal to Task struct. Task.Payload is interface{}.
	// When we unmarshal JSON to interface{}, it becomes map[string]interface{}.

	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		_ = w.taskStore.SaveError(task.ID, fmt.Errorf("failed to marshal payload: %w", err))
		return
	}
	if err := json.Unmarshal(payloadBytes, &req); err != nil {
		_ = w.taskStore.SaveError(task.ID, fmt.Errorf("failed to unmarshal payload to DiagnosisRequest: %w", err))
		return
	}

	// Execute diagnosis
	// We need a context for the diagnosis execution
	diagCtx := context.Background()
	progressChan := make(chan interfaces.DiagnosisProgress, 100)

	// Drain progress channel in background
	go func() {
		for range progressChan {
			// In future, we could update progress in TaskStore
		}
	}()

	result, err := w.diagnosisManager.RunDiagnosis(diagCtx, &req, progressChan)

	if err != nil {
		_ = w.taskStore.SaveError(task.ID, err)
		w.notifyCompletion(task.ID, "FAILED", nil, err)
	} else {
		_ = w.taskStore.SaveResult(task.ID, result)
		w.notifyCompletion(task.ID, "COMPLETED", result, nil)
	}
}

func (w *Worker) notifyCompletion(taskID, status string, result *models.DiagnosisResult, err error) {
	if w.notifier == nil {
		return
	}

	payload := &notification.NotificationPayload{
		TaskID: taskID,
		Status: status,
		Result: result,
		Error:  err,
		// To field: Retrieve from task payload or user profile if available.
	// For now, we try to see if payload has an email field, or leave it empty.
	// The EmailNotifier should handle empty To if it has a default, or error out.
	// Here we assume the Notifier implementation or configuration might handle defaults.
	// If the payload was DiagnosisRequest, we could have added an Email field there.
	// For now, we proceed with empty To.
	}

	// Ideally, the request should contain notification preferences.

	if err := w.notifier.Notify(payload); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}
