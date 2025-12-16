package alert

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/alert/notifier"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

type FeedbackConfig struct {
	EnabledChannels []string
}

type FeedbackProcessor struct {
	notifiers []notifier.Notifier
	config    *FeedbackConfig
	logger    logger.Logger
}

func NewFeedbackProcessor(notifiers []notifier.Notifier, config *FeedbackConfig) *FeedbackProcessor {
	return &FeedbackProcessor{
		notifiers: notifiers,
		config:    config,
		logger:    logger.NewLogger("feedback-processor"),
	}
}

func (f *FeedbackProcessor) ProcessDiagnosisResult(ctx context.Context, alert *CorrelatedAlert, result *models.DiagnosisResult) error {
	msg := f.buildNotificationMessage(alert, result)

	var wg sync.WaitGroup
	for _, n := range f.notifiers {
		wg.Add(1)
		go func(n notifier.Notifier) {
			defer wg.Done()
			if err := n.Send(ctx, msg); err != nil {
				f.logger.Errorf("Failed to send notification via %s: %v", n.Type(), err)
			}
		}(n)
	}
	wg.Wait()
	return nil
}

func (f *FeedbackProcessor) buildNotificationMessage(alert *CorrelatedAlert, result *models.DiagnosisResult) *notifier.NotificationMessage {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Instance**: %s\n", alert.Instance))
	sb.WriteString(fmt.Sprintf("**Alerts**: %s\n", alert.Summary))
	sb.WriteString(fmt.Sprintf("**Diagnosis**: %s\n", result.Summary))

	if len(result.Issues) > 0 {
		sb.WriteString("\n### Issues Found:\n")
		for _, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", issue.Title, issue.Severity, issue.Description))
			if len(issue.Recommendations) > 0 {
				sb.WriteString("  *Recommendations*:\n")
				for _, rec := range issue.Recommendations {
					sb.WriteString(fmt.Sprintf("  - %s\n", rec.Description))
				}
			}
		}
	}

	title := fmt.Sprintf("[%s] Diagnosis Report for %s", result.Status, alert.Middleware)

	return &notifier.NotificationMessage{
		Title:    title,
		Content:  sb.String(),
		Severity: result.Status.String(),
		Link:     fmt.Sprintf("http://dashboard-url/diagnosis/%s", result.ID), // TODO: Configurable URL
	}
}
