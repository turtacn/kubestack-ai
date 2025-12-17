# Phase 04: AutoFix & Execution Capability Alignment

**Status:** ✅ Completed  
**Branch:** `feat/round5-phase04-autofix-execution`  
**Date:** 2024-12-17

## Overview

Phase 04 introduces controlled, opt-in AutoFix capabilities to KubeStack-AI with strict safety boundaries and complete audit trails. This phase establishes the **governance framework** for automated remediation without implementing dangerous real-world execution.

## Key Achievements

### ✅ Centralized Execution Governance

- **AutoFixManager**: Single source of truth for all fix executions
- **Enforced Lifecycle**: Validate → Execute → Record (cannot be bypassed)
- **Separation of Concerns**: Analyzer suggests, Manager executes

### ✅ Safety Boundaries

- **Opt-In by Default**: AutoFix disabled unless explicitly enabled
- **Validation Rules**: Pre-execution safety checks (blocks dangerous operations)
- **Risk Assessment**: Automatic risk scoring with approval gates
- **Dry-Run Mode**: Simulate execution without making changes

### ✅ Complete Audit Trail

- **ExecutionRecord**: Every execution produces a permanent record
- **Structured Logging**: Detailed per-action logs
- **Rollback Tracking**: Records rollback attempts and outcomes

### ✅ Comprehensive Testing

- **7 Unit Tests**: Cover all critical paths and edge cases
- **Integration Test**: End-to-end AutoFix flow with orchestrator
- **Test Coverage**: Validation, execution, recording, dry-run, approval

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│              Diagnosis Orchestrator                       │
│                  (opt-in flag)                           │
└─────────────────────┬────────────────────────────────────┘
                      │
                      ├─► Issues with CanAutoFix hints
                      │
                      v
┌──────────────────────────────────────────────────────────┐
│                  AutoFixManager                           │
│         (Validate → Execute → Record)                    │
└─────────────────────┬────────────────────────────────────┘
                      │
                      ├──► ExecutionRecordStore
                      │
                      └──► Audit Trail
```

## Components

### Core Implementation

| Component | Location | Purpose |
|-----------|----------|---------|
| **AutoFixManager** | `internal/core/execution/manager.go` | Centralized execution governance |
| **Types** | `internal/core/execution/types.go` | Data structures (FixPlan, FixResult, ExecutionRecord) |
| **Orchestrator** | `internal/core/diagnosis/orchestrator.go` | AutoFix integration with diagnosis |

### Testing

| Test Suite | Location | Coverage |
|------------|----------|----------|
| **Unit Tests** | `internal/core/execution/manager_test.go` | 7 test cases |
| **Integration Test** | `test/integration/diagnosis_orchestrator_flow_test.go` | End-to-end AutoFix |

### Documentation

| Document | Purpose |
|----------|---------|
| [design-autofix-execution.md](./design-autofix-execution.md) | Complete design and architecture |
| [api-guide.md](./api-guide.md) | API reference and examples |
| [README.md](./README.md) | This file - overview and quickstart |

## Quick Start

### 1. Enable AutoFix in Diagnosis

```go
import (
    "github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/execution"
)

// Create orchestrator with AutoFix enabled
opts := &diagnosis.OrchestratorOptions{
    EnableAutoFix: true, // Opt-in required
}
orchestrator := diagnosis.NewOrchestratorWithOptions(pm, analyzers, opts)

// Run diagnosis
report, err := orchestrator.RunDiagnosis(ctx, req, progress)

// Check for auto-fixable issues
if hints, ok := report.Metadata["autofix_hints"]; ok {
    log.Printf("Found fixable issues: %v", hints)
}
```

### 2. Build and Execute Fix Plan

```go
// Create AutoFix manager
recordStore := execution.NewInMemoryRecordStore()
execMgr := execution.NewManager(execution.NewPlanner())

autoFixOpts := &execution.AutoFixOptions{
    Enabled: true,
    DryRun: true, // Always start with dry-run
    MaxRiskLevel: execution.RiskLevelMedium,
}

manager := execution.NewAutoFixManager(execMgr, recordStore, autoFixOpts)

// Build plan from diagnosis results
plan, err := manager.BuildFixPlan(ctx, report.ID, report.Issues, autoFixOpts)
if err != nil {
    log.Fatalf("Failed to build plan: %v", err)
}

// Execute (will validate automatically)
result, err := manager.ExecuteFixPlan(ctx, plan)
if err != nil {
    log.Printf("Execution failed: %v", err)
}

// Record for audit
record, _ := manager.RecordExecution(ctx, plan, result, "admin@example.com")
```

### 3. Run Tests

```bash
# Unit tests
go test ./internal/core/execution -v

# Integration tests
go test ./test/integration -v -run TestDiagnosis_DryRunAutoFix

# All tests
make test
```

## Safety Features

### 1. Opt-In Model

AutoFix is **disabled by default**. Must be explicitly enabled at two levels:

```go
// Level 1: Orchestrator
diagOpts := &diagnosis.OrchestratorOptions{
    EnableAutoFix: true, // Required
}

// Level 2: AutoFix execution
autoFixOpts := &execution.AutoFixOptions{
    Enabled: true, // Required
}
```

### 2. Validation Enforcement

Every execution **must pass validation first**:

```go
// This is automatic - you cannot bypass it
result, err := manager.ExecuteFixPlan(ctx, plan)
// ^ Validates before executing

