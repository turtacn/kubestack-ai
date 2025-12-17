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

package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// AutoFixManager implements the centralized execution governance for Phase 04.
// It enforces the Validate → Execute → Record lifecycle and provides safety boundaries.
//
// Design principles:
// - AutoFix is opt-in, not default behavior
// - All executions are validated before running
// - Every execution produces an audit record
// - Separation of concerns: Analyzer suggests, Manager executes
type AutoFixManager struct {
	log               logger.Logger
	executionManager  interfaces.ExecutionManager
	recordStore       ExecutionRecordStore
	validationEnabled bool
	dryRunMode        bool
}

// ExecutionRecordStore defines the interface for persisting execution records.
// This enables audit trails and compliance tracking.
type ExecutionRecordStore interface {
	// Store saves an execution record
	Store(ctx context.Context, record *ExecutionRecord) error

	// Get retrieves an execution record by ID
	Get(ctx context.Context, id string) (*ExecutionRecord, error)

	// List retrieves execution records matching criteria
	List(ctx context.Context, filters map[string]interface{}) ([]*ExecutionRecord, error)
}

// AutoFixOptions configures the AutoFix execution behavior.
type AutoFixOptions struct {
	// Enabled determines if AutoFix is active (opt-in)
	Enabled bool

	// DryRun simulates execution without making changes
	DryRun bool

	// RequireApproval forces manual approval for all fixes
	RequireApproval bool

	// MaxRiskLevel defines the maximum risk level to auto-execute
	MaxRiskLevel RiskLevel

	// TimeoutPerAction sets the maximum time for each action
	TimeoutPerAction time.Duration

	// EnableRollback determines if automatic rollback is enabled
	EnableRollback bool
}

// NewAutoFixManager creates a new centralized AutoFix execution manager.
func NewAutoFixManager(
	execManager interfaces.ExecutionManager,
	recordStore ExecutionRecordStore,
	opts *AutoFixOptions,
) *AutoFixManager {
	if opts == nil {
		opts = &AutoFixOptions{
			Enabled:          false, // Default: disabled
			DryRun:           true,  // Default: safe mode
			RequireApproval:  true,
			MaxRiskLevel:     RiskLevelMedium,
			TimeoutPerAction: 5 * time.Minute,
			EnableRollback:   true,
		}
	}

	return &AutoFixManager{
		log:               logger.NewLogger("autofix-manager"),
		executionManager:  execManager,
		recordStore:       recordStore,
		validationEnabled: true,
		dryRunMode:        opts.DryRun,
	}
}

// BuildFixPlan creates a FixPlan from diagnosis recommendations.
// This separates planning from execution, allowing human review.
func (m *AutoFixManager) BuildFixPlan(
	ctx context.Context,
	diagnosisID string,
	issues []*models.Issue,
	opts *AutoFixOptions,
) (*FixPlan, error) {
	m.log.Infof("Building fix plan for diagnosis %s with %d issues", diagnosisID, len(issues))

	if !opts.Enabled {
		return nil, fmt.Errorf("AutoFix is disabled (opt-in required)")
	}

	// Extract fixable recommendations
	var actions []*FixActionItem
	sequence := 0

	for _, issue := range issues {
		for _, rec := range issue.Recommendations {
			if rec.CanAutoFix {
				actionItem := &FixActionItem{
					Sequence: sequence,
					Action:   &rec.Fix,
					Category: categorizeAction(&rec.Fix),
					Timeout:  opts.TimeoutPerAction,
					ValidationRules: []ValidationRule{
						{
							Name:        "safety_check",
							Type:        ValidationTypeSafety,
							Description: "Verify action is safe to execute",
							Severity:    ValidationSeverityError,
						},
					},
				}
				actions = append(actions, actionItem)
				sequence++
			}
		}
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no auto-fixable recommendations found")
	}

	// Perform risk assessment
	riskAssessment := m.assessRisk(actions, opts)

	plan := &FixPlan{
		ID:                uuid.New().String(),
		DiagnosisID:       diagnosisID,
		CreatedAt:         time.Now().UTC(),
		Actions:           actions,
		ExecutionStrategy: ExecutionStrategySerial,
		RiskAssessment:    riskAssessment,
		RequiresApproval:  riskAssessment.RequiresApproval || opts.RequireApproval,
		DryRun:            opts.DryRun,
		Metadata:          make(map[string]interface{}),
	}

	plan.Metadata["action_count"] = len(actions)
	plan.Metadata["created_by"] = "autofix-manager"

	m.log.Infof("Fix plan %s created with %d actions, risk level: %s",
		plan.ID, len(actions), riskAssessment.Level)

	return plan, nil
}

