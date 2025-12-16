package notifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// DingTalkNotifier sends notifications to DingTalk.
type DingTalkNotifier struct {
	webhookURL string
	secret     string
	httpClient *http.Client
	logger     logger.Logger
}

// NewDingTalkNotifier creates a new DingTalkNotifier.
func NewDingTalkNotifier(webhookURL, secret string) *DingTalkNotifier {
	return &DingTalkNotifier{
		webhookURL: webhookURL,
		secret:     secret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.NewLogger("dingtalk-notifier"),
	}
}

// Send sends a notification to DingTalk.
func (d *DingTalkNotifier) Send(ctx context.Context, msg *NotificationMessage) error {
	body, err := d.buildMarkdownBody(msg)
	if err != nil {
		return err
	}

	targetURL := d.webhookURL
	if d.secret != "" {
		timestamp, sign := d.generateSignature()
		// Check if URL already has query params
		separator := "?"
		if strings.Contains(targetURL, "?") {
			separator = "&"
		}
		targetURL = fmt.Sprintf("%s%stimestamp=%s&sign=%s", targetURL, separator, timestamp, sign)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk api returned status: %s", resp.Status)
	}

	// Could parse response to check "errcode": 0

	return nil
}

func (d *DingTalkNotifier) Type() string {
	return "dingtalk"
}

func (d *DingTalkNotifier) generateSignature() (string, string) {
	ts := time.Now().UnixMilli()
	timestamp := strconv.FormatInt(ts, 10)
	stringToSign := timestamp + "\n" + d.secret

	h := hmac.New(sha256.New, []byte(d.secret))
	h.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
	sign = url.QueryEscape(sign)
	return timestamp, sign
}

func (d *DingTalkNotifier) buildMarkdownBody(msg *NotificationMessage) ([]byte, error) {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": msg.Title,
			"text":  msg.Content,
		},
		"at": map[string]interface{}{
			"isAtAll": false,
		},
	}
	return json.Marshal(payload)
}
