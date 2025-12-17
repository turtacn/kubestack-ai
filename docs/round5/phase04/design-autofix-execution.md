# Phase 04: AutoFix & Execution Design Document

**Phase ID:** P04  
**Branch:** `feat/round5-phase04-autofix-execution`  
**Status:** Implemented  
**Created:** 2024-12-17

## Executive Summary

Phase 04 introduces controlled AutoFix and Execution capabilities to KubeStack-AI with strict safety boundaries and audit trails. This phase addresses the critical design gap between diagnosis (analysis/recommendations) and automated remediation (fix execution).

### Key Principles

1. **Opt-In by Default**: AutoFix is disabled unless explicitly enabled
2. **Validate → Execute → Record**: Enforced lifecycle with pre-execution validation
3. **Separation of Concerns**: Analyzer suggests, ExecutionManager executes
4. **Safety Boundaries**: Whitelist-based operations, risk assessment, approval gates
5. **Complete Audit Trail**: Every execution produces a structured record

## Architecture Overview

### Component Hierarchy

```
┌─────────────────────────────────────────────────────────────┐
│                    Diagnosis Orchestrator                    │
│                    (opt-in AutoFix flag)                     │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ├─► Analysis Results (with CanAutoFix hints)
                 │
                 v
┌─────────────────────────────────────────────────────────────┐
│                      AutoFixManager                          │
│              (Centralized Execution Governance)              │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ├──► 1. BuildFixPlan (from recommendations)
                 │
                 ├──► 2. ValidatePlan (safety checks)
                 │
                 ├──► 3. ExecuteFixPlan (perform actions)
                 │
                 └──► 4. RecordExecution (audit trail)
```

### Data Flow

```
DiagnosisReport → Issues + Recommendations
                      │
                      ├─CanAutoFix=true → FixPlan
                      │                      │
                      │                      ├─► ValidationReport
                      │                      │
                      │                      ├─► FixResult
                      │                      │
                      │                      └─► ExecutionRecord
                      │
                      └─CanAutoFix=false → Manual remediation
```

## Core Components

### 1. AutoFixManager

**Location:** `internal/core/execution/manager.go`

**Responsibilities:**
- Build execution plans from diagnosis recommendations
- Perform pre-execution validation
- Execute fixes with safety controls
- Generate audit records

**Key Methods:**

```go
// Build a fix plan from diagnosis issues
BuildFixPlan(ctx, diagnosisID, issues, opts) (*FixPlan, error)

// Validate plan before execution (enforced)
ValidatePlan(ctx, plan) (*ValidationReport, error)

// Execute validated plan
ExecuteFixPlan(ctx, plan) (*FixResult, error)

// Create audit record
RecordExecution(ctx, plan, result, approvedBy) (*ExecutionRecord, error)
```

### 2. Data Structures

**Location:** `internal/core/execution/types.go`

#### FixPlan
Complete specification of what will be executed:
- Actions to perform (sequenced)
- Risk assessment
- Approval requirements
- Execution strategy (serial/parallel)

#### FixResult
Outcome of execution:
- Overall status (success/failed/rolled back)
- Per-action results
- Validation report
- Execution logs
- Rollback status

#### ExecutionRecord
Permanent audit trail:
- Plan + Result
- Timestamp and approver
- System state snapshot
- Tags for filtering

### 3. Orchestrator Integration

**Location:** `internal/core/diagnosis/orchestrator.go`

**Changes:**
- Added `autoFixEnabled` flag (default: false)
- New constructor: `NewOrchestratorWithOptions`
- Extracts AutoFix hints from analysis results
- Adds hints to report metadata

**Configuration:**

```go
opts := &OrchestratorOptions{
    EnableAutoFix: true,  // Opt-in required
}
orchestrator := NewOrchestratorWithOptions(pm, analyzers, opts)
```

## Safety Mechanisms

### 1. Opt-In Model

AutoFix is **disabled by default**. Must be explicitly enabled:

```go
opts := &AutoFixOptions{
    Enabled: true,  // Required
    DryRun: true,   // Recommended for initial testing
}
```

