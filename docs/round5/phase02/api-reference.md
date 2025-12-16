# Phase 02 API Reference

## Package: `internal/core/analysis`

### Interface: Analyzer

```go
type Analyzer interface {
    // Name returns the unique identifier of the analyzer
    Name() string
    
    // Analyze performs analysis on collected data and returns issues found
    Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error)
}
```

### Type: AnalysisResult

```go
type AnalysisResult struct {
    AnalyzerName string
    Issues       []report.ReportIssue
    Metadata     map[string]interface{}
}
```

**Constructor:**
```go
func NewAnalysisResult(analyzerName string) *AnalysisResult
```

---

## Package: `internal/core/diagnosis`

### Type: Orchestrator

```go
type Orchestrator struct {
    pluginManager interfaces.PluginManager
    analyzers     []analysis.Analyzer
    logger        *logger.Logger
}
```

**Constructor:**
```go
func NewOrchestrator(
    pluginManager interfaces.PluginManager,
    analyzers []analysis.Analyzer,
) *Orchestrator
```

**Methods:**

#### RunDiagnosis

```go
func (o *Orchestrator) RunDiagnosis(
    ctx context.Context,
    req *models.DiagnosisRequest,
    progress chan<- interfaces.DiagnosisProgress,
) (*report.DiagnosisReport, error)
```

Executes the complete diagnosis pipeline:
1. Data collection from plugins
2. Analysis through all registered analyzers
3. Report generation with aggregated results

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `req`: Diagnosis request specifying target middleware and instance
- `progress`: Channel for real-time progress updates (buffered channel recommended)

