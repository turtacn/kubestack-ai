package channels

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       []string
}

type EmailNotifier struct {
	name string
	cfg  SMTPConfig
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(name string, cfg SMTPConfig) *EmailNotifier {
	return &EmailNotifier{
		name: name,
		cfg:  cfg,
	}
}

func (e *EmailNotifier) Send(ctx context.Context, a *types.Alert) error {
	subject := fmt.Sprintf("[%s] Alert: %s (%s)", a.Severity, a.RuleName, a.Status)

	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>%s</h2>
			<p><strong>Status:</strong> %s</p>
			<p><strong>Severity:</strong> %s</p>
			<p><strong>Value:</strong> %.2f</p>
			<p><strong>Fired At:</strong> %s</p>
			<h3>Labels</h3>
			<ul>
				%s
			</ul>
			<h3>Annotations</h3>
			<ul>
				%s
			</ul>
		</body>
		</html>
	`, subject, a.Status, a.Severity, a.Value, a.FiredAt, formatMap(a.Labels), formatMap(a.Annotations))

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", strings.Join(e.cfg.To, ","), subject, body))

	auth := smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.Host)
	addr := fmt.Sprintf("%s:%d", e.cfg.Host, e.cfg.Port)

	// Note: smtp.SendMail is blocking. In production, consider using a pool or async mechanism if high throughput needed.
	// Also context cancellation is not supported by smtp.SendMail natively.
	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, e.cfg.From, e.cfg.To, msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (e *EmailNotifier) Name() string {
	return e.name
}

func formatMap(m map[string]string) string {
	var s string
	for k, v := range m {
		s += fmt.Sprintf("<li><strong>%s:</strong> %s</li>", k, v)
	}
	return s
}
