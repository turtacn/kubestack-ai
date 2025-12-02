package task

import (
	"context"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/robfig/cron/v3"
)

// CronScheduler schedules diagnosis tasks based on cron expressions.
type CronScheduler struct {
	cron    *cron.Cron
	queue   TaskQueue
	config  config.CronConfig
	logger  logger.Logger
}

// NewCronScheduler creates a new CronScheduler.
func NewCronScheduler(cfg config.CronConfig, queue TaskQueue, logger logger.Logger) *CronScheduler {
	return &CronScheduler{
		cron:    cron.New(),
		queue:   queue,
		config:  cfg,
		logger:  logger,
	}
}

// Start starts the cron scheduler.
func (s *CronScheduler) Start() error {
	if !s.config.Enabled || s.config.InspectionSchedule == "" {
		s.logger.Info("Cron scheduler is disabled or schedule is empty.")
		return nil
	}

	s.logger.Infof("Starting cron scheduler with schedule: %s", s.config.InspectionSchedule)

	_, err := s.cron.AddFunc(s.config.InspectionSchedule, func() {
		s.logger.Info("Triggering scheduled diagnosis task.")

		task := &Task{
			ID:        "scheduled-" + time.Now().Format("20060102-150405"),
			Type:      "diagnosis",
			Payload:   map[string]string{"scope": "all"}, // Default payload
			CreatedAt: time.Now(),
		}

		if err := s.queue.Enqueue(context.Background(), task); err != nil {
			s.logger.Errorf("Failed to enqueue scheduled task: %v", err)
		} else {
			s.logger.Infof("Enqueued scheduled task: %s", task.ID)
		}
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

// Stop stops the cron scheduler.
func (s *CronScheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}
