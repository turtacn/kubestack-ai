package diagnosis

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// ReportGenerator generates reports
type ReportGenerator struct {
	templates map[ReportFormat]*template.Template
}

// ReportFormat format type
type ReportFormat string

const (
	FormatText     ReportFormat = "text"
	FormatMarkdown ReportFormat = "markdown"
	FormatJSON     ReportFormat = "json"
)

// Report structure
type Report struct {
	Title       string
	GeneratedAt time.Time
	Format      ReportFormat
	Content     string
}

func NewReportGenerator() *ReportGenerator {
	gen := &ReportGenerator{
		templates: make(map[ReportFormat]*template.Template),
	}
	gen.registerTemplates()
	return gen
}

func (g *ReportGenerator) registerTemplates() {
	g.templates[FormatMarkdown] = template.Must(template.New("markdown").Parse(markdownTemplate))
	g.templates[FormatText] = template.Must(template.New("text").Parse(textTemplate))
}

func (g *ReportGenerator) Generate(ctx context.Context, result *DiagnosisResult, format ReportFormat) (*Report, error) {
	report := &Report{
		Title:       fmt.Sprintf("%s Diagnosis Report", result.MiddlewareType),
		GeneratedAt: time.Now(),
		Format:      format,
	}

	data := g.prepareTemplateData(result)

	tmpl, ok := g.templates[format]
	if !ok {
		// Fallback to text if not found (or json handling)
		if format == FormatJSON {
			// json marshalling
			// skip for now
		}
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	report.Content = buf.String()
	return report, nil
}

func (g *ReportGenerator) prepareTemplateData(result *DiagnosisResult) map[string]interface{} {
	data := map[string]interface{}{
		"MiddlewareType": result.MiddlewareType,
		"InstanceID":     result.InstanceID,
		"StartTime":      result.StartTime.Format(time.RFC3339),
		"EndTime":        result.EndTime.Format(time.RFC3339),
		"Duration":       result.Duration.String(),
		"HealthScore":    result.HealthScore,
		"Summary":        result.Summary,
		"Issues":         result.Issues,
	}

	data["CriticalIssues"] = g.filterIssuesBySeverity(result.Issues, plugin.SeverityCritical)
	data["ErrorIssues"] = g.filterIssuesBySeverity(result.Issues, plugin.SeverityError)
	data["WarningIssues"] = g.filterIssuesBySeverity(result.Issues, plugin.SeverityWarning)

	return data
}

func (g *ReportGenerator) filterIssuesBySeverity(issues []Issue, severity plugin.Severity) []Issue {
	filtered := make([]Issue, 0)
	for _, issue := range issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	// Sort by Name for consistency
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	return filtered
}

var markdownTemplate = `# {{.MiddlewareType}} Diagnosis Report

**Instance:** {{.InstanceID}}
**Time:** {{.StartTime}}
**Duration:** {{.Duration}}
**Health Score:** {{.HealthScore}}/100

## Summary
{{.Summary}}

## Issues

{{if .CriticalIssues}}### Critical
{{range .CriticalIssues}}- **{{.Name}}**: {{.Description}} ({{.Suggestion}})
{{end}}{{end}}

{{if .ErrorIssues}}### Error
{{range .ErrorIssues}}- **{{.Name}}**: {{.Description}} ({{.Suggestion}})
{{end}}{{end}}

{{if .WarningIssues}}### Warning
{{range .WarningIssues}}- **{{.Name}}**: {{.Description}} ({{.Suggestion}})
{{end}}{{end}}
`

var textTemplate = `{{.MiddlewareType}} Diagnosis Report
===========================
Instance: {{.InstanceID}}
Score: {{.HealthScore}}

Summary: {{.Summary}}

Issues:
{{range .Issues}}[{{.Severity}}] {{.Name}}: {{.Description}}
{{end}}
`
