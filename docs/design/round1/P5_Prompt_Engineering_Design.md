# P5: LLM Integration & Prompt Engineering Design

## 1. Overview
This phase enhances the AI capabilities of KubeStack-AI by introducing a robust prompt engineering framework. This includes structured output via JSON schemas, few-shot learning for better context, and prompt chaining for complex diagnostic workflows.

## 2. Architecture

### 2.1 Prompt Template Engine
- **Interface:** `PromptTemplate`
- **Implementation:** `GoTemplate` (wraps `text/template`)
- **Features:**
    - Variable injection (`{{.Variable}}`)
    - Conditional logic (`{{if .Condition}}`)
    - Loops (`{{range .Items}}`)
    - Custom functions (`truncate`, `join`)

### 2.2 Structured Output
- **Parser:** `StructuredOutputParser`
- **Schema:** `DiagnosisResult` (JSON)
- **Validation:** Uses `go-playground/validator` to ensure fields like `root_cause`, `severity`, and `confidence` are present and valid.
- **LLM Mode:** Leverages OpenAI's `response_format: {"type": "json_object"}`.

### 2.3 Few-Shot Learning
- **Manager:** `FewShotManager`
- **Mechanism:** Stores examples (Input -> Analysis -> Output) and retrieves the most relevant ones based on cosine similarity of embeddings (or category matching as fallback).
- **Benefit:** Drastically improves diagnosis accuracy for known patterns.

### 2.4 Diagnosis Chain
- **Executor:** `ChainExecutor`
- **Flow:**
    1. **Retrieval:** Fetch relevant docs from Knowledge Base.
    2. **Few-Shot:** Fetch similar past cases.
    3. **Prompting:** Construct prompt with Context + Examples.
    4. **Inference:** Call LLM.
    5. **Parsing:** Validate JSON output.

## 3. Data Flow
`User Query` -> `Retrieval` -> `Prompt Construction (Template + Context + Examples)` -> `LLM` -> `JSON Parsing` -> `DiagnosisResult`

## 4. Configuration
- Prompts are configured in `configs/prompts/prompt_config.yaml`.
- Examples are stored in `docs/examples/few_shot_examples.yaml`.
