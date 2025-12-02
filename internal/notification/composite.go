package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// CompositeNotifier sends notifications to multiple channels.
type CompositeNotifier struct {
	config  config.NotificationConfig
	logger  logger.Logger
}

func NewCompositeNotifier(cfg config.NotificationConfig) *CompositeNotifier {
	return &CompositeNotifier{
		config:  cfg,
		logger:  logger.NewLogger("notification"),
	}
}

// Notify sends a notification to all configured channels.
func (n *CompositeNotifier) Notify(ctx context.Context, result *models.DiagnosisResult) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Email Channel
	if n.config.Email.Host != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.sendEmail(result); err != nil {
				n.logger.Errorf("Failed to send email: %v", err)
				errChan <- err
			}
		}()
	}

	// Webhook/Slack Channel
	if n.config.Slack.Enabled && n.config.Slack.WebhookURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.sendSlack(result); err != nil {
				n.logger.Errorf("Failed to send slack notification: %v", err)
				errChan <- err
			}
		}()
	} else if n.config.Webhook.URL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.sendWebhook(result); err != nil {
				n.logger.Errorf("Failed to send webhook: %v", err)
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *CompositeNotifier) sendEmail(result *models.DiagnosisResult) error {
	n.logger.Infof("Sending Email for task %s to %s", result.ID, n.config.Email.DefaultTo)
	body := FormatEmailBody(result, n.config.DashboardURL)

	if n.config.Email.Host == "mock" {
		return nil
	}

	auth := smtp.PlainAuth("", n.config.Email.Username, n.config.Email.Password, n.config.Email.Host)
	to := []string{n.config.Email.DefaultTo}

	msg := []byte("To: " + n.config.Email.DefaultTo + "\r\n" +
		"Subject: Diagnosis Report: " + result.ID + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%d", n.config.Email.Host, n.config.Email.Port)
	err := smtp.SendMail(addr, auth, n.config.Email.From, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (n *CompositeNotifier) sendSlack(result *models.DiagnosisResult) error {
	n.logger.Infof("Sending Slack notification for task %s", result.ID)
	msg := FormatSlackMessage(result, n.config.DashboardURL)

	if n.config.Slack.WebhookURL == "mock" {
		return nil
	}

	payload := map[string]string{"text": msg}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(n.config.Slack.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned status: %s", resp.Status)
	}

	return nil
}

func (n *CompositeNotifier) sendWebhook(result *models.DiagnosisResult) error {
	n.logger.Infof("Sending generic webhook for task %s", result.ID)

	if n.config.Webhook.URL == "mock" {
		return nil
	}

	jsonPayload, err := json.Marshal(result)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(n.config.Webhook.URL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status: %s", resp.Status)
	}

	return nil
}
