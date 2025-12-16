# Phase 03: AI Analysis Integration Skeleton & Jules-Friendly Boundaries

**Phase ID:** P03  
**Branch:** `feat/round5-phase03-ai-analysis-skeleton`  
**Status:** ✅ Completed  
**Dependencies:** P02

---

## Overview

Phase 03 establishes the foundational architecture for AI/LLM integration into the KubeStack-AI diagnosis engine. Rather than immediately integrating a real LLM, this phase focuses on creating clean, well-defined boundaries between the orchestrator and AI analysis components.

### Key Objectives

1. **O1**: Create `AIAnalyzer` implementing the `Analyzer` interface
2. **O2**: Define stable Prompt templates and JSON Schema contracts
3. **O3**: Provide `MockLLMClient` for repeatable testing
4. **O4**: Ensure Jules-friendly implementation (single file/single function scope)

---

## Architecture Design

### Component Boundaries

```
┌──────────────────────────────────────────────────────────┐
│                   Diagnosis Orchestrator                  │
└─────────────┬──────────────────────────────┬──────────────┘
              │                              │
              ├─────────┐                   │
              │         │                   │
       ┌──────▼─────┐   │            ┌──────▼──────┐
       │   Rule     │   │            │      AI     │
       │  Analyzer  │   │            │   Analyzer  │
       └────────────┘   │            └──────┬──────┘
                        │                   │
                        │            ┌──────▼──────┐
                        │            │ LLM Client  │
                        │            │  Interface  │
                        │            └──────┬──────┘
                        │                   │
                        │         ┌─────────┴──────────┐
                        │         │                    │
                        │   ┌─────▼─────┐      ┌──────▼──────┐
                        │   │   Mock    │      │   OpenAI    │
                        │   │  Client   │      │   Client    │
                        │   └───────────┘      │  (Future)   │
                        │                      └─────────────┘
                        │
```

### AI Analyzer Role

**What AI Does:**
- Analyzes structured plugin data (metrics, logs, configs)
- Generates structured issues with evidence and recommendations
- Provides reasoning and confidence scores

**What AI Does NOT Do:**
- Control diagnosis flow
- Make orchestration decisions
- Execute fixes directly

### Replaceability Strategy

The design supports multiple AI implementations:
- **Mock** (Phase 03): Returns fixed JSON for testing
- **Rule-based** (Existing): Simple pattern matching
- **LLM** (Future): Real AI analysis (OpenAI, Gemini, etc.)
- **RAG/KB** (Future): Knowledge-base enhanced analysis

---

## Implementation Details

### 1. LLM Client Contract

**File:** `internal/core/llm/client.go`

The LLM client interface wraps the existing `internal/llm/interfaces.LLMClient`:

```go
package llm

import (
    llmInterface "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// Client is an alias for the existing LLM client interface
type Client = llmInterface.LLMClient
```

This provides a clean re-export for the analysis layer while maintaining compatibility with existing LLM infrastructure.

### 2. Mock LLM Client

**File:** `internal/core/llm/mock_client.go`

Provides deterministic responses for testing:

```go
type MockClient struct {
    Response    string
    Error       error
    CallCount   int
    LastRequest *llmInterface.LLMRequest
}

func (m *MockClient) Complete(ctx context.Context, req *llmInterface.LLMRequest) (*llmInterface.LLMResponse, error)
```

**Features:**
- Fixed JSON responses
- Request capture for verification
- Call count tracking
- Error simulation

### 3. AI Input/Output Schema

**File:** `internal/core/analysis/schema.go`

Defines the contract between orchestrator and AI:

```go
// AIInput - What goes to the LLM
type AIInput struct {
    PluginData  PluginDataSummary  `json:"plugin_data"`
    Context     ContextInfo        `json:"context"`
}

// AIOutput - What comes back from the LLM
type AIOutput struct {
    Summary     string             `json:"summary"`
    Reasoning   string             `json:"reasoning"`
    Issues      []AIIssue          `json:"issues"`
}

// AIIssue - Individual issue identified by AI
type AIIssue struct {
    ID              string              `json:"id"`
    Title           string              `json:"title"`
    Severity        string              `json:"severity"`
    Description     string              `json:"description"`
    Evidence        string              `json:"evidence"`
    Recommendations []AIRecommendation  `json:"recommendations"`
}
```

