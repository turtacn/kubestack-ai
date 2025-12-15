// Package execution implements the core logic for the execution engine.
package execution

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/confirm"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/plan"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/risk"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/rollback"
)

// --- Action Executor (Worker) Implementation ---

// actionExecutor is the concrete implementation of interfaces.ActionExecutor.
// It is the "hands" of the engine, performing the low-level, actual work.
type actionExecutor struct {
	log logger.Logger
}

func newActionExecutor() interfaces.ActionExecutor {
	return &actionExecutor{
		log: logger.NewLogger("action-executor"),
	}
}

func (e *actionExecutor) ExecuteCommand(ctx context.Context, command string) (string, string, error) {
	e.log.Infof("Executing command: %s", command)
	// SECURITY NOTE: In a real-world scenario, never execute arbitrary commands.
	// Commands must be sanitized, validated, and originate from a trusted source (e.g., predefined in plugins).
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		e.log.Errorf("Command execution failed. Stderr: %s", stderr.String())
	}
	return stdout.String(), stderr.String(), err
}

func (e *actionExecutor) ApplyConfiguration(ctx context.Context, configChange *models.ConfigChange) error {
	e.log.Infof("Applying config change to file %s: set %s = %s", configChange.File, configChange.Key, configChange.Value)
	// This is a placeholder. A real implementation would need to safely parse and write to
	// different config formats (YAML, INI, etc.) and handle file permissions and backups.
	return fmt.Errorf("ApplyConfiguration is not implemented")
}

func (e *actionExecutor) RollbackChanges(ctx context.Context, steps []*models.ExecutionStep) error {
	e.log.Info("Starting rollback for failed execution.")
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Action.RollbackCommand == "" {
			e.log.Warnf("No rollback command found for step '%s', skipping.", step.Name)
			continue
		}
		e.log.Infof("Rolling back step: %s (Command: `%s`)", step.Name, step.Action.RollbackCommand)
		_, _, err := e.ExecuteCommand(ctx, step.Action.RollbackCommand)
		if err != nil {
			e.log.Errorf("Rollback for step '%s' failed: %v", step.Name, err)
			return fmt.Errorf("rollback for step '%s' failed: %w", step.Name, err)
		}
		e.log.Infof("Rollback for step '%s' completed successfully.", step.Name)
	}
	e.log.Info("Rollback completed successfully.")
	return nil
}

// --- Execution Manager (Coordinator) Implementation ---

// manager is the concrete implementation of interfaces.ExecutionManager.
type manager struct {
	log             logger.Logger
	planner         interfaces.ExecutionPlanner
	executor        interfaces.ActionExecutor
	riskAssessor    *risk.RiskAssessor
	rollbackManager *rollback.RollbackManager
	confirmHandler  *confirm.ConfirmationHandler
	planStore       plan.PlanStore
}

type ManagerOptions struct {
	RiskThresholds    *risk.RiskThresholds
	ConfirmationHdlr  *confirm.ConfirmationHandler
	RollbackMgr       *rollback.RollbackManager
	PlanStore         plan.PlanStore
}

// NewManager creates a new instance of the execution manager.
func NewManager(planner interfaces.ExecutionPlanner, opts *ManagerOptions) interfaces.ExecutionManager {
	if opts == nil {
		opts = &ManagerOptions{}
	}
	if opts.ConfirmationHdlr == nil {
		// Default to NO confirmation channels if not provided
		opts.ConfirmationHdlr = confirm.NewConfirmationHandler(5*time.Minute, nil)
	}
	if opts.RollbackMgr == nil {
		opts.RollbackMgr = rollback.NewRollbackManager(rollback.NewInMemorySnapshotStore())
	}
	if opts.PlanStore == nil {
		opts.PlanStore = plan.NewFilePlanStore("/tmp/kubestack/plans")
	}

	return &manager{
		log:             logger.NewLogger("execution-manager"),
		planner:         planner,
		executor:        newActionExecutor(),
		riskAssessor:    risk.NewRiskAssessor(opts.RiskThresholds),
		rollbackManager: opts.RollbackMgr,
		confirmHandler:  opts.ConfirmationHdlr,
		planStore:       opts.PlanStore,
	}
}

// GeneratePlan delegates the task of generating an execution plan to the configured planner.
func (m *manager) GeneratePlan(ctx context.Context, issues []models.Issue) (*models.ExecutionPlan, error) {
	m.log.Info("Delegating execution planning to the planner component.")
	return m.planner.GeneratePlan(ctx, issues)
}

