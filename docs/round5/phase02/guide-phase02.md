# Phase 02 Developer Guide: Diagnosis Pipeline Orchestration

## Quick Start

### What Was Built

Phase 02 establishes the **diagnosis main pipeline** with:
1. **Stable Analyzer abstraction** - interface for all analysis implementations
2. **Three-stage Orchestrator** - Collection → Analysis → Report
3. **Unified Report structure** - consistent output for all consumers
4. **Comprehensive testing** - unit + integration tests

### Project Structure

```
internal/core/
├── analysis/
│   └── analyzer.go              # NEW: Analyzer interface
├── diagnosis/
│   ├── orchestrator.go          # NEW: Three-stage orchestrator
│   ├── orchestrator_test.go     # NEW: Unit tests
│   └── rule_analyzer.go         # UPDATED: Implements Analyzer
├── report/
│   └── diagnosis_report.go      # NEW: Unified report structures
└── models/
    └── diagnosis.go             # EXISTING: Legacy models

test/integration/
└── diagnosis_orchestrator_flow_test.go  # NEW: Integration tests

docs/round5/phase02/
├── implementation-summary.md    # NEW: Implementation details
└── guide-phase02.md            # NEW: This guide
```

---

## Core Components

### 1. Analyzer Interface

**Location:** `internal/core/analysis/analyzer.go`

**Purpose:** Provides a stable contract for all analysis implementations.

**Interface:**
```go
type Analyzer interface {
    Name() string
    Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error)
}
```

**Why It Matters:**
- Decouples analysis logic from data collection
- Allows multiple analysis strategies to coexist
- Easy to mock for testing
- Future-proof for AI/RAG integration

**Example Usage:**
```go
// Create an analyzer
analyzer := diagnosis.NewRuleBasedAnalyzer()

// Use it in orchestrator
orchestrator := diagnosis.NewOrchestrator(pluginManager, []analysis.Analyzer{analyzer})
```

---

### 2. Diagnosis Orchestrator

**Location:** `internal/core/diagnosis/orchestrator.go`

**Purpose:** Coordinates the complete diagnosis pipeline with clear stage boundaries.

**Key Method:**
```go
func (o *Orchestrator) RunDiagnosis(
    ctx context.Context,
    req *models.DiagnosisRequest,
    progress chan<- interfaces.DiagnosisProgress,
) (*report.DiagnosisReport, error)
```

**Three Stages:**

#### Stage 1: Collection
```go
// Invokes plugin manager to collect data
data, err := o.pluginManager.CollectData(ctx, req)
// Handles collection errors (critical - stops pipeline)
```

#### Stage 2: Analysis
```go
// Runs all analyzers sequentially
for _, analyzer := range o.analyzers {
    result, err := analyzer.Analyze(ctx, data)
    // Logs errors but continues with other analyzers
}
```

#### Stage 3: Report Generation
```go
// Aggregates results into unified report
report := o.buildReport(req, data, analysisResults)
```

**Progress Reporting:**
```go
progress := make(chan interfaces.DiagnosisProgress, 10)
go func() {
    for p := range progress {
        fmt.Printf("[%s] %s: %s\n", p.Step, p.Status, p.Message)
    }
}()
report, err := orchestrator.RunDiagnosis(ctx, req, progress)
```

**Error Handling:**
- **Collection errors:** Stop pipeline, return error
- **Analyzer errors:** Log warning, continue with other analyzers
- All errors reported via progress channel

---

### 3. Unified Report Structure

**Location:** `internal/core/report/diagnosis_report.go`

**Purpose:** Provides structured output consumable by CLI, API, and Web interfaces.

**Key Structures:**

#### DiagnosisReport
```go
type DiagnosisReport struct {
    ID        string           // Unique identifier
    Timestamp time.Time        // When diagnosis completed
    Target    TargetInfo       // What was diagnosed
    Status    DiagnosisStatus  // Overall health
    Summary   string           // High-level overview
    Issues    []ReportIssue    // Problems found
    Metrics   map[string]interface{}
    Metadata  ReportMetadata   // Context info
}
```

#### ReportIssue
```go
type ReportIssue struct {
    ID          string
    Source      string         // Which analyzer found this
    Title       string
    Severity    IssueSeverity  // Critical/High/Medium/Low
    Description string
    Evidence    []Evidence     // Supporting data
    Suggestions []Suggestion   // Recommendations
    Category    string
}
```

#### Suggestion with FixHint
```go
type Suggestion struct {
    Description string
    Priority    string
    FixHint     *FixHint
}

type FixHint struct {
    CanAutoFix bool
    Command    string
    Parameters map[string]interface{}
    RiskLevel  string
}
```

**JSON Serialization:**
```go
report := orchestrator.RunDiagnosis(ctx, req, progress)
jsonData, err := report.ToJSON()
// Use in API responses, file output, etc.
```

---

## Usage Examples

### Example 1: Basic Diagnosis Flow

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
    "github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

