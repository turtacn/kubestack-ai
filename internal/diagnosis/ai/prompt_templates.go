package ai

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

const DiagnosisPromptTemplate = `
You are an expert Kubernetes middleware diagnostician. Analyze the following system state:

## Context
- Plugin: {{.PluginName}}
- Timestamp: {{.Timestamp}}
- User Query: {{.UserQuery}}

## Observability Data
### System Logs (last 100 lines)
{{.SystemLogs}}

### Metrics
{{.MetricData}}

{{if .KnowledgeContext}}
## Relevant Knowledge Base
{{.KnowledgeContext}}
{{end}}

## Task
Identify the root cause and provide a structured diagnosis in the following JSON format:
{
  "severity": "Critical|High|Medium|Low",
  "category": "Performance|Availability|Configuration|Resource",
  "root_cause": "brief description",
  "affected_components": ["component1", "component2"],
  "confidence": 0.0-1.0
}
`

const RootCausePromptTemplate = `
You are a root cause analysis expert. Based on the initial diagnosis below, perform a deeper analysis to pinpoint the primary root cause.

## Initial Diagnosis
{{.InitialDiagnosis}}

## Additional Context
{{.AdditionalContext}}

## Task
Provide a detailed root cause analysis in the following JSON format:
{
  "primary_cause": "A detailed explanation of the single most likely root cause.",
  "contributing_factors": ["factor1", "factor2"],
  "evidence": ["log snippet", "metric reading"]
}
`

const RepairPromptTemplate = `
You are an experienced SRE. Based on the identified root cause, generate a safe and actionable repair plan.

## Root Cause Analysis
{{.RootCause}}

## System Constraints
{{.SystemConstraints}}

## Task
Generate a step-by-step repair plan in the following JSON format:
{
  "steps": [
    {
      "id": 1,
      "description": "Step 1 description",
      "command": "kubectl ...",
      "depends_on": []
    },
    {
      "id": 2,
      "description": "Step 2 description",
      "command": "redis-cli ...",
      "depends_on": [1]
    }
  ],
  "rollback_plan": "A description of how to revert the changes if something goes wrong."
}
`

const ExplainPromptTemplate = `
You are a technical expert. Explain the following concept or issue in a clear and concise way for a junior engineer.

## Topic
{{.Topic}}

## Context
{{.Context}}

## Task
Provide a simple explanation. Use analogies if helpful. Do not exceed 200 words.
`

const ClarifyPromptTemplate = `
You are a helpful diagnostic assistant. The current analysis has low confidence. To improve accuracy, ask the user for more specific information.

## Current Analysis
{{.CurrentAnalysis}}

## Ambiguous Areas
{{.AmbiguousAreas}}

## Task
Ask up to 3 clear, specific questions that will help resolve the ambiguity. Format them as a numbered list.
`

type PromptRenderer struct {
	templates map[string]*template.Template
}

func NewPromptRenderer() (*PromptRenderer, error) {
	templates := make(map[string]*template.Template)

	templateMap := map[string]string{
		"diagnosis":  DiagnosisPromptTemplate,
		"root_cause": RootCausePromptTemplate,
		"repair":     RepairPromptTemplate,
		"explain":    ExplainPromptTemplate,
		"clarify":    ClarifyPromptTemplate,
	}

	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },
		"truncate": func(s string, n int) string {
			if len(s) > n {
				return s[:n]
			}
			return s
		},
	}

	for name, tmplStr := range templateMap {
		tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		templates[name] = tmpl
	}

	return &PromptRenderer{templates: templates}, nil
}

func (r *PromptRenderer) Render(templateName string, data interface{}) (string, error) {
	tpl, ok := r.templates[templateName]
	if !ok {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}
	return buf.String(), nil
}
