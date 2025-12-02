package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
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
	config           config.NotificationConfig
	stopChan         chan struct{}
}

// NewWorker creates a new Worker.
func NewWorker(queue TaskQueue, manager interfaces.DiagnosisManager, store storage.TaskStore, notifier notification.Notifier, notifCfg config.NotificationConfig) *Worker {
	return &Worker{
		queue:            queue,
		diagnosisManager: manager,
		taskStore:        store,
		notifier:         notifier,
		config:           notifCfg,
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
	var req models.DiagnosisRequest
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
	diagCtx := context.Background()
	progressChan := make(chan interfaces.DiagnosisProgress, 100)

	go func() {
		for range progressChan {
			// In future, update progress
		}
	}()

	result, err := w.diagnosisManager.RunDiagnosis(diagCtx, &req, progressChan)

	if err != nil {
		_ = w.taskStore.SaveError(task.ID, err)
		// Assuming we don't notify on system error unless configured, or create a failed result
		// For now, only notify on DiagnosisResult with severity check
	} else {
		_ = w.taskStore.SaveResult(task.ID, result)
		w.notifyIfNeeded(diagCtx, result)
	}
}

func (w *Worker) notifyIfNeeded(ctx context.Context, result *models.DiagnosisResult) {
	if w.notifier == nil {
		return
	}

	// Logic to check severity
	shouldNotify := false

	// Convert config string to severity level or status for comparison.
	// We map Status to a priority/level.

	statusLevel := 0
	switch result.Status {
	case enum.StatusCritical:
		statusLevel = 3
	case enum.StatusWarning:
		statusLevel = 2
	case enum.StatusHealthy:
		statusLevel = 0
	case enum.StatusUnknown:
		statusLevel = 1
	}

	// Parse config threshold
	thresholdLevel := 2 // Default Warning
	thresholdStr := strings.ToLower(w.config.AlertSeverity)

	if strings.EqualFold(thresholdStr, "critical") {
		thresholdLevel = 3
	} else if strings.EqualFold(thresholdStr, "warning") {
		thresholdLevel = 2
	} else if strings.EqualFold(thresholdStr, "info") || strings.EqualFold(thresholdStr, "healthy") {
		thresholdLevel = 0
	}

	if statusLevel >= thresholdLevel {
		shouldNotify = true
	}

	if shouldNotify {
		if err := w.notifier.Notify(ctx, result); err != nil {
			log.Printf("Failed to send notification: %v", err)
		}
	}
}
