# Phase 03: API & Integration Guide

**Target Audience:** Developers integrating AI analyzer into KubeStack-AI

---

## Table of Contents

1. [AI Analyzer API](#ai-analyzer-api)
2. [LLM Client Interface](#llm-client-interface)
3. [Schema Definitions](#schema-definitions)
4. [Integration Examples](#integration-examples)
5. [Testing Strategies](#testing-strategies)
6. [Error Handling](#error-handling)

---

## AI Analyzer API

### Creating an AI Analyzer

```go
import (
    "github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/llm"
)

// Create analyzer with mock client
mockClient := llm.NewMockClient()
analyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{
    Middleware: "redis",
    Instance:   "redis-master-001",
    Namespace:  "production",
})
```

### Configuration Options

```go
type AIAnalyzerConfig struct {
    Middleware string  // Middleware type (e.g., "redis", "mysql")
    Instance   string  // Instance identifier
    Namespace  string  // Kubernetes namespace or environment
}
```

### Using in Diagnosis Flow

```go
import (
    "context"
    "github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
)

// Create orchestrator with AI analyzer
orchestrator := diagnosis.NewOrchestrator(
    pluginManager,
    []analysis.Analyzer{
        ruleAnalyzer,  // Rule-based
        aiAnalyzer,    // AI-driven
    },
)

// Run diagnosis
progress := make(chan *diagnosis.ProgressUpdate, 10)
report, err := orchestrator.RunDiagnosis(ctx, request, progress)
```

---

## LLM Client Interface

### Interface Definition

```go
package llm

import (
    llmInterface "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// Client is an alias for the existing LLM client interface
type Client = llmInterface.LLMClient
```

### Mock Client (Testing)

```go
// Create mock with fixed response
mockClient := llm.NewMockClient()
mockClient.SetResponse(`{
    "summary": "High memory usage detected",
    "reasoning": "Memory metrics show sustained usage above 85%",
    "issues": [
        {
            "id": "ai-001",
            "title": "Memory Pressure",
            "severity": "High",
            "description": "Instance is running low on available memory",
            "evidence": "Current: 92%, Threshold: 85%",
            "recommendations": [
                {
                    "priority": "High",
                    "description": "Increase memory allocation",
                    "fix_hint": {
                        "type": "config_change",
                        "command": "kubectl set resources",
                        "parameters": {"memory": "4Gi"}
                    }
                }
            ]
        }
    ]
}`)

// Verify calls
if mockClient.CallCount != 1 {
    t.Errorf("Expected 1 LLM call, got %d", mockClient.CallCount)
}
```

### Mock Client Methods

| Method | Purpose | Example |
|--------|---------|---------|
| `SetResponse(json)` | Set fixed JSON response | `mockClient.SetResponse("{...}")` |
| `SetError(err)` | Simulate LLM failure | `mockClient.SetError(errors.New("timeout"))` |
| `LastRequest` | Access last request sent | `mockClient.LastRequest.Messages[0].Content` |
| `CallCount` | Track number of calls | `if mockClient.CallCount > 0 { ... }` |

---

## Schema Definitions

### Input Schema (AIInput)

```go
type AIInput struct {
    PluginData PluginDataSummary `json:"plugin_data"`
    Context    ContextInfo       `json:"context"`
}

type PluginDataSummary struct {
    Metrics      map[string]interface{} `json:"metrics"`
    Logs         []LogEntry             `json:"logs"`
    Config       map[string]interface{} `json:"config"`
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

type ContextInfo struct {
    Middleware string `json:"middleware"`
    Instance   string `json:"instance"`
    Namespace  string `json:"namespace"`
    Timestamp  int64  `json:"timestamp"`
}
```

### Output Schema (AIOutput)

```go
type AIOutput struct {
    Summary   string    `json:"summary"`
    Reasoning string    `json:"reasoning"`
    Issues    []AIIssue `json:"issues"`
}

type AIIssue struct {
    ID              string              `json:"id"`
    Title           string              `json:"title"`
    Severity        string              `json:"severity"` // "Critical", "High", "Medium", "Low", "Info"
    Description     string              `json:"description"`
    Evidence        string              `json:"evidence"`
    Recommendations []AIRecommendation  `json:"recommendations"`
}

type AIRecommendation struct {
    Priority    string              `json:"priority"`
    Description string              `json:"description"`
    FixHint     *AIFixHint          `json:"fix_hint,omitempty"`
}

type AIFixHint struct {
    Type       string                 `json:"type"`        // "config_change", "restart", "scale", "manual"
    Command    string                 `json:"command"`
    Parameters map[string]interface{} `json:"parameters"`
}
```

### Severity Mapping

```go
import "github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"

// String to enum mapping
severityMap := map[string]enum.SeverityLevel{
    "Critical": enum.SeverityCritical,
    "High":     enum.SeverityHigh,
    "Medium":   enum.SeverityMedium,
    "Low":      enum.SeverityLow,
    "Info":     enum.SeverityInfo,
}

// Case-insensitive parsing
severity := parseSeverity("high") // Returns enum.SeverityHigh
```

---

## Integration Examples

### Example 1: Basic Integration

```go
package main

import (
    "context"
    "fmt"
    "github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/llm"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

func main() {
    // Create mock LLM client
    mockClient := llm.NewMockClient()
    mockClient.SetResponse(`{
        "summary": "Redis instance healthy",
        "reasoning": "All metrics within normal ranges",
        "issues": []
    }`)

    // Create AI analyzer
    analyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{
        Middleware: "redis",
        Instance:   "redis-001",
        Namespace:  "production",
    })

    // Prepare collected data
    data := &models.CollectedData{
        Metrics: map[string]interface{}{
            "memory_usage": 45.2,
            "cpu_usage":    23.1,
        },
        Logs: []models.LogEntry{
            {Message: "Redis ready", Level: "info"},
        },
        Config: map[string]interface{}{
            "maxmemory": "2gb",
        },
    }

    // Analyze
    result, err := analyzer.Analyze(context.Background(), data)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d issues\n", len(result.Issues))
}
```

### Example 2: Multi-Analyzer Setup

```go
package main

import (
    "context"
    "github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
    "github.com/kubestack-ai/kubestack-ai/internal/core/llm"
)

func setupDiagnosis() *diagnosis.Orchestrator {
    // Rule-based analyzer
    ruleAnalyzer := diagnosis.NewRuleAnalyzer()

    // AI analyzer with mock
    mockClient := llm.NewMockClient()
    mockClient.SetResponse(`{"summary":"AI analysis","issues":[]}`)
    
    aiAnalyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{
        Middleware: "mysql",
        Instance:   "mysql-primary",
        Namespace:  "default",
    })

    // Create orchestrator with both analyzers
    return diagnosis.NewOrchestrator(
        pluginManager,
        []analysis.Analyzer{
            ruleAnalyzer,  // Fast, deterministic rules
            aiAnalyzer,    // Deep AI analysis
        },
    )
}

func runDiagnosis(orchestrator *diagnosis.Orchestrator) {
    request := &diagnosis.DiagnosisRequest{
        MiddlewareType: "mysql",
        InstanceName:   "mysql-primary",
    }

    progress := make(chan *diagnosis.ProgressUpdate, 10)
    go func() {
        for update := range progress {
            fmt.Printf("[%s] %s\n", update.Stage, update.Message)
        }
    }()

    report, err := orchestrator.RunDiagnosis(context.Background(), request, progress)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Diagnosis complete: %d issues\n", len(report.Issues))
}
```

### Example 3: Custom LLM Response Handling

```go
func testCustomResponse(t *testing.T) {
    // Mock with multiple issues
    mockClient := llm.NewMockClient()
    mockClient.SetResponse(`{
        "summary": "Performance degradation detected",
        "reasoning": "Multiple indicators point to resource exhaustion",
        "issues": [
            {
                "id": "cpu-001",
                "title": "High CPU Usage",
                "severity": "High",
                "description": "CPU usage sustained above 80%",
                "evidence": "avg_cpu=85.3%, max_cpu=92.1%",
                "recommendations": [
                    {
                        "priority": "High",
                        "description": "Scale horizontally",
                        "fix_hint": {
                            "type": "scale",
                            "command": "kubectl scale",
                            "parameters": {"replicas": 3}
                        }
                    }
                ]
            },
            {
                "id": "mem-001",
                "title": "Memory Leak Suspected",
                "severity": "Critical",
                "description": "Memory usage increasing over time",
                "evidence": "Growth rate: 1.2 MB/hour",
                "recommendations": [
                    {
                        "priority": "Critical",
                        "description": "Restart and investigate",
                        "fix_hint": {
                            "type": "restart",
                            "command": "kubectl rollout restart",
                            "parameters": {}
                        }
                    }
                ]
            }
        ]
    }`)

    analyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{
        Middleware: "redis",
    })

    result, err := analyzer.Analyze(context.Background(), testData)
    require.NoError(t, err)
    assert.Len(t, result.Issues, 2)
    
    // Verify severity mapping
    assert.Equal(t, enum.SeverityHigh, result.Issues[0].Severity)
    assert.Equal(t, enum.SeverityCritical, result.Issues[1].Severity)
}
```

---

## Testing Strategies

### Unit Testing Pattern

```go
func TestAIAnalyzer_YourTest(t *testing.T) {
    // Setup: Create mock with expected response
    mockClient := llm.NewMockClient()
    mockClient.SetResponse(`{
        "summary": "Test summary",
        "issues": [...]
    }`)

    // Create analyzer
    analyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{
        Middleware: "test-middleware",
    })

    // Execute: Run analysis
    result, err := analyzer.Analyze(context.Background(), testData)

    // Verify: Assert expectations
    require.NoError(t, err)
    assert.Equal(t, "AIAnalyzer", analyzer.Name())
    assert.Len(t, result.Issues, 1)
    assert.Equal(t, 1, mockClient.CallCount)
}
```

### Integration Testing Pattern

```go
func TestFullDiagnosisFlow(t *testing.T) {
    // Setup: Create test plugin and analyzer
    testPlugin := createTestPlugin()
    mockClient := llm.NewMockClient()
    mockClient.SetResponse(`{"summary":"Integration test","issues":[]}`)
    
    aiAnalyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{
        Middleware: "test",
    })

    // Create orchestrator
    orchestrator := diagnosis.NewOrchestrator(
        testPluginManager,
        []analysis.Analyzer{aiAnalyzer},
    )

    // Execute: Run full diagnosis
    progress := make(chan *diagnosis.ProgressUpdate, 10)
    report, err := orchestrator.RunDiagnosis(ctx, request, progress)

    // Verify: Check end-to-end flow
    require.NoError(t, err)
    assert.NotNil(t, report)
    assert.NotEmpty(t, report.ID)
    
    // Verify AI analyzer was called
    assert.Equal(t, 1, mockClient.CallCount)
}
```

### Error Handling Tests

```go
func TestAIAnalyzer_LLMError(t *testing.T) {
    mockClient := llm.NewMockClient()
    mockClient.SetError(errors.New("LLM timeout"))
    
    analyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{})
    
    result, err := analyzer.Analyze(context.Background(), testData)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "LLM timeout")
    assert.Nil(t, result)
}

func TestAIAnalyzer_InvalidJSON(t *testing.T) {
    mockClient := llm.NewMockClient()
    mockClient.SetResponse(`{invalid json}`)
    
    analyzer := analysis.NewAIAnalyzer(mockClient, analysis.AIAnalyzerConfig{})
    
    result, err := analyzer.Analyze(context.Background(), testData)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "parse AI output")
}
```

---

## Error Handling

### Error Types

| Error | Cause | Handling Strategy |
|-------|-------|-------------------|
| LLM timeout | Network/API issue | Retry with backoff, fallback to rules |
| Invalid JSON | Malformed LLM response | Log error, return empty result |
| Schema violation | Missing required fields | Use defaults, log warning |
| Context error | Cancelled/timeout | Propagate to caller |

### Error Handling Example

```go
func (a *AIAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error) {
    // Handle context cancellation
    if ctx.Err() != nil {
        return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
    }

    // Call LLM with timeout
    response, err := a.llmClient.Complete(ctx, request)
    if err != nil {
        // Log and wrap error
        log.WithError(err).Error("LLM call failed")
        return nil, fmt.Errorf("LLM analysis failed: %w", err)
    }

    // Parse JSON with error handling
    output, err := parseAIOutput(response.Content)
    if err != nil {
        log.WithError(err).WithField("response", response.Content).Error("Failed to parse LLM output")
        return nil, fmt.Errorf("failed to parse AI output: %w", err)
    }

    // Convert to issues (defensive)
    issues := convertToIssues(output.Issues)
    
    return &AnalysisResult{
        AnalyzerName: a.Name(),
        Issues:       issues,
        Metadata: map[string]interface{}{
            "llm_model": response.Model,
            "tokens":    response.Usage.TotalTokens,
        },
    }, nil
}
```

### Graceful Degradation

```go
// In orchestrator, continue if AI analyzer fails
for _, analyzer := range o.analyzers {
    result, err := analyzer.Analyze(ctx, collectedData)
    if err != nil {
        log.WithError(err).WithField("analyzer", analyzer.Name()).Warn("Analyzer failed, continuing...")
        continue // Don't fail entire diagnosis
    }
    
    allIssues = append(allIssues, result.Issues...)
}
```

---

## Best Practices

### 1. Always Use Context

```go
// Good: Pass context for cancellation
result, err := analyzer.Analyze(ctx, data)

// Bad: No cancellation support
result, err := analyzer.Analyze(nil, data)
```

### 2. Set Timeouts

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := analyzer.Analyze(ctx, data)
```

### 3. Log Errors with Context

```go
import "github.com/sirupsen/logrus"

log.WithFields(logrus.Fields{
    "analyzer":   analyzer.Name(),
    "middleware": config.Middleware,
    "instance":   config.Instance,
}).WithError(err).Error("Analysis failed")
```

### 4. Validate Inputs

```go
if data == nil {
    return nil, errors.New("collected data cannot be nil")
}

if len(data.Metrics) == 0 && len(data.Logs) == 0 {
    log.Warn("No data to analyze")
    return &AnalysisResult{Issues: []models.Issue{}}, nil
}
```

### 5. Use Type-Safe Enums

```go
// Good: Type-safe enum
severity := enum.SeverityHigh

// Bad: String constants
severity := "High" // Error-prone
```

---

## Performance Considerations

### Caching Strategies

```go
type CachedAIAnalyzer struct {
    analyzer *AIAnalyzer
    cache    map[string]*AnalysisResult
    mu       sync.RWMutex
}

func (c *CachedAIAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error) {
    cacheKey := computeHash(data)
    
    c.mu.RLock()
    if cached, ok := c.cache[cacheKey]; ok {
        c.mu.RUnlock()
        return cached, nil
    }
    c.mu.RUnlock()
    
    result, err := c.analyzer.Analyze(ctx, data)
    if err == nil {
        c.mu.Lock()
        c.cache[cacheKey] = result
        c.mu.Unlock()
    }
    
    return result, err
}
```

### Async Analysis

```go
func analyzeAsync(analyzer analysis.Analyzer, data *models.CollectedData) <-chan *AnalysisResult {
    resultChan := make(chan *AnalysisResult, 1)
    
    go func() {
        result, err := analyzer.Analyze(context.Background(), data)
        if err != nil {
            log.WithError(err).Error("Async analysis failed")
            resultChan <- nil
        } else {
            resultChan <- result
        }
        close(resultChan)
    }()
    
    return resultChan
}
```

---

## Migration from Mock to Real LLM

### Step 1: Implement Real Client

```go
// File: internal/core/llm/openai_client.go
package llm

import (
    "context"
    "github.com/sashabaranov/go-openai"
    llmInterface "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

type OpenAIClient struct {
    client *openai.Client
    model  string
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
    return &OpenAIClient{
        client: openai.NewClient(apiKey),
        model:  model,
    }
}

func (c *OpenAIClient) Complete(ctx context.Context, req *llmInterface.LLMRequest) (*llmInterface.LLMResponse, error) {
    // Implementation using OpenAI API
    // ...
}
```

### Step 2: Configuration

```go
// In config file or environment
type Config struct {
    LLM struct {
        Provider string // "mock", "openai", "gemini"
        APIKey   string
        Model    string
    }
}
```

### Step 3: Factory Pattern

```go
func NewLLMClient(config Config) llm.Client {
    switch config.LLM.Provider {
    case "mock":
        return llm.NewMockClient()
    case "openai":
        return llm.NewOpenAIClient(config.LLM.APIKey, config.LLM.Model)
    default:
        panic("unknown LLM provider")
    }
}
```

### Step 4: No Changes to AI Analyzer!

```go
// AI analyzer code remains unchanged
aiAnalyzer := analysis.NewAIAnalyzer(llmClient, config)
```

---

## Summary

This API guide provides comprehensive coverage of:

- ✅ AI Analyzer creation and configuration
- ✅ LLM client interface and mock implementation
- ✅ Complete schema definitions
- ✅ Real-world integration examples
- ✅ Testing strategies and patterns
- ✅ Error handling best practices
- ✅ Performance optimization
- ✅ Migration path to real LLM

For more details, see:
- [Design Document](./design-ai-analyzer.md)
- [README](./README.md)
- [Completion Summary](./SUMMARY.md)

---

**Document Version:** 1.0  
**Last Updated:** 2025-12-16  
**Phase:** P03 - AI Analysis Integration Skeleton