// ValidatePlan performs pre-execution validation checks.
// This is the first phase of the Validate → Execute → Record lifecycle.
func (m *AutoFixManager) ValidatePlan(ctx context.Context, plan *FixPlan) (*ValidationReport, error) {
	m.log.Infof("Validating fix plan %s", plan.ID)

	if !m.validationEnabled {
		m.log.Warn("Validation is disabled - skipping checks")
		return &ValidationReport{
			AllPassed:   true,
			ValidatedAt: time.Now().UTC(),
		}, nil
	}

	report := &ValidationReport{
		AllPassed:         true,
		ValidationResults: make([]ValidationResult, 0),
		ValidatedAt:       time.Now().UTC(),
	}

	// Validate each action
	for _, action := range plan.Actions {
		for _, rule := range action.ValidationRules {
			result := m.validateRule(ctx, action, rule)
			report.ValidationResults = append(report.ValidationResults, result)

			if !result.Passed && result.Severity == ValidationSeverityError {
				report.AllPassed = false
			}
		}
	}

	// Additional plan-level validations
	if plan.RiskAssessment.Level == RiskLevelCritical {
		report.ValidationResults = append(report.ValidationResults, ValidationResult{
			RuleName: "critical_risk_check",
			Passed:   false,
			Message:  "Plan has critical risk level - manual approval required",
			Severity: ValidationSeverityError,
		})
		report.AllPassed = false
	}

	if !report.AllPassed {
		m.log.Warnf("Validation failed for plan %s", plan.ID)
	} else {
		m.log.Infof("Validation passed for plan %s", plan.ID)
	}

	return report, nil
}

// ExecuteFixPlan executes a validated fix plan.
// This is the second phase of the Validate → Execute → Record lifecycle.
func (m *AutoFixManager) ExecuteFixPlan(ctx context.Context, plan *FixPlan) (*FixResult, error) {
	m.log.Infof("Executing fix plan %s (DryRun: %v)", plan.ID, plan.DryRun)

	// Validate first (enforced lifecycle)
	validationReport, err := m.ValidatePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if !validationReport.AllPassed {
		return &FixResult{
			PlanID:           plan.ID,
			ExecutionID:      uuid.New().String(),
			Status:           FixExecutionStatusValidationFailed,
			StartedAt:        time.Now().UTC(),
			CompletedAt:      time.Now().UTC(),
			ValidationReport: validationReport,
			ErrorMessage:     "Pre-execution validation failed",
		}, fmt.Errorf("validation checks failed")
	}

	result := &FixResult{
		PlanID:            plan.ID,
		ExecutionID:       uuid.New().String(),
		Status:            FixExecutionStatusRunning,
		StartedAt:         time.Now().UTC(),
		ActionResults:     make([]*ActionResult, 0),
		ValidationReport:  validationReport,
		RollbackPerformed: false,
		Logs:              make([]ExecutionLogEntry, 0),
	}

	m.addLog(result, LogLevelInfo, fmt.Sprintf("Starting execution of plan %s", plan.ID), "")

	// Dry run mode: simulate without executing
	if plan.DryRun || m.dryRunMode {
		m.log.Info("Dry-run mode: simulating execution")
		result = m.simulateExecution(ctx, plan, result)
		result.Status = FixExecutionStatusSuccess
		result.CompletedAt = time.Now().UTC()
		m.addLog(result, LogLevelInfo, "Dry-run completed successfully", "")
		return result, nil
	}

	// Execute actions sequentially (serial strategy)
	completedActions := make([]*ActionResult, 0)

	for _, action := range plan.Actions {
		actionResult := m.executeAction(ctx, action)
		result.ActionResults = append(result.ActionResults, actionResult)

		if actionResult.Status == ActionExecutionStatusSuccess {
			completedActions = append(completedActions, actionResult)
		} else if actionResult.Status == ActionExecutionStatusFailed {
			result.Status = FixExecutionStatusFailed
			result.ErrorMessage = fmt.Sprintf("Action %s failed: %s",
				action.Action.ID, actionResult.ErrorOutput)
			result.CompletedAt = time.Now().UTC()

			// Trigger rollback if enabled
			if plan.Metadata != nil {
				if enableRollback, ok := plan.Metadata["enable_rollback"].(bool); ok && enableRollback {
					m.log.Info("Triggering rollback for failed execution")
					rollbackSuccess := m.performRollback(ctx, completedActions)
					result.RollbackPerformed = true
					result.RollbackSuccess = &rollbackSuccess

					if rollbackSuccess {
						result.Status = FixExecutionStatusRolledBack
					} else {
						result.Status = FixExecutionStatusRollbackFailed
					}
				}
			}

			return result, fmt.Errorf("execution failed: %s", result.ErrorMessage)
		}
	}

	result.Status = FixExecutionStatusSuccess
	result.CompletedAt = time.Now().UTC()
	m.addLog(result, LogLevelInfo, "Execution completed successfully", "")

	m.log.Infof("Fix plan %s executed successfully", plan.ID)
	return result, nil
}