**Returns:**
- `*report.DiagnosisReport`: Structured diagnosis report with issues and metadata
- `error`: Error if collection fails (analyzer errors don't fail the pipeline)

**Example:**
```go
orchestrator := diagnosis.NewOrchestrator(pluginManager, analyzers)
progress := make(chan interfaces.DiagnosisProgress, 10)
report, err := orchestrator.RunDiagnosis(ctx, req, progress)
```

---

### Type: RuleBasedAnalyzer

```go
type RuleBasedAnalyzer struct {
    logger *logger.Logger
}
```

**Constructor:**
```go
func NewRuleBasedAnalyzer() *RuleBasedAnalyzer
```

**Implements:**
- `analysis.Analyzer` interface
- `interfaces.DiagnosisAnalyzer` interface (legacy compatibility)

**Methods:**

#### Name

```go
func (a *RuleBasedAnalyzer) Name() string
```

Returns: `"RuleBasedAnalyzer"`

#### Analyze

```go
func (a *RuleBasedAnalyzer) Analyze(
    ctx context.Context,
    data *models.CollectedData,
) (*analysis.AnalysisResult, error)
```

Performs rule-based analysis on collected data.

**Current Rules:**
- CPU usage > 80% → High severity
- Memory usage > 85% → High severity
- Error log count > 10 → Medium severity

---

## Package: `internal/core/report`

### Type: DiagnosisReport

```go
type DiagnosisReport struct {
    ID        string                 // Unique report identifier
    Timestamp time.Time              // Report generation time
    Target    TargetInfo             // Target middleware information
    Status    DiagnosisStatus        // Overall health status
    Summary   string                 // High-level summary
    Issues    []ReportIssue          // Issues detected
    Metrics   map[string]interface{} // Key metrics
    Metadata  ReportMetadata         // Context metadata
}
```

**Methods:**

#### ToJSON

```go
func (r *DiagnosisReport) ToJSON() ([]byte, error)
```

Serializes the report to JSON format.

**Example:**
```go
jsonData, err := report.ToJSON()
if err != nil {
    return err
}
fmt.Println(string(jsonData))
```

---

### Type: ReportIssue

```go
type ReportIssue struct {
    ID          string        // Unique issue identifier
    Source      string        // Analyzer that detected the issue
    Title       string        // Short description
    Severity    IssueSeverity // Severity level
    Description string        // Detailed explanation
    Evidence    []Evidence    // Supporting evidence
    Suggestions []Suggestion  // Recommended actions
    Category    string        // Issue category
}
```

**Helper Functions:**

#### ConvertIssue

```go
func ConvertIssue(legacy *models.Issue, source string) ReportIssue
```

Converts legacy models.Issue to ReportIssue format.

---

### Type: IssueSeverity

```go
type IssueSeverity string

const (
    SeverityCritical IssueSeverity = "critical"
    SeverityHigh     IssueSeverity = "high"
    SeverityMedium   IssueSeverity = "medium"
    SeverityLow      IssueSeverity = "low"
    SeverityInfo     IssueSeverity = "info"
)
```

---

### Type: DiagnosisStatus

```go
type DiagnosisStatus string

const (
    StatusHealthy  DiagnosisStatus = "Healthy"
    StatusWarning  DiagnosisStatus = "Warning"
    StatusCritical DiagnosisStatus = "Critical"
)
```

---

### Type: Evidence

```go
type Evidence struct {
    Type        string      // Type of evidence (metric, log, config, etc.)
    Key         string      // Evidence key/identifier
    Value       interface{} // Evidence value
    Description string      // Human-readable description
}
```

---

### Type: Suggestion

```go
type Suggestion struct {
    Description string   // Recommended action description
    Priority    string   // Priority level (high, medium, low)
    FixHint     *FixHint // Optional automated fix hint
}
```

---

### Type: FixHint

```go
type FixHint struct {
    CanAutoFix bool                   // Whether automated fix is possible
    Command    string                 // Command to execute
    Parameters map[string]interface{} // Command parameters
    RiskLevel  string                 // Risk assessment (low, medium, high)
}
```

---

### Type: TargetInfo

```go
type TargetInfo struct {
    Middleware enum.MiddlewareType // Middleware type
    Instance   string               // Instance identifier
    Namespace  string               // K8s namespace or bare-metal location
}
```

---

### Type: ReportMetadata

```go
type ReportMetadata struct {
    AnalyzerCount int                    // Number of analyzers used
    CollectedAt   time.Time              // Data collection timestamp
    Custom        map[string]interface{} // Custom metadata
}
```

---

## Package: `internal/core/interfaces`

### Type: DiagnosisProgress

```go
type DiagnosisProgress struct {
    Step    string // Current step (Collection, Analysis, Reporting)
    Status  string // Status (InProgress, Completed, Failed)
    Message string // Detailed status message
}
```

**Usage:**
```go
progress := make(chan interfaces.DiagnosisProgress, 10)
go func() {
    for p := range progress {
        log.Printf("[%s] %s: %s", p.Step, p.Status, p.Message)
    }
}()
```

---

## Usage Patterns

### Pattern 1: Basic Diagnosis

```go
import (
    "context"
    "github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Setup
pluginManager := initPluginManager()
analyzer := diagnosis.NewRuleBasedAnalyzer()
orchestrator := diagnosis.NewOrchestrator(pluginManager, []analysis.Analyzer{analyzer})

// Execute
progress := make(chan interfaces.DiagnosisProgress, 10)
report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)
```

### Pattern 2: Custom Analyzer

```go
type MyAnalyzer struct{}

func (a *MyAnalyzer) Name() string {
    return "MyAnalyzer"
}

func (a *MyAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
    result := analysis.NewAnalysisResult(a.Name())
    // Custom logic here
    return result, nil
}

// Usage
orchestrator := diagnosis.NewOrchestrator(pm, []analysis.Analyzer{&MyAnalyzer{}})
```

### Pattern 3: Multiple Analyzers

```go
analyzers := []analysis.Analyzer{
    diagnosis.NewRuleBasedAnalyzer(),
    &MyCustomAnalyzer{},
    &AnotherAnalyzer{},
}
orchestrator := diagnosis.NewOrchestrator(pluginManager, analyzers)
```

### Pattern 4: Progress Monitoring

```go
progress := make(chan interfaces.DiagnosisProgress, 10)
done := make(chan bool)

go func() {
    for p := range progress {
        switch p.Status {
        case "InProgress":
            fmt.Printf("⏳ %s: %s\n", p.Step, p.Message)
        case "Completed":
            fmt.Printf("✅ %s: %s\n", p.Step, p.Message)
        case "Failed":
            fmt.Printf("❌ %s: %s\n", p.Step, p.Message)
        }
    }
    done <- true
}()

report, err := orchestrator.RunDiagnosis(ctx, req, progress)
close(progress)
<-done
```

### Pattern 5: Report Handling

```go
report, err := orchestrator.RunDiagnosis(ctx, req, progress)
if err != nil {
    return err
}

// Check status
switch report.Status {
case report.StatusHealthy:
    fmt.Println("✅ All systems healthy")
case report.StatusWarning:
    fmt.Printf("⚠️  Found %d issues\n", len(report.Issues))
case report.StatusCritical:
    fmt.Printf("🚨 Critical issues detected: %d\n", len(report.Issues))
}

// Process issues
for _, issue := range report.Issues {
    fmt.Printf("\n[%s] %s\n", issue.Severity, issue.Title)
    for _, suggestion := range issue.Suggestions {
        fmt.Printf("  → %s\n", suggestion.Description)
        if suggestion.FixHint != nil && suggestion.FixHint.CanAutoFix {
            fmt.Printf("    Auto-fix: %s\n", suggestion.FixHint.Command)
        }
    }
}

// Export to JSON
jsonData, _ := report.ToJSON()
os.WriteFile("report.json", jsonData, 0644)
```

---

## Error Handling

### Orchestrator Errors

```go
report, err := orchestrator.RunDiagnosis(ctx, req, progress)
if err != nil {
    // Collection phase failed (critical error)
    log.Fatalf("Diagnosis failed: %v", err)
}
```

### Analyzer Errors

Analyzer errors don't fail the pipeline:
```go
// In analyzer implementation
func (a *MyAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
    if data == nil {
        // Return error - will be logged but won't stop other analyzers
        return nil, fmt.Errorf("data is nil")
    }
    // ...
}
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

report, err := orchestrator.RunDiagnosis(ctx, req, progress)
if err == context.DeadlineExceeded {
    log.Println("Diagnosis timed out")
}
```

---

## Testing APIs

### Mock Analyzer

```go
type mockAnalyzer struct {
    name   string
    issues []report.ReportIssue
}

func (m *mockAnalyzer) Name() string { return m.name }

func (m *mockAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
    result := analysis.NewAnalysisResult(m.name)
    result.Issues = m.issues
    return result, nil
}
```

### Mock Plugin Manager

```go
type mockPluginManager struct {
    data *models.CollectedData
}

func (m *mockPluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
    return m.data, nil
}
```

---

## Version History

**v1.0 (Phase 02):**
- Initial implementation
- Rule-based analyzer
- Three-stage orchestrator
- Unified report structure

**Planned (Phase 03):**
- AI-enhanced analyzer
- LLM integration
- Natural language explanations

---

## See Also

- [Implementation Summary](./implementation-summary.md)
- [Developer Guide](./guide-phase02.md)
- [Architecture Document](../../architecture.md)
