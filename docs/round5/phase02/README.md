# Phase 02: Diagnosis Pipeline Orchestration

**Status:** ✅ **COMPLETED**  
**Branch:** `feat/round5-phase02-diagnosis-pipeline`  
**Date:** 2025-12-16

---

## Overview

Phase 02 establishes the **diagnosis main pipeline** with clear stage boundaries, stable analyzer abstraction, and unified report structure. This phase creates the foundation for parallel evolution of rule-based, AI-enhanced, and RAG-based analysis approaches.

## Quick Links

📖 **Documentation:**
- [Implementation Summary](./implementation-summary.md) - Complete implementation details
- [Developer Guide](./guide-phase02.md) - How to use and extend the pipeline
- [API Reference](./api-reference.md) - Complete API documentation

🏗️ **Architecture:**
- [Architecture Document](../../architecture.md) - Updated with implementation notes

🧪 **Code:**
- `internal/core/analysis/analyzer.go` - Analyzer interface
- `internal/core/diagnosis/orchestrator.go` - Three-stage orchestrator
- `internal/core/report/diagnosis_report.go` - Unified report structure
- `internal/core/diagnosis/rule_analyzer.go` - Rule-based analyzer implementation

---

## What Was Built

### 1. Analyzer Interface ✅
- **Location:** `internal/core/analysis/analyzer.go`
- **Purpose:** Stable contract for all analysis implementations
- **Benefits:** Decouples analysis from data collection, supports multiple strategies

### 2. Three-Stage Orchestrator ✅
- **Location:** `internal/core/diagnosis/orchestrator.go`
- **Stages:**
  1. **Collection:** Gather data from plugins
  2. **Analysis:** Process data through analyzers
  3. **Reporting:** Generate structured report
- **Features:** Progress reporting, graceful error handling

### 3. Unified Report Structure ✅
- **Location:** `internal/core/report/diagnosis_report.go`
- **Purpose:** Consistent output for CLI/API/Web
- **Includes:** Issues, evidence, suggestions, fix hints, metadata

### 4. Rule-Based Analyzer v1 ✅
- **Location:** `internal/core/diagnosis/rule_analyzer.go`
- **Rules:** CPU threshold, memory threshold, error log count
- **Features:** Implements new interface, backward compatible

### 5. Comprehensive Testing ✅
- **Unit Tests:** `internal/core/diagnosis/orchestrator_test.go`
- **Integration Tests:** `test/integration/diagnosis_orchestrator_flow_test.go`
- **Status:** All tests passing

---

## Key Achievements

| Goal | Status | Notes |
|------|--------|-------|
| GAP-D: Analyzer abstraction | ✅ | Stable interface created |
| GAP-E: Stage boundaries | ✅ | Three-stage pipeline clear |
| GAP-F: Report structure | ✅ | Unified structure implemented |
| Testing | ✅ | Unit + integration tests pass |
| Documentation | ✅ | Complete docs with examples |
| Build & Run | ✅ | Binary compiles and runs |

---

## Quick Start

### Basic Usage

```go
// Setup
pluginManager := initPluginManager()
analyzer := diagnosis.NewRuleBasedAnalyzer()
orchestrator := diagnosis.NewOrchestrator(pluginManager, []analysis.Analyzer{analyzer})

// Execute diagnosis
req := &models.DiagnosisRequest{
    TargetMiddleware: enum.Redis,
    Instance:         "redis-master-001",
}

progress := make(chan interfaces.DiagnosisProgress, 10)
report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

// Handle results
fmt.Printf("Status: %s\n", report.Status)
fmt.Printf("Issues: %d\n", len(report.Issues))
```

### Custom Analyzer

```go
type MyAnalyzer struct{}

func (a *MyAnalyzer) Name() string {
    return "MyAnalyzer"
}

func (a *MyAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
    result := analysis.NewAnalysisResult(a.Name())
    // Your analysis logic here
    return result, nil
}

// Use it
orchestrator := diagnosis.NewOrchestrator(pm, []analysis.Analyzer{&MyAnalyzer{}})
```

---

## Architecture Diagram

```
DiagnosisRequest
    ↓
[Orchestrator]
    ↓
┌─────────────┐
│ Collection  │ ← Plugin Manager → Metrics + Logs + Config
└─────────────┘
    ↓
┌─────────────┐
│  Analysis   │ ← Analyzer 1 → AnalysisResult 1
│             │ ← Analyzer 2 → AnalysisResult 2
│             │ ← Analyzer N → AnalysisResult N
└─────────────┘
    ↓
┌─────────────┐
│  Reporting  │ → DiagnosisReport
└─────────────┘
    ↓
┌─────────────────────────────────┐
│         Output                  │
├─────────────────────────────────┤
│ CLI:  Formatted Text            │
│ API:  JSON Response             │
│ Web:  Structured UI             │
└─────────────────────────────────┘
```

---