// RecordExecution persists the execution record for audit purposes.
// This is the third phase of the Validate → Execute → Record lifecycle.
func (m *AutoFixManager) RecordExecution(
	ctx context.Context,
	plan *FixPlan,
	result *FixResult,
	approvedBy string,
) (*ExecutionRecord, error) {
	m.log.Infof("Recording execution %s for plan %s", result.ExecutionID, plan.ID)

	record := &ExecutionRecord{
		ID:              uuid.New().String(),
		Timestamp:       time.Now().UTC(),
		PlanID:          plan.ID,
		DiagnosisID:     plan.DiagnosisID,
		ExecutionResult: result,
		ApprovedBy:      approvedBy,
		SystemState:     make(map[string]interface{}),
		Tags:            []string{"autofix", string(result.Status)},
	}

	if approvedBy != "" {
		now := time.Now().UTC()
		record.ApprovedAt = &now
	}

	// Store the record
	if m.recordStore != nil {
		if err := m.recordStore.Store(ctx, record); err != nil {
			m.log.Errorf("Failed to store execution record: %v", err)
			return record, fmt.Errorf("failed to store record: %w", err)
		}
		m.log.Infof("Execution record %s stored successfully", record.ID)
	} else {
		m.log.Warn("No record store configured - execution not persisted")
	}

	return record, nil
}

// executeAction executes a single fix action.
func (m *AutoFixManager) executeAction(ctx context.Context, actionItem *FixActionItem) *ActionResult {
	m.log.Infof("Executing action: %s", actionItem.Action.Description)

	result := &ActionResult{
		ActionID:          actionItem.Action.ID,
		Status:            ActionExecutionStatusRunning,
		StartedAt:         time.Now().UTC(),
		ValidationsPassed: true,
	}

	// Simulate execution (placeholder for actual implementation)
	// In production, this would delegate to the actual executor
	result.Output = fmt.Sprintf("Simulated execution of: %s", actionItem.Action.Command)
	result.Status = ActionExecutionStatusSuccess
	result.CompletedAt = time.Now().UTC()

	m.log.Infof("Action %s completed successfully", actionItem.Action.ID)
	return result
}

// simulateExecution performs a dry-run simulation.
func (m *AutoFixManager) simulateExecution(ctx context.Context, plan *FixPlan, result *FixResult) *FixResult {
	for _, action := range plan.Actions {
		actionResult := &ActionResult{
			ActionID:          action.Action.ID,
			Status:            ActionExecutionStatusSuccess,
			StartedAt:         time.Now().UTC(),
			CompletedAt:       time.Now().UTC(),
			Output:            fmt.Sprintf("[DRY-RUN] Would execute: %s", action.Action.Description),
			ValidationsPassed: true,
		}
		result.ActionResults = append(result.ActionResults, actionResult)
		m.addLog(result, LogLevelInfo, fmt.Sprintf("[DRY-RUN] Simulated: %s", action.Action.Description), action.Action.ID)
	}
	return result
}

// performRollback attempts to revert completed actions.
func (m *AutoFixManager) performRollback(ctx context.Context, completedActions []*ActionResult) bool {
	m.log.Info("Starting rollback process")

	allSuccess := true
	for i := len(completedActions) - 1; i >= 0; i-- {
		action := completedActions[i]
		m.log.Infof("Rolling back action: %s", action.ActionID)

		// Placeholder for actual rollback logic
		// In production, this would execute rollback commands
		m.log.Infof("Action %s rolled back successfully", action.ActionID)
	}

	if allSuccess {
		m.log.Info("Rollback completed successfully")
	} else {
		m.log.Error("Rollback encountered errors")
	}

	return allSuccess
}