### 2. Validation Rules

Every action is validated before execution:

| Rule Type | Description | Severity |
|-----------|-------------|----------|
| Safety | Blocks dangerous commands (rm -rf, DROP DATABASE) | Error |
| Prerequisite | Checks requirements are met | Error |
| Authorization | Verifies permissions | Error |
| Capability | Confirms system can perform action | Warning |

### 3. Risk Assessment

Automatic risk scoring based on:
- Action category (restart = higher risk)
- Command complexity
- Number of actions
- System impact

**Risk Levels:**
- **Low**: Minor config changes
- **Medium**: Service restarts, scaling
- **High**: Multiple restarts, complex changes
- **Critical**: Requires manual approval

### 4. Dry-Run Mode

Simulates execution without making changes:

```go
opts := &AutoFixOptions{
    Enabled: true,
    DryRun: true,  // Simulation only
}
```

All actions return simulated results with `[DRY-RUN]` prefix.

### 5. Execution Records

Every execution (successful or failed) produces an audit record:

```go
record := &ExecutionRecord{
    ID: "exec-123",
    Timestamp: time.Now(),
    PlanID: "plan-456",
    DiagnosisID: "diag-789",
    ExecutionResult: result,
    ApprovedBy: "admin@example.com",
}
```

Records are stored in `ExecutionRecordStore` for compliance and troubleshooting.

## Usage Examples

### Example 1: Basic AutoFix Workflow

```go
// 1. Run diagnosis with AutoFix enabled
opts := &diagnosis.OrchestratorOptions{EnableAutoFix: true}
orchestrator := diagnosis.NewOrchestratorWithOptions(pm, analyzers, opts)

report, err := orchestrator.RunDiagnosis(ctx, req, progress)

// 2. Check for auto-fixable issues
if hints, ok := report.Metadata["autofix_hints"]; ok {
    log.Printf("Found fixable issues: %v", hints)
}

// 3. Build fix plan
autoFixMgr := execution.NewAutoFixManager(execMgr, recordStore, &execution.AutoFixOptions{
    Enabled: true,
    DryRun: true,
})

plan, err := autoFixMgr.BuildFixPlan(ctx, report.ID, report.Issues, opts)

// 4. Validate and execute
result, err := autoFixMgr.ExecuteFixPlan(ctx, plan)

// 5. Record execution
record, err := autoFixMgr.RecordExecution(ctx, plan, result, "admin")
```

### Example 2: Dry-Run Testing

```go
// Always test with dry-run first
opts := &execution.AutoFixOptions{
    Enabled: true,
    DryRun: true,  // No actual changes
}

manager := execution.NewAutoFixManager(execMgr, recordStore, opts)
plan, _ := manager.BuildFixPlan(ctx, diagnosisID, issues, opts)

// This will simulate execution
result, err := manager.ExecuteFixPlan(ctx, plan)

// Check simulated results
for _, actionResult := range result.ActionResults {
    log.Printf("[DRY-RUN] %s: %s", actionResult.ActionID, actionResult.Output)
}
```

### Example 3: High-Risk Approval Flow

```go
opts := &execution.AutoFixOptions{
    Enabled: true,
    DryRun: false,
    RequireApproval: true,
    MaxRiskLevel: execution.RiskLevelMedium,
}

plan, _ := manager.BuildFixPlan(ctx, diagnosisID, issues, opts)

// Check if approval is needed
if plan.RequiresApproval {
    // Present plan to operator for review
    approval := requestApproval(plan)
    if !approval {
        return errors.New("execution not approved")
    }
}

// Execute with approval
result, _ := manager.ExecuteFixPlan(ctx, plan)
record, _ := manager.RecordExecution(ctx, plan, result, "admin@example.com")
```

## Testing Strategy

### Unit Tests

**Location:** `internal/core/execution/manager_test.go`

Seven comprehensive test cases:

1. **TestCase-1**: Validation is always performed first
2. **TestCase-2**: All executions produce audit records
3. **TestCase-3**: High-risk operations require approval
4. **TestCase-4**: Dry-run mode doesn't make actual changes
5. **TestCase-5**: AutoFix is disabled by default (opt-in)
6. **TestCase-6**: Validation failure blocks execution
7. **TestCase-7**: Risk assessment logic validation

