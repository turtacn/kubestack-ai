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

// Package execution defines the types and structures for controlled AutoFix execution.
// This module implements Phase 04's requirement for centralized execution governance
// with clear safety boundaries and audit trails.
package execution

import (
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// FixPlan represents a complete plan for executing automated fixes.
// It is generated from diagnosis recommendations but includes additional
// safety checks, risk assessments, and execution strategies.
//
// Design rationale: Separates planning from execution, allowing for
// human review before actual changes are applied.
type FixPlan struct {
	// ID uniquely identifies this fix plan
	ID string `json:"id" yaml:"id"`

	// DiagnosisID links this plan to the original diagnosis
	DiagnosisID string `json:"diagnosisId" yaml:"diagnosisId"`

	// CreatedAt is the timestamp when this plan was generated
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`

	// Actions contains all fix actions to be executed
	Actions []*FixActionItem `json:"actions" yaml:"actions"`

	// ExecutionStrategy defines how actions should be executed
	// (serial, parallel, or conditional)
	ExecutionStrategy ExecutionStrategy `json:"executionStrategy" yaml:"executionStrategy"`

	// RiskAssessment contains the overall risk evaluation
	RiskAssessment *RiskAssessment `json:"riskAssessment" yaml:"riskAssessment"`

	// RequiresApproval indicates if this plan needs explicit approval
	RequiresApproval bool `json:"requiresApproval" yaml:"requiresApproval"`

	// DryRun indicates whether this is a simulation run
	DryRun bool `json:"dryRun" yaml:"dryRun"`

	// Metadata holds additional plan-level information
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// FixActionItem represents a single executable fix action within a plan.
// It wraps the base FixAction with additional execution-specific metadata.
type FixActionItem struct {
	// Sequence defines the order of execution (lower executes first)
	Sequence int `json:"sequence" yaml:"sequence"`

	// Action is the actual fix action from the diagnosis recommendation
	Action *models.FixAction `json:"action" yaml:"action"`

	// ValidationRules define checks that must pass before execution
	ValidationRules []ValidationRule `json:"validationRules,omitempty" yaml:"validationRules,omitempty"`

	// Timeout defines the maximum execution time
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// DependsOn lists action IDs that must complete successfully first
	DependsOn []string `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`

	// Category classifies the action type for dependency analysis
	Category ActionCategory `json:"category" yaml:"category"`

	// Risk contains action-specific risk assessment
	Risk *ActionRisk `json:"risk,omitempty" yaml:"risk,omitempty"`
}

// FixResult represents the outcome of executing a fix plan.
// This is the complete audit trail of what was attempted and achieved.
type FixResult struct {
	// PlanID links this result to the executed plan
	PlanID string `json:"planId" yaml:"planId"`

	// ExecutionID uniquely identifies this execution attempt
	ExecutionID string `json:"executionId" yaml:"executionId"`

	// Status indicates the overall execution outcome
	Status FixExecutionStatus `json:"status" yaml:"status"`

	// StartedAt is when execution began
	StartedAt time.Time `json:"startedAt" yaml:"startedAt"`

	// CompletedAt is when execution finished
	CompletedAt time.Time `json:"completedAt" yaml:"completedAt"`

	// ActionResults contains outcomes for each action
	ActionResults []*ActionResult `json:"actionResults" yaml:"actionResults"`

	// ValidationReport contains pre-execution validation results
	ValidationReport *ValidationReport `json:"validationReport,omitempty" yaml:"validationReport,omitempty"`

	// RollbackPerformed indicates if rollback was triggered
	RollbackPerformed bool `json:"rollbackPerformed" yaml:"rollbackPerformed"`

	// RollbackSuccess indicates if rollback succeeded (if performed)
	RollbackSuccess *bool `json:"rollbackSuccess,omitempty" yaml:"rollbackSuccess,omitempty"`

	// ErrorMessage contains the primary error if execution failed
	ErrorMessage string `json:"errorMessage,omitempty" yaml:"errorMessage,omitempty"`

	// Logs contains detailed execution logs
	Logs []ExecutionLogEntry `json:"logs,omitempty" yaml:"logs,omitempty"`
}

// ExecutionRecord is the persistent audit record stored after each fix execution.
// This ensures all AutoFix operations are tracked for security and compliance.
type ExecutionRecord struct {
	// ID uniquely identifies this execution record
	ID string `json:"id" yaml:"id"`

	// Timestamp is when this record was created
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// PlanID links to the executed plan
	PlanID string `json:"planId" yaml:"planId"`

	// DiagnosisID links to the original diagnosis
	DiagnosisID string `json:"diagnosisId" yaml:"diagnosisId"`

	// ExecutionResult contains the complete execution outcome
	ExecutionResult *FixResult `json:"executionResult" yaml:"executionResult"`

	// ApprovedBy records who approved the execution (if applicable)
	ApprovedBy string `json:"approvedBy,omitempty" yaml:"approvedBy,omitempty"`

	// ApprovedAt records when approval was granted
	ApprovedAt *time.Time `json:"approvedAt,omitempty" yaml:"approvedAt,omitempty"`

	// SystemState captures relevant system state before execution
	SystemState map[string]interface{} `json:"systemState,omitempty" yaml:"systemState,omitempty"`

	// Tags for categorization and filtering
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ActionResult represents the outcome of a single action execution.
type ActionResult struct {
	// ActionID identifies the action that was executed
	ActionID string `json:"actionId" yaml:"actionId"`

	// Status indicates if this action succeeded, failed, or was skipped
	Status ActionExecutionStatus `json:"status" yaml:"status"`

	// StartedAt is when this action began
	StartedAt time.Time `json:"startedAt" yaml:"startedAt"`

	// CompletedAt is when this action finished
	CompletedAt time.Time `json:"completedAt" yaml:"completedAt"`

	// Output contains the action's stdout/result
	Output string `json:"output,omitempty" yaml:"output,omitempty"`

	// ErrorOutput contains stderr or error details
	ErrorOutput string `json:"errorOutput,omitempty" yaml:"errorOutput,omitempty"`

	// ValidationsPassed indicates if pre-execution validations succeeded
	ValidationsPassed bool `json:"validationsPassed" yaml:"validationsPassed"`

	// Changes records what was actually modified
	Changes []string `json:"changes,omitempty" yaml:"changes,omitempty"`
}

// ValidationReport contains the results of pre-execution validation checks.
type ValidationReport struct {
	// AllPassed indicates if all validations succeeded
	AllPassed bool `json:"allPassed" yaml:"allPassed"`

	// ValidationResults contains individual validation outcomes
	ValidationResults []ValidationResult `json:"validationResults" yaml:"validationResults"`

	// ValidatedAt is when validation was performed
	ValidatedAt time.Time `json:"validatedAt" yaml:"validatedAt"`
}

// ValidationResult represents the outcome of a single validation rule.
type ValidationResult struct {
	// RuleName identifies the validation rule
	RuleName string `json:"ruleName" yaml:"ruleName"`

	// Passed indicates if the validation succeeded
	Passed bool `json:"passed" yaml:"passed"`

	// Message provides details about the validation outcome
	Message string `json:"message,omitempty" yaml:"message,omitempty"`

	// Severity indicates the importance of this validation
	Severity ValidationSeverity `json:"severity" yaml:"severity"`
}

// ValidationRule defines a check that must pass before action execution.
type ValidationRule struct {
	// Name uniquely identifies this validation rule
	Name string `json:"name" yaml:"name"`

	// Type categorizes the validation (e.g., "PrerequisiteCheck", "SafetyCheck")
	Type ValidationType `json:"type" yaml:"type"`

	// Description explains what this rule validates
	Description string `json:"description" yaml:"description"`

	// Severity indicates the importance (Warning allows continuation, Error blocks)
	Severity ValidationSeverity `json:"severity" yaml:"severity"`

	// CheckFunction would contain the actual validation logic (not serialized)
	// This is populated at runtime
}

// ExecutionLogEntry represents a single log line from execution.
type ExecutionLogEntry struct {
	// Timestamp is when this log was generated
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// Level indicates log severity (DEBUG, INFO, WARN, ERROR)
	Level LogLevel `json:"level" yaml:"level"`

	// Message is the log content
	Message string `json:"message" yaml:"message"`

	// ActionID associates this log with a specific action
	ActionID string `json:"actionId,omitempty" yaml:"actionId,omitempty"`

	// Context provides additional structured data
	Context map[string]interface{} `json:"context,omitempty" yaml:"context,omitempty"`
}

// RiskAssessment evaluates the overall risk of a fix plan.
type RiskAssessment struct {
	// Level indicates the overall risk level
	Level RiskLevel `json:"level" yaml:"level"`

	// Score is a numeric risk score (0-100)
	Score int `json:"score" yaml:"score"`

	// Factors lists specific risk contributors
	Factors []RiskFactor `json:"factors" yaml:"factors"`

	// RequiresApproval is true if risk level demands human review
	RequiresApproval bool `json:"requiresApproval" yaml:"requiresApproval"`

	// Mitigations suggests ways to reduce risk
	Mitigations []string `json:"mitigations,omitempty" yaml:"mitigations,omitempty"`
}

// ActionRisk evaluates risk for a single action.
type ActionRisk struct {
	// Level indicates action-specific risk
	Level RiskLevel `json:"level" yaml:"level"`

	// Reason explains why this risk level was assigned
	Reason string `json:"reason" yaml:"reason"`
}

// RiskFactor describes a specific contributor to overall risk.
type RiskFactor struct {
	// Name identifies the risk factor
	Name string `json:"name" yaml:"name"`

	// Description explains the risk
	Description string `json:"description" yaml:"description"`

	// Severity indicates the factor's impact
	Severity RiskLevel `json:"severity" yaml:"severity"`
}

// ExecutionStrategy defines how actions should be executed.
type ExecutionStrategy string

const (
	// ExecutionStrategySerial executes actions one at a time
	ExecutionStrategySerial ExecutionStrategy = "serial"

	// ExecutionStrategyParallel executes independent actions concurrently
	ExecutionStrategyParallel ExecutionStrategy = "parallel"

	// ExecutionStrategyConditional executes based on previous results
	ExecutionStrategyConditional ExecutionStrategy = "conditional"
)

// FixExecutionStatus indicates the overall outcome of fix execution.
type FixExecutionStatus string

const (
	// FixExecutionStatusPending indicates execution hasn't started
	FixExecutionStatusPending FixExecutionStatus = "pending"

	// FixExecutionStatusValidating indicates pre-execution validation is running
	FixExecutionStatusValidating FixExecutionStatus = "validating"

	// FixExecutionStatusValidationFailed indicates validation checks failed
	FixExecutionStatusValidationFailed FixExecutionStatus = "validation_failed"

	// FixExecutionStatusRunning indicates execution is in progress
	FixExecutionStatusRunning FixExecutionStatus = "running"

	// FixExecutionStatusSuccess indicates all actions completed successfully
	FixExecutionStatusSuccess FixExecutionStatus = "success"

	// FixExecutionStatusPartialSuccess indicates some actions succeeded
	FixExecutionStatusPartialSuccess FixExecutionStatus = "partial_success"

	// FixExecutionStatusFailed indicates execution failed
	FixExecutionStatusFailed FixExecutionStatus = "failed"

	// FixExecutionStatusRolledBack indicates failure triggered successful rollback
	FixExecutionStatusRolledBack FixExecutionStatus = "rolled_back"

	// FixExecutionStatusRollbackFailed indicates rollback also failed
	FixExecutionStatusRollbackFailed FixExecutionStatus = "rollback_failed"

	// FixExecutionStatusAborted indicates execution was manually stopped
	FixExecutionStatusAborted FixExecutionStatus = "aborted"
)

// ActionExecutionStatus indicates the outcome of a single action.
type ActionExecutionStatus string

const (
	// ActionExecutionStatusPending indicates action hasn't started
	ActionExecutionStatusPending ActionExecutionStatus = "pending"

	// ActionExecutionStatusRunning indicates action is executing
	ActionExecutionStatusRunning ActionExecutionStatus = "running"

	// ActionExecutionStatusSuccess indicates action completed successfully
	ActionExecutionStatusSuccess ActionExecutionStatus = "success"

	// ActionExecutionStatusFailed indicates action failed
	ActionExecutionStatusFailed ActionExecutionStatus = "failed"

	// ActionExecutionStatusSkipped indicates action was not executed
	ActionExecutionStatusSkipped ActionExecutionStatus = "skipped"

	// ActionExecutionStatusRolledBack indicates action was reverted
	ActionExecutionStatusRolledBack ActionExecutionStatus = "rolled_back"
)

// ActionCategory classifies fix actions for dependency analysis.
type ActionCategory string

const (
	// ActionCategoryValidation represents diagnostic/check actions
	ActionCategoryValidation ActionCategory = "validation"

	// ActionCategoryConfiguration represents config changes
	ActionCategoryConfiguration ActionCategory = "configuration"

	// ActionCategoryRestart represents service restart actions
	ActionCategoryRestart ActionCategory = "restart"

	// ActionCategoryScale represents scaling operations
	ActionCategoryScale ActionCategory = "scale"

	// ActionCategoryCleanup represents cleanup/maintenance actions
	ActionCategoryCleanup ActionCategory = "cleanup"

	// ActionCategoryOther represents uncategorized actions
	ActionCategoryOther ActionCategory = "other"
)

// RiskLevel indicates the severity of risk.
type RiskLevel string

const (
	// RiskLevelLow indicates minimal risk
	RiskLevelLow RiskLevel = "low"

	// RiskLevelMedium indicates moderate risk
	RiskLevelMedium RiskLevel = "medium"

	// RiskLevelHigh indicates significant risk
	RiskLevelHigh RiskLevel = "high"

	// RiskLevelCritical indicates severe risk
	RiskLevelCritical RiskLevel = "critical"
)

// ValidationType categorizes validation rules.
type ValidationType string

const (
	// ValidationTypePrerequisite checks if requirements are met
	ValidationTypePrerequisite ValidationType = "prerequisite"

	// ValidationTypeSafety checks for potentially dangerous conditions
	ValidationTypeSafety ValidationType = "safety"

	// ValidationTypeAuthorization checks if action is permitted
	ValidationTypeAuthorization ValidationType = "authorization"

	// ValidationTypeCapability checks if system can perform action
	ValidationTypeCapability ValidationType = "capability"
)

// ValidationSeverity indicates importance of a validation rule.
type ValidationSeverity string

const (
	// ValidationSeverityInfo indicates informational validation
	ValidationSeverityInfo ValidationSeverity = "info"

	// ValidationSeverityWarning allows continuation even if failed
	ValidationSeverityWarning ValidationSeverity = "warning"

	// ValidationSeverityError blocks execution if failed
	ValidationSeverityError ValidationSeverity = "error"
)

// LogLevel indicates the severity of a log entry.
type LogLevel string

const (
	// LogLevelDebug is for detailed diagnostic information
	LogLevelDebug LogLevel = "DEBUG"

	// LogLevelInfo is for informational messages
	LogLevelInfo LogLevel = "INFO"

	// LogLevelWarn is for warning messages
	LogLevelWarn LogLevel = "WARN"

	// LogLevelError is for error messages
	LogLevelError LogLevel = "ERROR"
)