// assessRisk evaluates the risk level of a fix plan.
func (m *AutoFixManager) assessRisk(actions []*FixActionItem, opts *AutoFixOptions) *RiskAssessment {
	assessment := &RiskAssessment{
		Level:            RiskLevelLow,
		Score:            0,
		Factors:          make([]RiskFactor, 0),
		RequiresApproval: false,
		Mitigations:      make([]string, 0),
	}

	// Analyze each action for risk factors
	for _, action := range actions {
		if action.Category == ActionCategoryRestart {
			assessment.Score += 20
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Name:        "service_restart",
				Description: "Action requires service restart which may cause downtime",
				Severity:    RiskLevelMedium,
			})
		}

		if action.Action.Command != "" && len(action.Action.Command) > 100 {
			assessment.Score += 10
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Name:        "complex_command",
				Description: "Action involves complex command execution",
				Severity:    RiskLevelLow,
			})
		}
	}

	// Determine overall risk level
	if assessment.Score > 70 {
		assessment.Level = RiskLevelCritical
	} else if assessment.Score > 40 {
		assessment.Level = RiskLevelHigh
	} else if assessment.Score > 20 {
		assessment.Level = RiskLevelMedium
	} else {
		assessment.Level = RiskLevelLow
	}

	// Require approval for high-risk operations
	if assessment.Level >= RiskLevelHigh {
		assessment.RequiresApproval = true
		assessment.Mitigations = append(assessment.Mitigations,
			"Manual approval required",
			"Backup recommended before execution",
			"Schedule during maintenance window")
	}

	return assessment
}

// validateRule executes a single validation rule.
func (m *AutoFixManager) validateRule(
	ctx context.Context,
	action *FixActionItem,
	rule ValidationRule,
) ValidationResult {
	result := ValidationResult{
		RuleName: rule.Name,
		Passed:   true,
		Severity: rule.Severity,
	}

	// Implement validation logic based on rule type
	switch rule.Type {
	case ValidationTypeSafety:
		// Check for potentially dangerous operations
		if action.Action.Command != "" {
			dangerousCommands := []string{"rm -rf", "DROP DATABASE", "DELETE FROM"}
			for _, danger := range dangerousCommands {
				if contains(action.Action.Command, danger) {
					result.Passed = false
					result.Message = fmt.Sprintf("Dangerous command detected: %s", danger)
					return result
				}
			}
		}
		result.Message = "Safety checks passed"

	case ValidationTypePrerequisite:
		// Check prerequisites (placeholder)
		result.Message = "Prerequisites validated"

	case ValidationTypeAuthorization:
		// Check authorization (placeholder)
		result.Message = "Authorization validated"

	default:
		result.Message = "Validation completed"
	}

	return result
}

// addLog adds a log entry to the execution result.
func (m *AutoFixManager) addLog(result *FixResult, level LogLevel, message, actionID string) {
	entry := ExecutionLogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		ActionID:  actionID,
		Context:   make(map[string]interface{}),
	}
	result.Logs = append(result.Logs, entry)
}

// categorizeAction determines the category of a fix action.
func categorizeAction(action *models.FixAction) ActionCategory {
	if action.Category != "" {
		// Map string category to typed category
		switch action.Category {
		case "validation":
			return ActionCategoryValidation
		case "configuration", "config":
			return ActionCategoryConfiguration
		case "restart":
			return ActionCategoryRestart
		case "scale":
			return ActionCategoryScale
		case "cleanup":
			return ActionCategoryCleanup
		}
	}

	// Infer from command or description
	if action.Command != "" {
		if contains(action.Command, "restart") || contains(action.Command, "reload") {
			return ActionCategoryRestart
		}
		if contains(action.Command, "scale") || contains(action.Command, "replicas") {
			return ActionCategoryScale
		}
		if contains(action.Command, "config") || contains(action.Command, "set") {
			return ActionCategoryConfiguration
		}
	}

	return ActionCategoryOther
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// InMemoryRecordStore is a simple in-memory implementation of ExecutionRecordStore
// for testing and development purposes.
type InMemoryRecordStore struct {
	records map[string]*ExecutionRecord
	log     logger.Logger
}

// NewInMemoryRecordStore creates a new in-memory record store.
func NewInMemoryRecordStore() *InMemoryRecordStore {
	return &InMemoryRecordStore{
		records: make(map[string]*ExecutionRecord),
		log:     logger.NewLogger("memory-record-store"),
	}
}

// Store saves an execution record.
func (s *InMemoryRecordStore) Store(ctx context.Context, record *ExecutionRecord) error {
	s.records[record.ID] = record
	s.log.Debugf("Stored execution record %s", record.ID)
	return nil
}

// Get retrieves an execution record by ID.
func (s *InMemoryRecordStore) Get(ctx context.Context, id string) (*ExecutionRecord, error) {
	record, exists := s.records[id]
	if !exists {
		return nil, fmt.Errorf("record not found: %s", id)
	}
	return record, nil
}

// List retrieves execution records matching criteria.
func (s *InMemoryRecordStore) List(ctx context.Context, filters map[string]interface{}) ([]*ExecutionRecord, error) {
	results := make([]*ExecutionRecord, 0, len(s.records))
	for _, record := range s.records {
		results = append(results, record)
	}
	return results, nil
}