## Test Results

### Unit Tests

```bash
$ go test ./internal/core/diagnosis/ -v
=== RUN   TestOrchestrator_CallOrder
--- PASS: TestOrchestrator_CallOrder (0.00s)
=== RUN   TestOrchestrator_ErrorPropagation
--- PASS: TestOrchestrator_ErrorPropagation (0.00s)
=== RUN   TestOrchestrator_ReportGeneration
--- PASS: TestOrchestrator_ReportGeneration (0.00s)
PASS
ok      github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis    0.009s
```

### Integration Tests

```bash
$ go test ./test/integration/diagnosis_orchestrator_flow_test.go -v
=== RUN   TestDiagnosis_MinimalFlow
--- PASS: TestDiagnosis_MinimalFlow (0.00s)
=== RUN   TestDiagnosis_ReportJSONSerialization
--- PASS: TestDiagnosis_ReportJSONSerialization (0.00s)
=== RUN   TestDiagnosis_MultipleAnalyzers
--- PASS: TestDiagnosis_MultipleAnalyzers (0.00s)
PASS
ok      command-line-arguments  0.010s
```

### Build & Run

```bash
$ go build -o /tmp/ksa ./cmd/ksa
$ /tmp/ksa --help
KubeStack-AI is a command-line tool that uses AI to help you diagnose,
analyze, and fix issues with your middleware infrastructure...
✅ SUCCESS
```

---

## File Structure

```
kubestack-ai/
├── docs/
│   ├── architecture.md                          # Updated with implementation notes
│   └── round5/phase02/
│       ├── README.md                            # This file
│       ├── implementation-summary.md            # Complete implementation details
│       ├── guide-phase02.md                     # Developer guide
│       └── api-reference.md                     # API documentation
├── internal/core/
│   ├── analysis/
│   │   └── analyzer.go                          # NEW: Analyzer interface
│   ├── diagnosis/
│   │   ├── orchestrator.go                      # NEW: Three-stage orchestrator
│   │   ├── orchestrator_test.go                 # NEW: Unit tests
│   │   └── rule_analyzer.go                     # UPDATED: Implements new interface
│   └── report/
│       └── diagnosis_report.go                  # NEW: Unified report structure
└── test/integration/
    └── diagnosis_orchestrator_flow_test.go      # NEW: Integration tests
```

---

## Commits

1. **6c15722** - feat: Phase 02 - Diagnosis Pipeline Orchestration Implementation
   - All core components implemented
   - Tests added and passing
   - Architecture document updated

2. **0592d16** - docs: Add comprehensive Phase 02 documentation
   - Developer guide with examples
   - Complete API reference
   - Implementation summary

---

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| AC-1: Analyzers depend only on interface | ✅ PASS |
| AC-2: CLI/API/Web use DiagnosisReport | ✅ PASS |
| AC-3: Integration tests pass | ✅ PASS |
| AC-4: Architecture docs updated | ✅ PASS |
| AC-5: Build, test, run successfully | ✅ PASS |

---

## Performance Metrics

- **Unit Test Execution:** 0.009s
- **Integration Test Execution:** 0.010s
- **Build Time:** < 10s (with dependencies)
- **Binary Size:** ~50MB

---

## Next Steps

### Phase 03: AI Enhancement (Planned)
- Implement AI-based analyzer using LLM
- Natural language explanation generation
- Context-aware root cause analysis

### Phase 04: RAG Integration (Planned)
- Knowledge base lookup analyzer
- Historical pattern matching
- Case-based reasoning

### Phase 05: AutoFix Execution (Planned)
- Fix suggestion to execution pipeline
- Risk assessment
- Automated rollback

---

## Getting Started

1. **Read the Architecture:**
   - Start with [architecture.md](../../architecture.md#诊断管道实现说明-diagnosis-pipeline-implementation-notes)
   - Review the implementation status section

2. **Follow the Guide:**
   - [Developer Guide](./guide-phase02.md) has complete usage examples
   - Includes migration guide from legacy code

3. **Check the API:**
   - [API Reference](./api-reference.md) documents all interfaces
   - Includes usage patterns and error handling

4. **Run the Tests:**
   ```bash
   go test ./internal/core/diagnosis/ -v
   go test ./test/integration/diagnosis_orchestrator_flow_test.go -v
   ```

5. **Build and Try:**
   ```bash
   go build -o ksa ./cmd/ksa
   ./ksa diagnose redis redis-master-001
   ```

---

## Support

For questions or issues:
1. Check the [implementation summary](./implementation-summary.md)
2. Review [test cases](../../test/integration/diagnosis_orchestrator_flow_test.go) for examples
3. Examine the [architecture document](../../architecture.md)
4. Open an issue on GitHub

---

## Contributors

- openhands <openhands@all-hands.dev>

---

**Phase 02 Status:** ✅ **COMPLETED AND READY FOR PHASE 03**
