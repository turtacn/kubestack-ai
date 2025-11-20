package ai

// This file contains the data structures and schemas used for parsing and validating AI model outputs.

// DiagnosisResult is the structured output for an initial diagnosis.
type DiagnosisResult struct {
	Severity           string   `json:"severity" validate:"required,oneof=Critical High Medium Low"`
	Category           string   `json:"category" validate:"required"`
	RootCause          string   `json:"root_cause" validate:"required,min=10"`
	AffectedComponents []string `json:"affected_components" validate:"required"`
	Confidence         float64  `json:"confidence" validate:"gte=0,lte=1"`
}

// RootCause is the structured output for a detailed root cause analysis.
type RootCause struct {
	PrimaryCause       string   `json:"primary_cause" validate:"required,min=20"`
	ContributingFactors []string `json:"contributing_factors" validate:"required"`
	Evidence           []string `json:"evidence" validate:"required"`
}

// RepairStep defines a single step in a repair plan.
type RepairStep struct {
	ID          int    `json:"id" validate:"required"`
	Description string `json:"description" validate:"required"`
	Command     string `json:"command,omitempty"`
	DependsOn   []int  `json:"depends_on"`
}

// RepairPlan defines a full repair plan with steps and a rollback strategy.
type RepairPlan struct {
	Steps        []RepairStep `json:"steps" validate:"required,min=1,dive"`
	RollbackPlan string       `json:"rollback_plan" validate:"required"`
}
