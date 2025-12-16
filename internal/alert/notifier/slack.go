package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// SlackNotifier sends notifications to Slack.
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	httpClient *http.Client
	logger     logger.Logger
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(webhookURL, channel, username string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.NewLogger("slack-notifier"),
	}
}

// Send sends a notification to Slack.
func (s *SlackNotifier) Send(ctx context.Context, msg *NotificationMessage) error {
	body, err := s.buildSlackPayload(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack api returned status: %s", resp.Status)
	}

	return nil
}

func (s *SlackNotifier) Type() string {
	return "slack"
}

func (s *SlackNotifier) buildSlackPayload(msg *NotificationMessage) ([]byte, error) {
	// Block Kit structure
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": msg.Title,
			},
		},
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": msg.Content,
			},
		},
	}

	if msg.Link != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": fmt.Sprintf("<%s|View Details>", msg.Link),
			},
		})
	}

	payload := map[string]interface{}{
		"blocks": blocks,
	}

	if s.channel != "" {
		payload["channel"] = s.channel
	}
	if s.username != "" {
		payload["username"] = s.username
	}

	return json.Marshal(payload)
}