// Check validation results
if result.ValidationReport != nil && !result.ValidationReport.AllPassed {
    // Review failures
    for _, vr := range result.ValidationReport.ValidationResults {
        if !vr.Passed {
            log.Printf("Failed: %s - %s", vr.RuleName, vr.Message)
        }
    }
}
```

### 3. Risk-Based Approval

High-risk operations require approval:

```go
plan, _ := manager.BuildFixPlan(ctx, diagnosisID, issues, opts)

if plan.RequiresApproval {
    // Present to operator
    approval := requestApproval(plan)
    if !approval {
        return errors.New("execution not approved")
    }
}
```

### 4. Dry-Run Testing

Always test with dry-run first:

```go
opts := &execution.AutoFixOptions{
    Enabled: true,
    DryRun: true, // Simulates without changes
}

result, _ := manager.ExecuteFixPlan(ctx, plan)

// All actions show [DRY-RUN] prefix
for _, ar := range result.ActionResults {
    log.Printf("%s: %s", ar.ActionID, ar.Output)
    // Output: "[DRY-RUN] Would execute: ..."
}
```

## Test Coverage

### Unit Tests (7 Test Cases)

1. ✅ **Validation First**: Verifies validation always runs before execution
2. ✅ **Record Always**: All executions produce audit records
3. ✅ **Reject Unauthorized**: High-risk operations require approval
4. ✅ **Dry-Run Mode**: Simulates without actual changes
5. ✅ **AutoFix Disabled**: Default opt-in behavior verified
6. ✅ **Validation Blocks**: Failed validation prevents execution
7. ✅ **Risk Assessment**: Correct risk level assignment

### Integration Test

✅ **Dry-Run AutoFix**: End-to-end flow from diagnosis to execution

## Configuration Reference

### AutoFixOptions

```go
type AutoFixOptions struct {
    Enabled          bool          // Default: false (opt-in)
    DryRun           bool          // Default: true (safe mode)
    RequireApproval  bool          // Default: true
    MaxRiskLevel     RiskLevel     // Default: Medium
    TimeoutPerAction time.Duration // Default: 5 minutes
    EnableRollback   bool          // Default: true
}
```

### Risk Levels

| Level | Description | Auto-Execute? |
|-------|-------------|---------------|
| Low | Minor config changes | ✅ Yes |
| Medium | Service restarts, scaling | ✅ Yes (if MaxRiskLevel allows) |
| High | Multiple restarts, complex changes | ⚠️ Requires approval |
| Critical | Dangerous operations | ❌ Blocked |

## Current Limitations

Phase 04 establishes the **framework** without real execution:

1. **Simulated Execution**: Actions are simulated, not actually performed
2. **In-Memory Storage**: Records stored in memory (not persistent)
3. **Serial Strategy**: Only serial execution implemented
4. **Basic Validation**: Advanced validation rules are placeholders

These limitations are **intentional** to allow design validation before implementing dangerous operations.

## Next Steps (Phase 05+)

1. **Real Execution**: Integrate with kubectl, systemctl, etc.
2. **Persistent Storage**: Database-backed execution record store
3. **Parallel Execution**: Execute independent actions concurrently
4. **Advanced Validation**: Plugin-specific validation rules
5. **Notification Integration**: Alert operators on execution results

## Acceptance Criteria

All Phase 04 acceptance criteria met:

- [x] **AC-1**: AutoFix disabled by default (opt-in) ✅
- [x] **AC-2**: All executions produce ExecutionRecords ✅
- [x] **AC-3**: Analyzer cannot directly call ExecuteFix ✅
- [x] **AC-4**: Integration tests pass ✅
- [x] **AC-5**: Validation always runs before execution ✅
- [x] **AC-6**: High-risk operations require approval ✅
- [x] **AC-7**: Dry-run mode available ✅

## Documentation

- **[Design Document](./design-autofix-execution.md)**: Complete architecture and design decisions
- **[API Guide](./api-guide.md)**: Detailed API reference with examples
- **[Implementation Summary](./implementation-summary.md)**: Code changes and file structure

## Resources

### Code Locations

```
internal/core/execution/
├── manager.go          # AutoFixManager implementation
├── manager_test.go     # Unit tests (7 test cases)
├── types.go            # Data structures
├── executor.go         # Legacy execution manager
├── planner.go          # Execution planner
└── risk_rules.go       # Risk assessment rules

internal/core/diagnosis/
└── orchestrator.go     # AutoFix integration

test/integration/
└── diagnosis_orchestrator_flow_test.go  # Integration tests

docs/round5/phase04/
├── README.md                      # This file
├── design-autofix-execution.md    # Design document
└── api-guide.md                   # API reference
```

### Running Tests

```bash
# Quick test
go test ./internal/core/execution -v -short

# Full test suite
go test ./internal/core/execution ./test/integration -v

# Specific test
go test ./internal/core/execution -v -run TestExecutionManager_ValidateFirst

# With coverage
go test ./internal/core/execution -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Getting Help

- **Design Questions**: See [design-autofix-execution.md](./design-autofix-execution.md)
- **API Usage**: See [api-guide.md](./api-guide.md)
- **Test Failures**: Check test output and verify AutoFix is enabled
- **Configuration**: Review AutoFixOptions defaults

## Conclusion

Phase 04 successfully establishes the **governance framework** for AutoFix with:

✅ **Safety First**: Opt-in, validation, risk assessment  
✅ **Audit Trail**: Complete execution records  
✅ **Testable**: Comprehensive test coverage  
✅ **Documented**: Design, API, and usage guides  
✅ **Extensible**: Ready for real execution in future phases  

The system is now ready for careful, incremental addition of actual execution capabilities.
