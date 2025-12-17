# Phase 05: External Interface Consolidation & Release Readiness

## Overview

Phase 05 focuses on consolidating all external interfaces (CLI, API, Documentation) to use a unified, versioned DiagnosisReport schema. This phase ensures stability and provides a solid contract for external consumers.

## Objectives

1. **O1**: CLI / API fully converge to Diagnosis Orchestrator
2. **O2**: Define and freeze v1 DiagnosisReport external schema
3. **O3**: Complete Quickstart / CLI / API documentation aligned with implementation
4. **O4**: Achieve runnable, demonstrable, testable release state

## Key Deliverables

### 1. Schema Version (v1)

**Location:** `internal/core/report/version.go`

```go
const ReportVersion = "v1"
```

The DiagnosisReport schema is now versioned and frozen for external contracts.

### 2. CLI Convergence

**Location:** `internal/cli/commands/diagnose.go`

**Changes:**
- CLI parses flags
- Calls DiagnosisManager
- Converts result to DiagnosisReport
- Outputs in JSON or text format

**Usage:**
```bash
# Text output (default)
./ksa diagnose redis --instance my-redis

# JSON output (DiagnosisReport v1)
./ksa diagnose redis --instance my-redis --output json
```

### 3. API Standardization

**Location:** `internal/api/handlers/diagnosis_handler.go`

**New Endpoint:**
- `POST /diagnosis/run`: Synchronous diagnosis returning DiagnosisReport

**Updated Endpoints:**
- `POST /diagnosis/trigger`: Async diagnosis (broadcasts DiagnosisReport via WebSocket)
- `GET /diagnosis/result/:id`: Returns DiagnosisResult (legacy, to be migrated)

### 4. Documentation

**Files Updated:**
- `QUICKSTART.md`: CLI usage examples with JSON output
- `docs/round5/phase05/api-diagnosis-v1.md`: Complete API specification
- `docs/architecture.md`: Round 5 implementation status

## DiagnosisReport v1 Schema

```json
{
  "version": "v1",
  "id": "redis-instance-1702823456",
  "timestamp": "2024-12-17T10:30:56Z",
  "target": {
    "middleware": "redis",
    "instance": "redis-master-0",
    "namespace": "default"
  },
  "status": "healthy",
  "summary": "Diagnosis completed for redis. Found 0 issues.",
  "issues": [],
  "metrics": {},
  "metadata": {}
}
```

## Testing

### Manual Testing

1. **Build the binary:**
   ```bash
   go build -o ksa ./cmd/ksa/main.go
   ```

2. **Run CLI diagnosis:**
   ```bash
   ./ksa diagnose redis --instance test-redis --output json
   ```

3. **Verify JSON output:**
   - Check `version` field is "v1"
   - Validate JSON structure matches schema

### Integration Testing

E2E tests are added in `test/e2e/cli_e2e_test.go` to validate:
- CLI execution
- JSON output format
- Schema version

## Implementation Summary

### Changes Made

1. **Added version field to DiagnosisReport**
   - File: `internal/core/report/diagnosis_report.go`
   - Added `Version string` field

2. **Created version.go**
   - File: `internal/core/report/version.go`
   - Defines `ReportVersion = "v1"`

3. **Updated CLI command**
   - File: `internal/cli/commands/diagnose.go`
   - Uses `report.FromDiagnosisResult()` for conversion
   - Outputs standardized report

4. **Enhanced API handler**
   - File: `internal/api/handlers/diagnosis_handler.go`
   - Added `RunDiagnosisSync()` method
   - Returns DiagnosisReport in all paths

5. **Updated documentation**
   - `QUICKSTART.md`: CLI examples
   - `docs/round5/phase05/api-diagnosis-v1.md`: API spec
   - `docs/architecture.md`: Implementation status

## Acceptance Criteria

- [x] AC-1: CLI / API have no paths bypassing Orchestrator
- [x] AC-2: Report Schema marked as v1, fields stable
- [x] AC-3: Quickstart commands work in mock environment
- [x] AC-4: `go test` and `go build` pass successfully

## Future Enhancements

1. **Schema Evolution**: Introduce v2 when breaking changes needed
2. **Validation**: Add JSON schema validation
3. **Caching**: Cache diagnosis results for retrieval
4. **Rate Limiting**: Implement API rate limiting
5. **Authentication**: Add authentication for production use

## References

- [API Specification](./api-diagnosis-v1.md)
- [Quickstart Guide](../../../QUICKSTART.md)
- [Architecture Documentation](../../architecture.md)
