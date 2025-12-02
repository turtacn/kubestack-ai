package notification

import (
	"fmt"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// FormatEmailBody formats the diagnosis result into an HTML email body.
func FormatEmailBody(result *models.DiagnosisResult, dashboardURL string) string {
	var sb strings.Builder

	sb.WriteString("<html><body>")
	sb.WriteString(fmt.Sprintf("<h2>Diagnosis Report: %s</h2>", result.ID))
	sb.WriteString(fmt.Sprintf("<p><strong>Status:</strong> %s</p>", result.Status.String()))
	sb.WriteString(fmt.Sprintf("<p><strong>Time:</strong> %s</p>", result.Timestamp.Format("2006-01-02 15:04:05")))

	if len(result.Issues) > 0 {
		sb.WriteString("<h3>Top Issues</h3>")
		sb.WriteString("<ul>")
		for i, issue := range result.Issues {
			if i >= 3 {
				break
			}
			sb.WriteString(fmt.Sprintf("<li><strong>%s</strong>: %s (Severity: %s)</li>", issue.Title, issue.Description, issue.Severity.String()))
		}
		sb.WriteString("</ul>")
	} else {
		sb.WriteString("<p>No issues found.</p>")
	}

	if dashboardURL != "" {
		sb.WriteString(fmt.Sprintf("<p><a href=\"%s\">View Full Report</a></p>", dashboardURL))
	}

	sb.WriteString("</body></html>")
	return sb.String()
}

// FormatSlackMessage formats the diagnosis result into a Markdown/Slack message.
func FormatSlackMessage(result *models.DiagnosisResult, dashboardURL string) string {
	var sb strings.Builder

	icon := ":white_check_mark:"
	if result.Status == enum.StatusCritical {
		icon = ":rotating_light:"
	} else if result.Status == enum.StatusWarning {
		icon = ":warning:"
	}

	sb.WriteString(fmt.Sprintf("%s *Diagnosis Alert*\n", icon))
	sb.WriteString(fmt.Sprintf("*Task ID:* %s\n", result.ID))
	sb.WriteString(fmt.Sprintf("*Status:* %s\n", result.Status.String()))
	sb.WriteString(fmt.Sprintf("*Issues Found:* %d\n", len(result.Issues)))

	if len(result.Issues) > 0 {
		sb.WriteString("*Top Issues:*\n")
		for i, issue := range result.Issues {
			if i >= 3 {
				break
			}
			sb.WriteString(fmt.Sprintf("• *%s*: %s\n", issue.Title, issue.Description))
		}
	}

	if dashboardURL != "" {
		sb.WriteString(fmt.Sprintf("\n<%s|View Full Report>", dashboardURL))
	}

	return sb.String()
}
