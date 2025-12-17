# Phase 04: AutoFix API Reference

## Overview

This document provides detailed API reference for the AutoFix and Execution components introduced in Phase 04.

## AutoFixManager API

### Constructor

#### NewAutoFixManager

```go
func NewAutoFixManager(
    execManager interfaces.ExecutionManager,
    recordStore ExecutionRecordStore,
    opts *AutoFixOptions,
) *AutoFixManager
```

Creates a new AutoFix manager with centralized execution governance.

**Parameters:**
- `execManager`: Legacy execution manager interface (for backward compatibility)
- `recordStore`: Storage for execution records (use `NewInMemoryRecordStore()` for testing)
- `opts`: Configuration options (if nil, uses safe defaults)

**Returns:** Configured AutoFixManager instance

**Example:**
```go
recordStore := execution.NewInMemoryRecordStore()
execMgr := execution.NewManager(execution.NewPlanner())

opts := &execution.AutoFixOptions{
    Enabled: true,
    DryRun: true,
    MaxRiskLevel: execution.RiskLevelMedium,
}

manager := execution.NewAutoFixManager(execMgr, recordStore, opts)
```

---

### Methods

#### BuildFixPlan

```go
func (m *AutoFixManager) BuildFixPlan(
    ctx context.Context,
    diagnosisID string,
    issues []*models.Issue,
    opts *AutoFixOptions,
) (*FixPlan, error)
```

Creates an execution plan from diagnosis recommendations.

**Parameters:**
- `ctx`: Request context
- `diagnosisID`: ID of the source diagnosis
- `issues`: Issues containing auto-fixable recommendations
- `opts`: Execution options (must have `Enabled: true`)

**Returns:** 
- `*FixPlan`: Complete execution plan with risk assessment
- `error`: Non-nil if AutoFix is disabled or no fixable recommendations found

**Example:**
```go
plan, err := manager.BuildFixPlan(ctx, "diag-123", report.Issues, opts)
if err != nil {
    return fmt.Errorf("failed to build plan: %w", err)
}

log.Printf("Created plan with %d actions, risk: %s", 
    len(plan.Actions), plan.RiskAssessment.Level)
```

---

#### ValidatePlan

```go
func (m *AutoFixManager) ValidatePlan(
    ctx context.Context,
    plan *FixPlan,
) (*ValidationReport, error)
```

Performs pre-execution validation checks on a fix plan.

**Parameters:**
- `ctx`: Request context
- `plan`: Fix plan to validate

**Returns:**
- `*ValidationReport`: Validation results with pass/fail per rule
- `error`: Only on system errors (not validation failures)

**Validation Rules:**
- Safety checks (dangerous commands)
- Prerequisite verification
- Authorization checks
- Risk level assessment

**Example:**
```go
report, err := manager.ValidatePlan(ctx, plan)
if err != nil {
    return fmt.Errorf("validation error: %w", err)
}

if !report.AllPassed {
    for _, result := range report.ValidationResults {
        if !result.Passed && result.Severity == execution.ValidationSeverityError {
            log.Printf("BLOCKED: %s - %s", result.RuleName, result.Message)
        }
    }
    return errors.New("validation failed")
}
```

---

#### ExecuteFixPlan

```go
func (m *AutoFixManager) ExecuteFixPlan(
    ctx context.Context,
    plan *FixPlan,
) (*FixResult, error)
```

Executes a validated fix plan. **Automatically validates first** (enforced lifecycle).

**Parameters:**
- `ctx`: Request context
- `plan`: Fix plan to execute

**Returns:**
- `*FixResult`: Complete execution results with per-action outcomes
- `error`: Non-nil if validation or execution fails

**Execution Flow:**
1. Validate plan (automatic)
2. If dry-run: simulate execution
3. If real: execute actions sequentially
4. On failure: trigger rollback (if enabled)

**Example:**
```go
result, err := manager.ExecuteFixPlan(ctx, plan)
if err != nil {
    log.Printf("Execution failed: %v", err)
    if result != nil && result.RollbackPerformed {
        log.Printf("Rollback status: success=%v", *result.RollbackSuccess)
    }
    return err
}

log.Printf("Execution completed: status=%s", result.Status)
for _, actionResult := range result.ActionResults {
    log.Printf("  Action %s: %s", actionResult.ActionID, actionResult.Status)
}
```

---

#### RecordExecution

```go
func (m *AutoFixManager) RecordExecution(
    ctx context.Context,
    plan *FixPlan,
    result *FixResult,
    approvedBy string,
) (*ExecutionRecord, error)
```

Creates and stores a permanent audit record.

**Parameters:**
- `ctx`: Request context
- `plan`: The executed plan
- `result`: Execution results
- `approvedBy`: Identifier of approver (email, user ID, etc.)

**Returns:**
- `*ExecutionRecord`: Stored audit record
- `error`: Non-nil if storage fails

