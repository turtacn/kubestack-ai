# RAG Retrieval Pipeline Design

## 1. Overview

The RAG (Retrieval-Augmented Generation) system in KubeStack-AI implements a multi-stage retrieval pipeline to improve the relevance and accuracy of the context provided to the LLM.

## 2. Pipeline Stages

The pipeline consists of the following stages:

1.  **Query Processing**:
    *   **Rewriting**: Corrects typos and normalizes terms (e.g., "reddis" -> "redis").
    *   **Expansion** (Optional): Generates hypothetical documents (HyDE) or synonymous queries to broaden the search scope.

2.  **Recall (Retrieval)**:
    *   **Vector Search**: Retrieves documents based on semantic similarity.
    *   **Keyword Search**: Retrieves documents based on BM25 keyword matching (if enabled).

3.  **Fusion (Coarse Ranking)**:
    *   **RRF (Reciprocal Rank Fusion)**: Merges results from different sources (Vector, Keyword, Expanded Queries) by rank.
    *   **Weighted Fusion**: Alternative strategy using weighted scores.

4.  **Fine Ranking (Reranking)**:
    *   **Threshold Reranker**: Filters out documents below a certain relevance score.
    *   **LLM Reranker**: Uses an LLM to assess the specific relevance of each candidate document to the query.

5.  **Generation**:
    *   The top-ranked documents are used as context in the prompt sent to the LLM.

## 3. Architecture

### Components

*   `RAGEngine`: Orchestrator.
*   `MultiStageRetriever`: Handles Recall, Fusion, and Reranking phases.
*   `QueryRewriter` / `QueryExpander`: Pre-processing of the user query.
*   `Indexer`: Manages document ingestion and updates.

### Data Flow

```mermaid
graph LR
    UserQuery --> QueryRewriter
    QueryRewriter --> QueryExpander
    QueryExpander --Queries--> MultiStageRetriever
    MultiStageRetriever --Results--> Fusion
    Fusion --Candidates--> Reranker
    Reranker --TopDocs--> LLM
    LLM --> Answer
```

## 4. Configuration

The pipeline is configurable via `rag.yaml` (or `knowledge.yaml` in older configs), allowing control over TopK at each stage, fusion strategies, and reranking thresholds.
