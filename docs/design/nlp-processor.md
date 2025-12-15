# NLP Processor Design

## Overview

The NLP Processor is a core component of the KubeStack-AI agent, responsible for understanding user input, extracting relevant entities, and managing conversation context. It acts as the bridge between natural language user requests and the system's execution capabilities.

## Architecture

The NLP Processor consists of the following sub-modules:

1.  **Tokenizer**: Splits text into tokens. Supports simple whitespace/punctuation splitting and can be extended for advanced tokenization (e.g., Jieba for Chinese).
2.  **Entity Extractor**: Identifies and extracts entities such as middleware types, metrics, time ranges, and instance IDs using regex patterns and dictionaries.
3.  **Intent Recognizer**: Determines the user's intent (e.g., Diagnose, Query, Fix) using a rule-based approach (regex/keywords) with optional LLM fallback.
4.  **Context Manager**: Manages multi-turn conversation history and maintains active entities across turns.

## Data Flow

1.  **Input**: User text + Session ID.
2.  **Preprocessing**: Text normalization.
3.  **Tokenization**: `Tokenizer.Tokenize(text)`.
4.  **Entity Extraction**: `Extractor.Extract(text, tokens)`.
5.  **Context Loading**: Load conversation history for the session.
6.  **Intent Recognition**: `Recognizer.Recognize(text, tokens, entities, history)`.
7.  **Context Update**: Save current turn (intent, entities, text) and update active entities.
8.  **Output**: Intent, Entities, Context.

## Components

### Intent Recognizer

-   **Types**: Diagnose, Query, Fix, Alert, Config, Explain, Help.
-   **Rule-Based**: Uses regex patterns for high-precision matching of common operational commands.
-   **LLM-Based (Optional)**: Uses an LLM to handle ambiguous or complex natural language queries that rules miss.

### Entity Extractor

-   **Types**:
    -   `middleware_type`: redis, mysql, etc.
    -   `metric_name`: memory, cpu, qps, etc.
    -   `time_range`: last 1h, yesterday, etc.
    -   `instance_id`: redis-cluster-01, ip:port.
    -   `threshold`: > 80%.
-   **Method**: Pattern matching (Regex) + Dictionary lookups.

### Context Manager

-   **Session Storage**: In-memory (default), extensible to Redis.
-   **Features**:
    -   Maintains recent N turns.
    -   Tracks "Active Entities" (e.g., if user mentioned "Redis" in turn 1, "it" in turn 2 refers to "Redis").

## Configuration

Configuration is managed via `configs/nlp.yaml`.

```yaml
nlp:
  tokenizer:
    type: simple
  intent:
    recognizer_type: hybrid
  context:
    max_turns: 10
    session_ttl: 30m
```