**Example:**
```go
record, err := manager.RecordExecution(ctx, plan, result, "admin@example.com")
if err != nil {
    log.Printf("Warning: failed to record execution: %v", err)
}

log.Printf("Execution recorded: ID=%s, Time=%s", record.ID, record.Timestamp)
```

---

## Orchestrator API Extensions

### NewOrchestratorWithOptions

```go
func NewOrchestratorWithOptions(
    pm interfaces.PluginManager,
    analyzers []analysis.Analyzer,
    opts *OrchestratorOptions,
) *Orchestrator
```

Creates an orchestrator with AutoFix capability.

**Parameters:**
- `pm`: Plugin manager
- `analyzers`: List of diagnosis analyzers
- `opts`: Orchestrator options (set `EnableAutoFix: true`)

**Example:**
```go
opts := &diagnosis.OrchestratorOptions{
    EnableAutoFix: true,
}

orchestrator := diagnosis.NewOrchestratorWithOptions(pm, analyzers, opts)
```

---

### IsAutoFixEnabled

```go
func (o *Orchestrator) IsAutoFixEnabled() bool
```

Returns whether AutoFix is enabled for this orchestrator.

---

### SetAutoFixEnabled

```go
func (o *Orchestrator) SetAutoFixEnabled(enabled bool)
```

Enables or disables AutoFix capability dynamically.

**Note:** Should only be called during initialization.

---

## Data Structures

### FixPlan

Complete specification of fix execution.

```go
type FixPlan struct {
    ID                string
    DiagnosisID       string
    CreatedAt         time.Time
    Actions           []*FixActionItem
    ExecutionStrategy ExecutionStrategy  // serial, parallel, conditional
    RiskAssessment    *RiskAssessment
    RequiresApproval  bool
    DryRun            bool
    Metadata          map[string]interface{}
}
```

### FixResult

Outcome of fix execution.

```go
type FixResult struct {
    PlanID            string
    ExecutionID       string
    Status            FixExecutionStatus
    StartedAt         time.Time
    CompletedAt       time.Time
    ActionResults     []*ActionResult
    ValidationReport  *ValidationReport
    RollbackPerformed bool
    RollbackSuccess   *bool
    ErrorMessage      string
    Logs              []ExecutionLogEntry
}
```

### ExecutionRecord

Permanent audit trail.

```go
type ExecutionRecord struct {
    ID              string
    Timestamp       time.Time
    PlanID          string
    DiagnosisID     string
    ExecutionResult *FixResult
    ApprovedBy      string
    ApprovedAt      *time.Time
    SystemState     map[string]interface{}
    Tags            []string
}
```

### AutoFixOptions

Configuration for AutoFix behavior.

```go
type AutoFixOptions struct {
    Enabled          bool          // Opt-in control
    DryRun           bool          // Simulate without changes
    RequireApproval  bool          // Force manual approval
    MaxRiskLevel     RiskLevel     // Maximum auto-executable risk
    TimeoutPerAction time.Duration // Per-action timeout
    EnableRollback   bool          // Auto-rollback on failure
}
```

**Defaults:**
```go
{
    Enabled: false,
    DryRun: true,
    RequireApproval: true,
    MaxRiskLevel: RiskLevelMedium,
    TimeoutPerAction: 5 * time.Minute,
    EnableRollback: true,
}
```

---

## Enumerations

### FixExecutionStatus

```go
const (
    FixExecutionStatusPending           // Not started
    FixExecutionStatusValidating        // Validation in progress
    FixExecutionStatusValidationFailed  // Validation checks failed
    FixExecutionStatusRunning           // Execution in progress
    FixExecutionStatusSuccess           // All actions succeeded
    FixExecutionStatusPartialSuccess    // Some actions succeeded
    FixExecutionStatusFailed            // Execution failed
    FixExecutionStatusRolledBack        // Failed with successful rollback
    FixExecutionStatusRollbackFailed    // Rollback also failed
    FixExecutionStatusAborted           // Manually stopped
)
```

### RiskLevel

```go
const (
    RiskLevelLow      // Minor config changes
    RiskLevelMedium   // Service restarts, scaling
    RiskLevelHigh     // Multiple restarts, complex changes
    RiskLevelCritical // Requires manual approval
)
```

### ActionCategory

```go
const (
    ActionCategoryValidation     // Diagnostic/check actions
    ActionCategoryConfiguration  // Config changes
    ActionCategoryRestart        // Service restart actions
    ActionCategoryScale          // Scaling operations
    ActionCategoryCleanup        // Cleanup/maintenance
    ActionCategoryOther          // Uncategorized
)
```

---

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| "AutoFix is disabled" | `Enabled: false` in options | Set `opts.Enabled = true` |
| "no auto-fixable actions found" | No recommendations with `CanAutoFix=true` | Check diagnosis recommendations |
| "validation checks failed" | Safety/prerequisite checks failed | Review validation report, fix issues |
| "cannot validate a failed or incomplete execution" | Trying to validate non-successful result | Only validate successful executions |

### Error Example

