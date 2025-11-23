# Prompt Template Guide

## Overview
We use Go's `text/template` engine for dynamic prompt generation. This allows us to inject context, metrics, and logs into the prompt before sending it to the LLM.

## Syntax
- **Variables:** `{{.VariableName}}`
- **Conditions:** `{{if .Condition}} ... {{else}} ... {{end}}`
- **Loops:** `{{range .List}} ... {{end}}`

## Available Functions
- `truncate <string> <length>`: Truncates a string to the specified length.
- `join <list> <sep>`: Joins a list of strings with a separator.

## Standard Template Structure
1. **Role Definition:** "You are an expert..."
2. **Task:** "Analyze the following..."
3. **Context:**
    - User Query
    - Metrics
    - Logs
    - Knowledge Base Snippets
4. **Examples (Few-Shot):** Dynamically injected.
5. **Output Requirements:** JSON Schema definition.

## Best Practices
- Keep instructions clear and imperative.
- Use the JSON mode enforcement in the system prompt.
- Limit the length of logs/metrics using `truncate` to avoid hitting token limits.