**Design Principles:**
- JSON-serializable structures
- Clear field documentation
- Conversion utilities to/from `models.Issue`
- Severity mapping with enum support

### 4. Prompt Templates

**File:** `internal/core/analysis/prompt_templates.go`

Defines stable prompts for AI analysis:

**System Prompt:**
```
You are a middleware diagnosis expert specializing in {middleware_type}.
Analyze the provided data and identify issues with evidence.

CRITICAL: Respond ONLY with valid JSON. No explanations, no markdown.

Required JSON structure:
{
  "summary": "Brief overview",
  "reasoning": "Your analysis process",
  "issues": [...]
}
```

**User Prompt:**
```
Diagnose the following {middleware_type} instance:
Instance: {instance}
Namespace: {namespace}

Metrics:
{metrics_json}

Logs:
{logs_summary}

Configuration:
{config_json}

Identify issues with evidence and recommendations.
```

**Features:**
- JSON-only output constraint
- Schema examples embedded
- Context injection (middleware type, instance, namespace)
- Structured data presentation

### 5. AI Analyzer Implementation

**File:** `internal/core/analysis/ai_analyzer.go`

Implements the `Analyzer` interface:

```go
type AIAnalyzer struct {
    llmClient  llm.Client
    config     AIAnalyzerConfig
}

func (a *AIAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error) {
    // 1. Build AI input from collected data
    input := buildAIInput(data, a.config)
    
    // 2. Render prompt with input data
    prompt := renderPrompt(input)
    
    // 3. Call LLM
    response, err := a.llmClient.Complete(ctx, prompt)
    
    // 4. Parse JSON response
    output := parseAIOutput(response)
    
    // 5. Convert to AnalysisResult
    return convertToAnalysisResult(output)
}
```

**Key Methods:**
- `buildAIInput()`: Converts `CollectedData` to `AIInput`
- `renderPrompt()`: Templates prompt with context
- `parseAIOutput()`: Parses and validates JSON
- `cleanJSONResponse()`: Strips markdown artifacts
- `convertToIssues()`: Maps `AIIssue` → `models.Issue`

---

## Testing Strategy

### Unit Tests

**File:** `internal/core/analysis/ai_analyzer_test.go`

**Test Cases:**
1. `TestAIAnalyzer_ParseValidJSON`: Valid LLM response → correct issues
2. `TestAIAnalyzer_InvalidJSON`: Malformed JSON → error handling
3. `TestAIAnalyzer_LLMError`: LLM failure → graceful degradation
4. `TestAIAnalyzer_EmptyIssues`: No issues found → valid empty result
5. `TestAIAnalyzer_MultipleIssues`: Multiple issues → all parsed correctly
6. `TestAIAnalyzer_CleanJSONResponse`: Markdown-wrapped JSON → cleaned
7. `TestAIAnalyzer_ValidateSeverity`: Severity mapping → enum values
8. `TestAIAnalyzer_SetMiddlewareContext`: Context injection → correct prompt
9. `TestBuildAIInput`: Data conversion → structured input

### Integration Tests

**File:** `test/integration/ai_analyzer_integration_test.go`

**Test Scenarios:**
1. `TestDiagnosis_WithAIAnalyzer`: Full diagnosis flow with AI analyzer
2. `TestDiagnosis_WithMultipleAnalyzers`: AI + Rule analyzers together

**Verification Points:**
- AI-sourced issues appear in report
- Issue structure matches schema
- Severity levels correctly mapped
- Mock client called with correct data
- Report metadata includes LLM info

---

## GAP Analysis Results

### GAP-G: Missing AI Analyzer Layer ✅ RESOLVED

**Before Phase 03:**
- No clear entry point for AI analysis
- Would require modifying orchestrator for AI integration

**After Phase 03:**
- `AIAnalyzer` implements standard `Analyzer` interface
- Orchestrator unchanged - plug-and-play integration
- Clear separation of concerns

### GAP-H: Unstable Prompt/Schema Contracts ✅ RESOLVED

**Before Phase 03:**
- No defined prompt structure
- No JSON schema for AI I/O
- Ad-hoc LLM interactions