func main() {
    // Setup
    pluginManager := initPluginManager()
    ruleAnalyzer := diagnosis.NewRuleBasedAnalyzer()
    orchestrator := diagnosis.NewOrchestrator(pluginManager, []analysis.Analyzer{ruleAnalyzer})
    
    // Create diagnosis request
    req := &models.DiagnosisRequest{
        TargetMiddleware: enum.Redis,
        Instance:         "redis-master-001",
        Namespace:        "production",
    }
    
    // Setup progress monitoring
    progress := make(chan interfaces.DiagnosisProgress, 10)
    go func() {
        for p := range progress {
            fmt.Printf("[%s] %s: %s\n", p.Step, p.Status, p.Message)
        }
    }()
    
    // Run diagnosis
    report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)
    if err != nil {
        fmt.Printf("Diagnosis failed: %v\n", err)
        return
    }
    
    // Display results
    fmt.Printf("\n=== Diagnosis Report ===\n")
    fmt.Printf("Status: %s\n", report.Status)
    fmt.Printf("Issues Found: %d\n", len(report.Issues))
    
    for _, issue := range report.Issues {
        fmt.Printf("\n[%s] %s\n", issue.Severity, issue.Title)
        fmt.Printf("  Description: %s\n", issue.Description)
        fmt.Printf("  Source: %s\n", issue.Source)
        
        for _, suggestion := range issue.Suggestions {
            fmt.Printf("  → %s\n", suggestion.Description)
        }
    }
}
```

### Example 2: Custom Analyzer Implementation

```go
package mypackage

import (
    "context"
    
    "github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
    "github.com/kubestack-ai/kubestack-ai/internal/core/report"
)

// CustomAnalyzer implements custom analysis logic
type CustomAnalyzer struct {
    threshold float64
}

func NewCustomAnalyzer(threshold float64) *CustomAnalyzer {
    return &CustomAnalyzer{threshold: threshold}
}

func (a *CustomAnalyzer) Name() string {
    return "CustomAnalyzer"
}

func (a *CustomAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
    result := analysis.NewAnalysisResult(a.Name())
    
    // Custom analysis logic
    if data.Metrics != nil {
        if cpuUsage, ok := data.Metrics.Data["cpu_usage_percent"].(float64); ok {
            if cpuUsage > a.threshold {
                issue := report.ReportIssue{
                    ID:          "custom-cpu-high",
                    Source:      a.Name(),
                    Title:       "High CPU Usage Detected",
                    Severity:    report.SeverityHigh,
                    Description: fmt.Sprintf("CPU usage %.2f%% exceeds threshold %.2f%%", cpuUsage, a.threshold),
                    Category:    "Performance",
                    Evidence: []report.Evidence{
                        {
                            Type:  "metric",
                            Key:   "cpu_usage_percent",
                            Value: cpuUsage,
                        },
                    },
                    Suggestions: []report.Suggestion{
                        {
                            Description: "Consider scaling horizontally or optimizing workload",
                            Priority:    "high",
                        },
                    },
                }
                result.Issues = append(result.Issues, issue)
            }
        }
    }
    
    return result, nil
}