```go
result, err := manager.ExecuteFixPlan(ctx, plan)
if err != nil {
    // Check if validation failed
    if result != nil && result.Status == execution.FixExecutionStatusValidationFailed {
        log.Println("Validation failures:")
        for _, vr := range result.ValidationReport.ValidationResults {
            if !vr.Passed {
                log.Printf("  - %s: %s", vr.RuleName, vr.Message)
            }
        }
    }
    return err
}
```

---

## Complete Example: End-to-End Flow

```go
package main

import (
    "context"
    "log"
    
    "github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/execution"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

func main() {
    ctx := context.Background()
    
    // 1. Setup components
    recordStore := execution.NewInMemoryRecordStore()
    execMgr := execution.NewManager(execution.NewPlanner())
    
    // 2. Create AutoFix manager (disabled by default)
    autoFixOpts := &execution.AutoFixOptions{
        Enabled: true,
        DryRun: true, // Safe mode
        MaxRiskLevel: execution.RiskLevelMedium,
    }
    autoFixMgr := execution.NewAutoFixManager(execMgr, recordStore, autoFixOpts)
    
    // 3. Run diagnosis with AutoFix hints
    diagOpts := &diagnosis.OrchestratorOptions{
        EnableAutoFix: true,
    }
    orchestrator := diagnosis.NewOrchestratorWithOptions(pm, analyzers, diagOpts)
    
    req := &models.DiagnosisRequest{
        TargetMiddleware: "redis",
        Instance: "redis-master",
    }
    
    progress := make(chan interfaces.DiagnosisProgress, 10)
    report, err := orchestrator.RunDiagnosis(ctx, req, progress)
    if err != nil {
        log.Fatal(err)
    }
    
    // 4. Check for auto-fixable issues
    if hints, ok := report.Metadata["autofix_hints"]; ok {
        log.Printf("AutoFix hints found: %v", hints)
    } else {
        log.Println("No auto-fixable issues found")
        return
    }
    
    // 5. Build fix plan
    plan, err := autoFixMgr.BuildFixPlan(ctx, report.ID, report.Issues, autoFixOpts)
    if err != nil {
        log.Fatalf("Failed to build plan: %v", err)
    }
    
    log.Printf("Plan created: %d actions, risk: %s", 
        len(plan.Actions), plan.RiskAssessment.Level)
    
    // 6. Review and approve if needed
    if plan.RequiresApproval {
        log.Println("Plan requires approval:")
        for _, action := range plan.Actions {
            log.Printf("  - %s", action.Action.Description)
        }
        
        // In production: present to operator for approval
        // For now: auto-approve low/medium risk
        if plan.RiskAssessment.Level == execution.RiskLevelCritical {
            log.Fatal("Critical risk - manual approval required")
        }
    }
    
    // 7. Execute (dry-run mode)
    result, err := autoFixMgr.ExecuteFixPlan(ctx, plan)
    if err != nil {
        log.Printf("Execution failed: %v", err)
        if result != nil && result.RollbackPerformed {
            log.Printf("Rollback performed: success=%v", *result.RollbackSuccess)
        }
        return
    }
    
    log.Printf("Execution completed: status=%s", result.Status)
    
    // 8. Review results
    for _, actionResult := range result.ActionResults {
        log.Printf("Action %s: %s", actionResult.ActionID, actionResult.Status)
        log.Printf("  Output: %s", actionResult.Output)
    }
    
    // 9. Record for audit
    record, err := autoFixMgr.RecordExecution(ctx, plan, result, "admin@example.com")
    if err != nil {
        log.Printf("Warning: failed to record: %v", err)
    } else {
        log.Printf("Execution recorded: %s", record.ID)
    }
}
```

---

## Best Practices

### 1. Always Start with Dry-Run

```go
// Test first
opts := &execution.AutoFixOptions{
    Enabled: true,
    DryRun: true,
}
```

### 2. Review Plans Before Execution

```go
plan, _ := manager.BuildFixPlan(ctx, diagnosisID, issues, opts)

// Log for review
for i, action := range plan.Actions {
    log.Printf("[%d] %s", i, action.Action.Description)
    log.Printf("    Category: %s", action.Category)
    log.Printf("    Risk: %s", action.Risk.Level)
}
```

### 3. Handle Validation Failures Gracefully

```go
result, err := manager.ExecuteFixPlan(ctx, plan)
if result != nil && !result.ValidationReport.AllPassed {
    // Validation failed - safe to retry after fixing issues
    for _, vr := range result.ValidationReport.ValidationResults {
        if !vr.Passed {
            notifyOperator(vr.RuleName, vr.Message)
        }
    }
}
```

### 4. Always Record Executions

```go
// Record even on failure
record, err := manager.RecordExecution(ctx, plan, result, approver)
if err != nil {
    // Log warning but don't fail the operation
    log.Printf("Failed to create audit record: %v", err)
}
```

---

## See Also

- [Design Document](./design-autofix-execution.md)
- [Implementation Summary](./implementation-summary.md)
- [Testing Guide](./testing-guide.md)