**After Phase 03:**
- Prompt templates in `prompt_templates.go`
- Schema definitions in `schema.go`
- JSON-only output enforced
- Schema versioning possible

### GAP-I: Jules Integration Complexity ✅ RESOLVED

**Before Phase 03:**
- Unclear how Jules would interact
- Global business logic understanding required

**After Phase 03:**
- Single-file implementations (`ai_analyzer.go`)
- Clear function boundaries
- Mock client for testing without LLM
- Jules can work on isolated functions

---

## Future Integration Path

### Phase 03 → Real LLM Integration

**Step 1: Implement Real LLM Client**
```go
// File: internal/core/llm/openai_client.go
type OpenAIClient struct {
    client *openai.Client
    model  string
}

func (c *OpenAIClient) Complete(ctx, req) (*Response, error) {
    // Call OpenAI API
}
```

**Step 2: Swap Mock with Real Client**
```go
// In orchestrator setup
llmClient := llm.NewOpenAIClient(apiKey, model)
aiAnalyzer := analysis.NewAIAnalyzer(llmClient, config)
```

**No Changes Required:**
- ✅ Orchestrator code unchanged
- ✅ Prompt templates unchanged
- ✅ Schema contracts unchanged
- ✅ Test infrastructure reusable

### Phase 03 → Jules Enhancement

**Jules Can Now:**
1. Enhance prompt templates (single function)
2. Add new schema fields (single struct)
3. Improve JSON parsing (single function)
4. Add new mock responses (single method)

**Jules Does NOT Need To:**
- Understand full diagnosis flow
- Modify orchestrator logic
- Change plugin system
- Refactor analyzer interface

---

## Files Created/Modified

### New Files
- ✅ `internal/core/llm/client.go` - LLM client interface wrapper
- ✅ `internal/core/llm/mock_client.go` - Mock implementation
- ✅ `internal/core/analysis/schema.go` - AI I/O schemas
- ✅ `internal/core/analysis/prompt_templates.go` - Prompt definitions
- ✅ `internal/core/analysis/ai_analyzer.go` - AI analyzer implementation
- ✅ `internal/core/analysis/ai_analyzer_test.go` - Unit tests
- ✅ `test/integration/ai_analyzer_integration_test.go` - Integration tests

### Modified Files
- ✅ `docs/architecture.md` - Updated with AI analyzer architecture
- ✅ `docs/round5/phase03/design-ai-analyzer.md` - This document

---

## Acceptance Criteria Status

- ✅ **AC-1**: AIAnalyzer implements Analyzer interface without orchestrator changes
- ✅ **AC-2**: Prompt + Schema are model-agnostic and stable
- ✅ **AC-3**: Integration tests are repeatable with MockLLMClient
- ✅ **AC-4**: Jules can implement single file/function without global context

---

## Key Takeaways

### What This Phase Achieved

1. **Clean Boundaries**: Clear separation between orchestration and AI analysis
2. **Stable Contracts**: Prompt and schema define unchanging AI interface
3. **Testability**: Mock client enables deterministic testing
4. **Extensibility**: Easy to swap mock → real LLM
5. **Jules-Friendly**: Single-file/function implementations

### What This Phase Did NOT Do

1. ❌ Integrate real LLM (by design)
2. ❌ Implement RAG/KB retrieval (future phase)
3. ❌ Add prompt optimization (future iteration)
4. ❌ Implement multi-turn conversations (future feature)

### Design Philosophy

> **"Structure before capability"**
> 
> Phase 03 prioritizes architectural soundness over immediate functionality.
> By establishing clean contracts and boundaries first, we enable rapid iteration
> on AI capabilities without destabilizing the core system.

---

## Next Steps

### Immediate (Phase 04+)
- Integrate real OpenAI/Gemini client
- Add RAG-based knowledge retrieval
- Implement prompt optimization
- Add confidence scoring

### Future Enhancements
- Multi-turn diagnostic conversations
- Few-shot learning examples
- Context-aware prompt adaptation
- A/B testing of prompt variants

---

**Phase 03 Status:** ✅ **COMPLETE**

All objectives met. System ready for real LLM integration.
