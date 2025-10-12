// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package search provides high-level components for searching the knowledge base.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/rag"
)

// Searcher defines a high-level interface for performing searches over the knowledge
// base. It abstracts away the details of the underlying search strategy (e.g.,
// semantic, keyword, hybrid), allowing different search methods to be used
// interchangeably by the application's core logic.
type Searcher interface {
	// Search takes a user query and returns a ranked list of relevant documents
	// from the knowledge base.
	//
	// Parameters:
	//   ctx (context.Context): The context for the search operation.
	//   query (string): The user's search query.
	//
	// Returns:
	//   []rag.Document: A slice of documents, ordered by relevance.
	//   error: An error if the search operation fails.
	Search(ctx context.Context, query string) ([]rag.Document, error)
}

// semanticSearcher implements the Searcher interface. It orchestrates the process
// of retrieving relevant documents using semantic understanding, powered by a RAG retriever.
type semanticSearcher struct {
	log           logger.Logger
	retriever     rag.Retriever
	llmClient     interfaces.LLMClient
	promptBuilder *prompt.Builder
}

// NewSemanticSearcher creates a new searcher that performs semantic searches.
// It relies on an underlying RAG (Retrieval-Augmented Generation) retriever to
// find documents based on vector similarity.
//
// Parameters:
//   retriever (rag.Retriever): The retriever component that performs the vector search.
//
// Returns:
//   Searcher: A new instance of a semantic searcher.
//   error: An error if the provided retriever is nil.
func NewSemanticSearcher(retriever rag.Retriever, llmClient interfaces.LLMClient, promptBuilder *prompt.Builder) (Searcher, error) {
	if retriever == nil {
		return nil, fmt.Errorf("retriever cannot be nil")
	}
	if llmClient == nil {
		return nil, fmt.Errorf("llmClient cannot be nil")
	}
	if promptBuilder == nil {
		return nil, fmt.Errorf("promptBuilder cannot be nil")
	}
	return &semanticSearcher{
		log:           logger.NewLogger("semantic-searcher"),
		retriever:     retriever,
		llmClient:     llmClient,
		promptBuilder: promptBuilder,
	}, nil
}

// Search implements the Searcher interface for semantic search. It takes a user
// query, passes it to the underlying RAG retriever, and returns the results.
// This implementation provides a framework where more advanced RAG techniques,
// such as query transformation and result re-ranking, can be added in the future.
//
// Parameters:
//   ctx (context.Context): The context for the search operation.
//   query (string): The user's search query.
//
// Returns:
//   []rag.Document: A slice of documents retrieved based on semantic similarity.
//   error: An error if the retrieval process fails.
func (s *semanticSearcher) Search(ctx context.Context, query string) ([]rag.Document, error) {
	s.log.Infof("Performing semantic search for query: %.50s...", query)

	// **Pre-retrieval Step (Placeholder): Query Transformation**
	expandedQuery, err := s.expandQuery(ctx, query)
	if err != nil {
		s.log.Warnf("Failed to expand query, falling back to original query: %v", err)
		expandedQuery = query
	}

	// **Retrieval Step**
	// The number of documents to retrieve (topK) could be configurable.
	const topK = 5
	retrievedDocs, err := s.retriever.Retrieve(ctx, expandedQuery, topK)
	if err != nil {
		return nil, err
	}

	// **Post-retrieval Step (Placeholder): Re-ranking & Filtering**
	rerankedDocs, err := s.rerank(ctx, query, retrievedDocs)
	if err != nil {
		s.log.Warnf("Failed to re-rank documents, falling back to original ranking: %v", err)
		rerankedDocs = retrievedDocs
	}

	return rerankedDocs, nil
}

func (s *semanticSearcher) rerank(ctx context.Context, query string, docs []rag.Document) ([]rag.Document, error) {
	s.log.Debugf("Re-ranking %d documents for query: %s", len(docs), query)
	docsJSON, err := json.Marshal(docs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal documents for re-ranking: %w", err)
	}

	data := map[string]string{
		"Query":     query,
		"Documents": string(docsJSON),
	}

	messages, err := s.promptBuilder.Build("rerank", data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build re-rank prompt: %w", err)
	}

	req := &interfaces.LLMRequest{
		Messages:    messages,
		Temperature: 0.1,
	}

	resp, err := s.llmClient.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call for re-ranking failed: %w", err)
	}

	var rerankedDocs []rag.Document
	if err := json.Unmarshal([]byte(resp.Message.Content), &rerankedDocs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal re-ranked documents: %w", err)
	}

	s.log.Debugf("Successfully re-ranked %d documents.", len(rerankedDocs))
	return rerankedDocs, nil
}

func (s *semanticSearcher) expandQuery(ctx context.Context, query string) (string, error) {
	s.log.Debugf("Expanding query: %s", query)
	data := map[string]string{"Query": query}
	messages, err := s.promptBuilder.Build("query-expansion", data, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build query expansion prompt: %w", err)
	}

	req := &interfaces.LLMRequest{
		Messages:    messages,
		Temperature: 0.4,
	}

	resp, err := s.llmClient.SendMessage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call for query expansion failed: %w", err)
	}

	// The expanded queries are returned as a single string with newlines.
	// We'll replace newlines with spaces to create a single, expanded query string.
	expandedQuery := strings.ReplaceAll(resp.Message.Content, "\n", " ")
	s.log.Debugf("Expanded query: %s", expandedQuery)
	return expandedQuery, nil
}