# P1: Diagnosis Flow Fix & RAG Enhancement

## 1. Overview
This phase addresses the disconnected AI analysis flow in the diagnosis manager and enhances the RAG pipeline with a Hybrid Searcher and a Reranker.

## 2. Architecture Changes

### Diagnosis Manager
*   **Before**: The `RunDiagnosis` method called `AnalyzeData` which only ran rule-based analyzers. The AI analysis part was mocked or incomplete.
*   **After**: `RunDiagnosis` now orchestrates `DiagnosisChain`. It first attempts AI-driven diagnosis using the chain. If successful, it merges AI findings with rule-based findings. If the AI chain fails, it falls back to rule-based analysis only.

### RAG Engine
*   **Before**: Used a simple retriever (often just semantic search).
*   **After**:
    *   **Hybrid Search**: Combines vector search (semantic) and BM25 (keyword) search.
    *   **Reranking**: Integrated a `Reranker` component. A `SimpleReranker` (TF-IDF based) is provided as a default implementation to re-score and re-order retrieval results before passing them to the LLM.

### Diagnosis Chain
*   **Flow**:
    1.  **Retrieve**: Fetches top-k (10) candidate documents related to the diagnosis query.
    2.  **Few-Shot**: Retrieves similar past diagnosis examples (if configured).
    3.  **Prompt**: Constructs a context-rich prompt.
    4.  **LLM**: Generates a structured JSON diagnosis.
    5.  **Parse**: Validates and parses the JSON into a Go struct.

## 3. Configuration
New configuration added to `config.yaml`:

```yaml
knowledge:
  retrieval:
    reranker:
      type: "simple" # Options: simple, api
      top_k: 5
```

## 4. Testing
*   **Unit Tests**: Added `internal/core/diagnosis/manager_test.go` to verify the integration of `DiagnosisChain` within the `Manager`.
*   **Mocks**: Utilized mocks for PluginManager, MiddlewarePlugin, Retriever, and LLMClient to ensure isolated testing of the core logic.
