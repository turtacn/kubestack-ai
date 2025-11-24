# Knowledge Base Design

## Overview

The Knowledge Base (KB) module is a core component of KubeStack-AI, responsible for storing, managing, and retrieving diagnostic rules, failure patterns, and best practices. It empowers the diagnostic engine with expert knowledge to identify root causes and suggest remediations.

## Architecture

The Knowledge Base system consists of the following components:

1.  **Rule Definition**: Structured representation of diagnostic rules.
2.  **Knowledge Base**: In-memory storage with efficient indexing and querying capabilities.
3.  **Rule Engine**: Logic to evaluate rules against current diagnostic context (metrics, logs, etc.).
4.  **Loader**: Mechanism to load rules from external sources (YAML files) with support for hot reloading.
5.  **LLM Integration**: Enhances diagnostics by providing relevant knowledge context to Large Language Models.
6.  **API**: RESTful interface for managing rules.

### Data Model

#### Rule

A Rule is the fundamental unit of knowledge.

```go
type Rule struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    MiddlewareType string   `json:"middleware_type"` // e.g., redis, mysql, kafka
    Category       string   `json:"category"`        // e.g., performance, stability
    Severity       string   `json:"severity"`        // HIGH, CRITICAL, MEDIUM, LOW
    Condition      string   `json:"condition"`       // Logical expression, e.g., "memory_usage > 80"
    Recommendation string   `json:"recommendation"`  // Suggested action
    Priority       int      `json:"priority"`        // Execution priority
    Tags           []string `json:"tags"`
    Version        string   `json:"version"`
}
```

### Components

#### KnowledgeBase (`internal/knowledge/base.go`)
- Manages the lifecycle of rules.
- Provides thread-safe access.
- Maintains indexes (by MiddlewareType, Tags) for fast retrieval.

#### RuleEngine (`internal/knowledge/rule_engine.go`)
- **Matching**: Finds rules applicable to the current `DiagnosisContext`.
- **Evaluation**: Uses `ConditionEvaluator` to execute the `Condition` expression against collected metrics.
- **Execution**: Generates `Recommendation` objects for matched rules.

#### RuleLoader (`internal/knowledge/loader.go`)
- Loads rules from YAML/JSON files.
- Watches file system for changes and triggers hot reloads.

#### LLMIntegration (`internal/knowledge/llm_integration.go`)
- Constructs prompts using diagnostic context and matched knowledge.
- Calls LLM API to generate human-readable, context-aware advice.

## Integration Flow

1.  **Diagnosis Request**: User triggers a diagnosis.
2.  **Data Collection**: Plugins collect metrics and identify anomalies.
3.  **Rule Matching**:
    - `DiagnosisManager` creates a `DiagnosisContext`.
    - `RuleEngine` queries the `KnowledgeBase` for relevant rules.
    - Conditions are evaluated against metrics.
4.  **LLM Enhancement** (Optional):
    - Matched rules and context are sent to LLM.
    - LLM generates detailed analysis and additional recommendations.
5.  **Result Aggregation**: Rule-based recommendations and LLM insights are merged into the final `DiagnosisResult`.

## Configuration

Configuration is managed via `configs/knowledge/rules_config.yaml`.

```yaml
knowledge:
  rule_files:
    - "internal/knowledge/repository/redis_rules.yaml"
  refresh_interval: 60s
  enable_llm_enhancement: true
```
