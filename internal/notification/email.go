package notification

import (
	"fmt"
	"net/smtp"
	"strings"
)

// EmailConfig holds configuration for sending emails.
type EmailConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	From        string
	DefaultTo   string `mapstructure:"default_to"` // Added for default recipient
}

// EmailNotifier sends notifications via Email.
type EmailNotifier struct {
	config EmailConfig
}

// NewEmailNotifier creates a new EmailNotifier.
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{config: config}
}

// Notify sends the notification payload via email.
func (e *EmailNotifier) Notify(payload *NotificationPayload) error {
	recipient := payload.To
	if recipient == "" {
		recipient = e.config.DefaultTo
	}

	if recipient == "" {
		// If no recipient is specified in payload and no default, we can't send.
		// Return nil or error? If it's optional, maybe just log?
		// But here we return error to indicate failure to send.
		return fmt.Errorf("no recipient specified for email notification and no default configured")
	}

	to := []string{recipient}
	subject := fmt.Sprintf("Diagnosis Task %s %s", payload.TaskID, payload.Status)

	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(fmt.Sprintf("Task ID: %s\n", payload.TaskID))
	bodyBuilder.WriteString(fmt.Sprintf("Status: %s\n", payload.Status))

	if payload.Error != nil {
		bodyBuilder.WriteString(fmt.Sprintf("Error: %v\n", payload.Error))
	} else if payload.Result != nil {
		bodyBuilder.WriteString(fmt.Sprintf("Result Summary: %s\n", payload.Result.Summary))
		// Maybe add a link to the console if available
	}

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", strings.Join(to, ","), subject, bodyBuilder.String()))

	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)

	// Note: In production, we should probably handle cases where Auth is not required or different Auth types.
	// For now, assume PlainAuth with TLS/STARTTLS support provided by smtp.SendMail.

	err := smtp.SendMail(addr, auth, e.config.From, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