### Integration Tests

**Location:** `test/integration/diagnosis_orchestrator_flow_test.go`

**TestCase-4**: `TestDiagnosis_DryRunAutoFix`
- End-to-end AutoFix flow
- Verifies AutoFix metadata in reports
- Confirms opt-in behavior

### Running Tests

```bash
# Run unit tests
go test ./internal/core/execution -v

# Run integration tests
go test ./test/integration -v -run TestDiagnosis_DryRunAutoFix

# Run all tests
make test
```

## Configuration

### AutoFixOptions

```go
type AutoFixOptions struct {
    // Enabled: opt-in control (default: false)
    Enabled bool
    
    // DryRun: simulate without changes (default: true)
    DryRun bool
    
    // RequireApproval: force manual approval (default: true)
    RequireApproval bool
    
    // MaxRiskLevel: highest risk to auto-execute (default: Medium)
    MaxRiskLevel RiskLevel
    
    // TimeoutPerAction: max time per action (default: 5m)
    TimeoutPerAction time.Duration
    
    // EnableRollback: automatic rollback on failure (default: true)
    EnableRollback bool
}
```

### OrchestratorOptions

```go
type OrchestratorOptions struct {
    // EnableAutoFix: enable AutoFix capability (default: false)
    EnableAutoFix bool
}
```

## Security Considerations

### 1. Command Whitelist

Only pre-approved command patterns are allowed:
- Configuration updates via structured APIs
- Service management via systemctl/kubectl
- Scaling operations via orchestrator APIs

### 2. Dangerous Command Blacklist

These patterns are **always blocked**:
- `rm -rf`
- `DROP DATABASE`
- `DELETE FROM` (without WHERE)
- Any command with shell injection patterns

### 3. Audit Requirements

All executions must be recorded:
- Who approved
- What was executed
- When it happened
- What the result was

### 4. Rollback Capability

Failed executions trigger automatic rollback:
- Reverse actions in LIFO order
- Execute rollback commands
- Record rollback status

## Limitations & Future Work

### Current Limitations

1. **No Real Execution**: Phase 04 implements the framework but uses simulated execution
2. **In-Memory Storage**: ExecutionRecordStore uses memory (not persistent)
3. **Serial Only**: Only serial execution strategy implemented
4. **Basic Validation**: Advanced validation rules are placeholders

### Phase 05 (Future)

1. **Real Execution**: Integrate with kubectl, systemctl, etc.
2. **Persistent Storage**: Database-backed record store
3. **Parallel Execution**: Execute independent actions concurrently
4. **Advanced Validation**: Plugin-specific validation rules
5. **Rollback Verification**: Verify rollback success
6. **Notification Integration**: Alert on execution results

## Acceptance Criteria

- [x] **AC-1**: AutoFix is disabled by default (opt-in)
- [x] **AC-2**: All executions produce ExecutionRecords
- [x] **AC-3**: Analyzer cannot directly call ExecuteFix
- [x] **AC-4**: Integration tests pass
- [x] **AC-5**: Validation always runs before execution
- [x] **AC-6**: High-risk operations require approval
- [x] **AC-7**: Dry-run mode available

## Conclusion

Phase 04 establishes the **governance framework** for AutoFix without implementing dangerous real-world execution. This allows:

1. **Design Validation**: Verify the lifecycle and safety mechanisms work
2. **Interface Stability**: Lock down APIs before implementing execution
3. **Testing Infrastructure**: Build comprehensive tests with mocks
4. **Incremental Rollout**: Add real execution capabilities gradually in future phases

The system is now ready for careful, controlled addition of actual execution capabilities.

## References

- [Architecture Documentation](../../architecture.md)
- [Execution Interface](../../../internal/core/interfaces/execution.go)
- [AutoFix Manager Implementation](../../../internal/core/execution/manager.go)
- [Phase 04 API Guide](./api-guide.md)