// ExecutePlan carries out the steps defined in an execution plan.
func (m *manager) ExecutePlan(ctx context.Context, execPlan *models.ExecutionPlan) (*models.ExecutionResult, error) {
	m.log.Infof("Starting execution of plan ID: %s with strategy: %s", execPlan.ID, execPlan.Strategy)

	sm := plan.NewStateMachine(execPlan)

	// === 阶段1: 风险评估 ===
	assessment, err := m.riskAssessor.Assess(ctx, execPlan)
	if err != nil {
		return nil, fmt.Errorf("risk assessment failed: %w", err)
	}
	// Note: ExecutionPlan struct needs to be updated to hold RiskAssessmentResult more specifically if needed.
	// Current model has 'Risk *RiskAssessment'. We can map it.
	execPlan.Risk = &models.RiskAssessment{
		TotalScore: assessment.Score,
		MaxSeverity: models.RiskLevel(assessment.Level - 1), // Assuming 0-based index match
		RequiresApproval: assessment.RequiresConfirm || assessment.RequiresApproval,
		Description: fmt.Sprintf("Risk Level: %s. Reasons: %s", assessment.Level, strings.Join(assessment.Reasons, "; ")),
	}

	// === 阶段2: 用户确认(若需要) ===
	if assessment.RequiresConfirm {
		sm.Transition("pending") // 进入待确认状态
		m.planStore.Save(ctx, execPlan)

		// Wait for confirmation
		// Note: This blocks. In async mode, this should just set state to pending and return.
		// For now we assume sync blocking or the caller handles it.
		// If we return here, we need a way to resume.

		resp, err := m.confirmHandler.RequestConfirmation(ctx, execPlan, assessment)
		if err != nil {
			sm.Transition("cancel")
			m.planStore.Update(ctx, execPlan)
			return nil, fmt.Errorf("confirmation failed: %w", err)
		}

		if !resp.Approved {
			sm.Transition("cancel")
			m.planStore.Update(ctx, execPlan)
			return nil, errors.New("execution plan rejected by user")
		}
		sm.Transition("approve")
		m.planStore.Update(ctx, execPlan)
	} else {
		// Auto approve if low risk
		if sm.CurrentState() == plan.StatePending {
			sm.Transition("approve")
		}
	}

	// === 阶段3: 创建回滚检查点 ===
	checkpointID, err := m.rollbackManager.CreateCheckpoint(ctx, execPlan)
	if err != nil {
		m.log.Warn("checkpoint creation failed", "error", err)
		// Non-fatal, continue but log warning
	}

	// === 阶段4: 执行操作 ===
	sm.Transition("start")
	m.planStore.Update(ctx, execPlan)

	result := &models.ExecutionResult{
		PlanID:    execPlan.ID,
		Status:    models.ExecutionStatusInProgress,
		StartTime: time.Now().UTC(),
		Logs:      make([]*models.ExecutionLog, 0),
	}

	var completedSteps []*models.ExecutionStep
	for _, step := range execPlan.Steps {
		step.Status = models.StepStatusRunning
		m.log.Infof("Executing step: %s", step.Name)
		_, stderr, err := m.executor.ExecuteCommand(ctx, step.Action.Command)
		if err != nil {
			step.Status = models.StepStatusFailed
			step.Result = fmt.Sprintf("Error: %v\nStderr: %s", err, stderr)
			m.log.Errorf("Step '%s' failed: %v", step.Name, err)

			sm.Transition("fail")

			// === 阶段5: 自动回滚 ===
			// Use Snapshot Rollback
			if checkpointID != "" {
				m.log.Info("Attempting snapshot rollback...")
				rollbackResult, err := m.rollbackManager.Rollback(ctx, checkpointID, execPlan)
				if err != nil {
					m.log.Errorf("Snapshot rollback failed: %v", err)
				} else {
					if rollbackResult.Success {
						m.log.Info("Snapshot rollback successful.")
						sm.Transition("rollback")
					} else {
						m.log.Warn("Snapshot rollback partial failure.")
					}
				}
			}

			// Fallback to command-based rollback if snapshot rollback is not enough or failed,
			// or as complimentary. Original logic:
			if err := m.executor.RollbackChanges(ctx, completedSteps); err != nil {
				result.Status = models.ExecutionStatusFailedWithRollbackFailure
				m.log.Errorf("Command rollback failed: %v", err)
			} else {
				if sm.CurrentState() != plan.StateRolledBack {
					result.Status = models.ExecutionStatusFailedWithRollbackSuccess
				}
				m.log.Info("Command rollback completed successfully.")
			}

			result.EndTime = time.Now().UTC()
			m.planStore.Update(ctx, execPlan)
			return result, fmt.Errorf("execution of step '%s' failed", step.Name)
		}
		step.Status = models.StepStatusSuccess
		completedSteps = append(completedSteps, step)
		m.log.Infof("Step '%s' completed successfully.", step.Name)
	}

	result.Status = models.ExecutionStatusSuccess
	result.EndTime = time.Now().UTC()
	sm.Transition("complete")
	m.planStore.Update(ctx, execPlan)
	m.log.Infof("Execution of plan %s completed successfully.", execPlan.ID)
	return result, nil
}

// ValidateExecution is responsible for verifying that an executed plan has
// successfully resolved the underlying issue.
func (m *manager) ValidateExecution(ctx context.Context, result *models.ExecutionResult) error {
	m.log.Infof("Validating execution result for plan ID: %s", result.PlanID)
	if result.Status != models.ExecutionStatusSuccess {
		return fmt.Errorf("cannot validate a failed or incomplete execution")
	}
	m.log.Info("Validation successful (placeholder implementation).")
	return nil
}
