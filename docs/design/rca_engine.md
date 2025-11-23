# Root Cause Analysis (RCA) Engine Design

## 1. Overview

The RCA Engine is responsible for inferring the underlying cause of detected anomalies. It bridges the gap between symptom detection (Anomaly Detection) and remediation (Execution).

## 2. Architecture

The RCA Engine is composed of:
1.  **Rules Engine**: A deterministic system that matches symptoms to known root causes based on predefined logic.
2.  **Knowledge Graph (Mock/Prototype)**: A semantic search component that finds similar historical cases to suggest solutions for novel issues.

### 2.1 Rules Engine

The Rules Engine evaluates a set of rules against the list of detected anomalies.

**Rule Structure:**
*   **Conditions**: A set of anomalies (Type + Severity) that must be present.
*   **Root Cause**: The conclusion if conditions are met.
*   **Actions**: Recommended steps to fix the issue.
*   **Priority**: Determines which rule wins if multiple match.

### 2.2 Knowledge Graph

The Knowledge Graph component allows for retrieval of historical incidents. currently implemented as a simple keyword/similarity matcher, it is designed to evolve into a vector-database-backed RAG (Retrieval-Augmented Generation) system.

## 3. Integration with Diagnosis Flow

1.  **Trigger**: RCA is triggered automatically if the Anomaly Detector reports any issues.
2.  **Input**: The list of `Anomaly` objects found.
3.  **Process**:
    *   Rules Engine scans for matches.
    *   (Optional) Knowledge Graph queries for similar cases.
4.  **Output**: An `RCAResult` containing the primary root cause and recommendations.
5.  **Reporting**: The RCA result is converted into a high-confidence `Issue` in the final diagnosis report.

## 4. Configuration

Rules are defined in `configs/rca/rules.yaml`.

```yaml
rules:
  - name: "Redis High Memory"
    conditions:
      - anomaly_type: "HighMemory"
    root_cause: "Redis Memory Fragmentation"
```