// Usage:
// customAnalyzer := NewCustomAnalyzer(70.0)
// orchestrator := diagnosis.NewOrchestrator(pm, []analysis.Analyzer{customAnalyzer})
```

### Example 3: Multiple Analyzers

```go
func runMultiAnalyzerDiagnosis() {
    pluginManager := initPluginManager()
    
    // Create multiple analyzers
    ruleAnalyzer := diagnosis.NewRuleBasedAnalyzer()
    customAnalyzer1 := NewCustomAnalyzer(70.0)
    customAnalyzer2 := NewAnotherAnalyzer()
    
    // Register all analyzers
    analyzers := []analysis.Analyzer{
        ruleAnalyzer,
        customAnalyzer1,
        customAnalyzer2,
    }
    
    orchestrator := diagnosis.NewOrchestrator(pluginManager, analyzers)
    
    // Run diagnosis - all analyzers will be executed
    report, err := orchestrator.RunDiagnosis(ctx, req, progress)
    
    // Report will contain issues from all analyzers
    // Metadata shows analyzer count
    fmt.Printf("Analyzers used: %d\n", report.Metadata.AnalyzerCount)
}
```

---

## Testing

### Running Tests

**Unit Tests:**
```bash
cd /workspace/project/kubestack-ai
go test ./internal/core/diagnosis/ -v
```

**Integration Tests:**
```bash
go test ./test/integration/diagnosis_orchestrator_flow_test.go -v
```

**All Tests:**
```bash
go test ./... -v
```

### Writing Tests

**Example Unit Test:**
```go
func TestMyAnalyzer(t *testing.T) {
    analyzer := NewMyAnalyzer()
    
    // Create mock data
    data := &models.CollectedData{
        Metrics: &models.MetricsData{
            Data: map[string]interface{}{
                "cpu_usage_percent": 90.0,
            },
        },
    }
    
    // Run analysis
    result, err := analyzer.Analyze(context.Background(), data)
    
    // Assert results
    if err != nil {
        t.Fatalf("Analyze failed: %v", err)
    }
    
    if len(result.Issues) == 0 {
        t.Error("Expected issues to be detected")
    }
    
    if result.Issues[0].Severity != report.SeverityHigh {
        t.Errorf("Expected high severity, got %s", result.Issues[0].Severity)
    }
}
```

---

## Migration Guide

### From Legacy DiagnosisManager to Orchestrator

**Old Code:**
```go
manager := diagnosis.NewManager(pluginManager)
result, err := manager.Diagnose(ctx, req)
```

**New Code:**
```go
analyzer := diagnosis.NewRuleBasedAnalyzer()
orchestrator := diagnosis.NewOrchestrator(pluginManager, []analysis.Analyzer{analyzer})
progress := make(chan interfaces.DiagnosisProgress, 10)
report, err := orchestrator.RunDiagnosis(ctx, req, progress)
```

**Benefits:**
- Clear separation of concerns
- Progress reporting
- Multiple analyzers support
- Structured output

---

## Best Practices

### 1. Analyzer Design

✅ **Do:**
- Keep analyzers focused on one type of analysis
- Return clear, actionable issues
- Include evidence with every issue
- Provide specific suggestions

❌ **Don't:**
- Mix data collection with analysis
- Return generic error messages
- Depend on other analyzers
- Modify the input data

### 2. Error Handling

```go
// Analyzer errors should be descriptive
func (a *MyAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
    if data == nil {
        return nil, fmt.Errorf("analyzer %s: collected data is nil", a.Name())
    }
    
    result := analysis.NewAnalysisResult(a.Name())
    
    // Safe access with type assertions
    if metrics, ok := data.Metrics.Data["cpu"].(float64); ok {
        // Process metrics
    }
    
    return result, nil
}
```

### 3. Progress Reporting

```go
// In orchestrator, report progress at key stages
o.reportProgress(progress, "Collection", "InProgress", "Collecting data from plugins...")
data, err := o.pluginManager.CollectData(ctx, req)
if err != nil {
    o.reportProgress(progress, "Collection", "Failed", fmt.Sprintf("Collection failed: %v", err))
    return nil, err
}
o.reportProgress(progress, "Collection", "Completed", "Data collection completed successfully")
```

### 4. Report Structure

```go
// Always provide complete issue information
issue := report.ReportIssue{
    ID:          generateID(),           // Unique identifier
    Source:      a.Name(),               // Analyzer name
    Title:       "Short Description",    // Clear title
    Severity:    report.SeverityHigh,   // Appropriate severity
    Description: "Detailed explanation", // Full context
    Category:    "Performance",          // Issue category
    Evidence: []report.Evidence{         // Supporting data
        {Type: "metric", Key: "cpu", Value: 95.0},
    },
    Suggestions: []report.Suggestion{    // Actionable advice
        {Description: "Scale up", Priority: "high"},
    },
}
```

---

## Troubleshooting

### Issue: Analyzer Not Being Called

**Check:**
1. Analyzer is in the orchestrator's analyzer list
2. No collection errors stopping the pipeline
3. Context not canceled

```go
analyzers := []analysis.Analyzer{myAnalyzer}  // Ensure analyzer is included
orchestrator := diagnosis.NewOrchestrator(pm, analyzers)
```

### Issue: No Progress Updates

**Check:**
1. Progress channel buffer size is adequate
2. Channel is being read
3. Channel not closed prematurely

```go
progress := make(chan interfaces.DiagnosisProgress, 10)  // Buffer size 10
go func() {
    for p := range progress {  // Must read from channel
        fmt.Printf("%+v\n", p)
    }
}()
```

### Issue: Report Status Incorrect

**Check:**
1. Issue severities are set correctly
2. Status calculation logic in buildReport()

```go
// Status is calculated automatically from issue severities
// Critical issues → "Critical"
// High issues → "Warning"  
// Others → "Healthy"
```

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

## References

- **Architecture Document:** `docs/architecture.md`
- **Implementation Summary:** `docs/round5/phase02/implementation-summary.md`
- **Analyzer Interface:** `internal/core/analysis/analyzer.go`
- **Orchestrator:** `internal/core/diagnosis/orchestrator.go`
- **Report Structures:** `internal/core/report/diagnosis_report.go`
- **Unit Tests:** `internal/core/diagnosis/orchestrator_test.go`
- **Integration Tests:** `test/integration/diagnosis_orchestrator_flow_test.go`

---

## Support

For questions or issues:
1. Check the implementation summary document
2. Review test cases for examples
3. Examine the architecture document
4. Open an issue on GitHub

---

**Phase 02 Status:** ✅ **COMPLETED**

All acceptance criteria met:
- ✅ Analyzer abstraction stable
- ✅ Three-stage orchestrator implemented
- ✅ Unified report structure defined
- ✅ Tests passing
- ✅ Documentation complete
- ✅ Binary compiles and runs
