// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package report

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// DiagnosisReport is the unified, structured output for all diagnosis operations.
// It provides a stable contract for CLI, API, and Web consumers.
type DiagnosisReport struct {
	// Version indicates the report schema version
	Version string `json:"version" yaml:"version"`

	// ID uniquely identifies this diagnosis session
	ID string `json:"id" yaml:"id"`

	// Timestamp when the diagnosis was completed
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// Target describes what was diagnosed
	Target DiagnosisTarget `json:"target" yaml:"target"`

	// Status represents the overall health status
	Status enum.DiagnosisStatus `json:"status" yaml:"status"`

	// Summary provides a high-level overview
	Summary string `json:"summary" yaml:"summary"`

	// Issues contains all identified problems
	Issues []ReportIssue `json:"issues" yaml:"issues"`

	// Metrics holds key diagnostic metrics
	Metrics map[string]interface{} `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Metadata contains additional contextual information
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// DiagnosisTarget describes what was diagnosed
type DiagnosisTarget struct {
	// Middleware type (e.g., Redis, MySQL)
	Middleware enum.MiddlewareType `json:"middleware" yaml:"middleware"`

	// Instance name
	Instance string `json:"instance" yaml:"instance"`

	// Namespace (for Kubernetes)
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

// ReportIssue represents a single issue in the diagnosis report
type ReportIssue struct {
	// ID uniquely identifies this issue
	ID string `json:"id" yaml:"id"`

	// Source indicates which analyzer found this issue
	Source string `json:"source" yaml:"source"`

	// Title is a concise description
	Title string `json:"title" yaml:"title"`

	// Severity indicates the seriousness
	Severity enum.SeverityLevel `json:"severity" yaml:"severity"`

	// Description provides detailed explanation
	Description string `json:"description" yaml:"description"`

	// Evidence contains supporting data
	Evidence []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`

	// Suggestions for resolving the issue
	Suggestions []Suggestion `json:"suggestions,omitempty" yaml:"suggestions,omitempty"`

	// Category classifies the issue type
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
}

// Evidence represents supporting data for an issue
type Evidence struct {
	// Type indicates the kind of evidence (e.g., "metric", "log", "config")
	Type string `json:"type" yaml:"type"`

	// Key identifies the specific data point
	Key string `json:"key" yaml:"key"`

	// Value is the actual evidence value
	Value interface{} `json:"value" yaml:"value"`

	// Context provides additional information
	Context string `json:"context,omitempty" yaml:"context,omitempty"`
}

// Suggestion provides actionable recommendations
type Suggestion struct {
	// ID uniquely identifies this suggestion
	ID string `json:"id" yaml:"id"`

	// Description explains the suggested action
	Description string `json:"description" yaml:"description"`

	// Priority indicates urgency
	Priority models.Priority `json:"priority" yaml:"priority"`

	// FixHint provides implementation guidance
	FixHint *FixHint `json:"fixHint,omitempty" yaml:"fixHint,omitempty"`

	// Category classifies the suggestion type
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
}

// FixHint provides guidance for automated or manual fixes
type FixHint struct {
	// CanAutoFix indicates if this can be automated
	CanAutoFix bool `json:"canAutoFix" yaml:"canAutoFix"`

	// Command is the suggested fix command (if applicable)
	Command string `json:"command,omitempty" yaml:"command,omitempty"`

	// Parameters for the fix
	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// RiskLevel indicates the risk of applying this fix
	RiskLevel string `json:"riskLevel,omitempty" yaml:"riskLevel,omitempty"`
}

// NewDiagnosisReport creates a new report with default values
func NewDiagnosisReport(id string, target DiagnosisTarget) *DiagnosisReport {
	return &DiagnosisReport{
		Version:   ReportVersion,
		ID:        id,
		Timestamp: time.Now(),
		Target:    target,
		Status:    enum.StatusHealthy,
		Issues:    make([]ReportIssue, 0),
		Metrics:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}
}

// AddIssue adds an issue to the report
func (r *DiagnosisReport) AddIssue(issue ReportIssue) {
	r.Issues = append(r.Issues, issue)
	r.updateStatus()
}

// AddIssues adds multiple issues to the report
func (r *DiagnosisReport) AddIssues(issues []ReportIssue) {
	r.Issues = append(r.Issues, issues...)
	r.updateStatus()
}

// updateStatus recalculates the overall status based on issues
func (r *DiagnosisReport) updateStatus() {
	if len(r.Issues) == 0 {
		r.Status = enum.StatusHealthy
		return
	}

	hasCritical := false
	for _, issue := range r.Issues {
		if issue.Severity == enum.SeverityCritical {
			hasCritical = true
			break
		}
	}

	if hasCritical {
		r.Status = enum.StatusCritical
	} else {
		r.Status = enum.StatusWarning
	}
}

// ToJSON converts the report to JSON format
func (r *DiagnosisReport) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %w", err)
	}
	return string(bytes), nil
}

// FromModelsIssue converts a models.Issue to a ReportIssue
func FromModelsIssue(issue *models.Issue) ReportIssue {
	reportIssue := ReportIssue{
		ID:          issue.ID,
		Source:      issue.Source,
		Title:       issue.Title,
		Severity:    issue.Severity,
		Description: issue.Description,
		Evidence:    make([]Evidence, 0),
		Suggestions: make([]Suggestion, 0),
	}

	// Convert evidence if present
	if issue.Evidence != "" {
		reportIssue.Evidence = append(reportIssue.Evidence, Evidence{
			Type:  "general",
			Key:   "evidence",
			Value: issue.Evidence,
		})
	}

	// Convert recommendations to suggestions
	for _, rec := range issue.Recommendations {
		suggestion := Suggestion{
			ID:          rec.ID,
			Description: rec.Description,
			Priority:    rec.Priority,
		}

		if rec.CanAutoFix {
			suggestion.FixHint = &FixHint{
				CanAutoFix: true,
				Command:    rec.Fix.Command,
				Parameters: rec.Fix.Parameters,
			}
		}

		reportIssue.Suggestions = append(reportIssue.Suggestions, suggestion)
	}

	return reportIssue
}

// FromModelsIssues converts multiple models.Issue to ReportIssue slice
func FromModelsIssues(issues []*models.Issue) []ReportIssue {
	reportIssues := make([]ReportIssue, 0, len(issues))
	for _, issue := range issues {
		reportIssues = append(reportIssues, FromModelsIssue(issue))
	}
	return reportIssues
}

// FromDiagnosisResult converts a models.DiagnosisResult to a DiagnosisReport
func FromDiagnosisResult(result *models.DiagnosisResult, req *models.DiagnosisRequest) *DiagnosisReport {
	target := DiagnosisTarget{
		Middleware: req.TargetMiddleware,
		Instance:   req.Instance,
		Namespace:  req.Namespace,
	}

	report := NewDiagnosisReport(result.ID, target)
	report.Timestamp = result.Timestamp
	report.Status = result.Status
	report.Summary = result.Summary
	report.AddIssues(FromModelsIssues(result.Issues))

	return report
}
