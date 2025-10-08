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

package rag

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// Embedder defines the interface for components that convert text into numerical
// vector embeddings. This abstraction allows the application to use different
// embedding models or services.
type Embedder interface {
	// EmbedQuery converts a single query string into its vector representation.
	// This is typically used to create an embedding for a user's search query.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
	// EmbedDocuments converts a batch of documents (text chunks) into their
	// vector representations. This is used for indexing knowledge base content.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}

// llmClientEmbedder is an implementation of Embedder that uses an underlying LLMClient
// (e.g., for OpenAI or Gemini) to perform the embedding. This is a flexible design
// that reuses the existing client connections and authentication.
type llmClientEmbedder struct {
	log       logger.Logger
	llmClient interfaces.LLMClient
	modelName string // The specific embedding model to use, e.g., "text-embedding-ada-002"

	// A simple in-memory cache to avoid re-embedding the same text, improving performance and reducing cost.
	cache     map[string][]float32
	cacheMu   sync.RWMutex
}

// NewEmbedder creates a new embedder that uses the provided LLM client to generate
// vector embeddings. It also includes a simple in-memory cache to reduce redundant
// API calls for identical texts.
//
// Parameters:
//   client (interfaces.LLMClient): The LLM client to use for generating embeddings.
//   modelName (string): The specific embedding model to use (e.g., "text-embedding-ada-002").
//
// Returns:
//   Embedder: A new instance of an embedder.
//   error: An error if the provided client is nil.
func NewEmbedder(client interfaces.LLMClient, modelName string) (Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("LLMClient cannot be nil")
	}
	return &llmClientEmbedder{
		log:       logger.NewLogger("embedder"),
		llmClient: client,
		modelName: modelName,
		cache:     make(map[string][]float32),
	}, nil
}

// EmbedQuery converts a single query string into its vector embedding. It includes
// a caching layer to avoid re-computing embeddings for identical queries, which
// improves performance and reduces costs.
//
// Parameters:
//   ctx (context.Context): The context for the embedding operation.
//   text (string): The query text to embed.
//
// Returns:
//   []float32: The generated vector embedding.
//   error: An error if the embedding generation fails.
func (e *llmClientEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	// Check cache first for faster responses.
	e.cacheMu.RLock()
	embedding, found := e.cache[text]
	e.cacheMu.RUnlock()
	if found {
		e.log.Debug("Embedding cache hit for query.")
		return embedding, nil
	}

	e.log.Debugf("Embedding query: %.50s...", text)
	// We can use the batch endpoint for a single query as well.
	embeddings, err := e.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("embedding service returned no vectors for query")
	}

	// Store the new embedding in the cache.
	e.cacheMu.Lock()
	e.cache[text] = embeddings[0]
	e.cacheMu.Unlock()

	return embeddings[0], nil
}

// EmbedDocuments converts a batch of text documents into their corresponding
// vector embeddings by calling the underlying LLM client.
//
// Parameters:
//   ctx (context.Context): The context for the embedding operation.
//   texts ([]string): A slice of document texts to embed.
//
// Returns:
//   [][]float32: A slice of vector embeddings, one for each input document.
//   error: An error if the embedding generation fails.
func (e *llmClientEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// TODO: Implement batch-aware caching to only embed texts not already in the cache.
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	e.log.Infof("Embedding %d documents.", len(texts))
	req := &interfaces.EmbeddingRequest{
		Input: texts,
		Model: e.modelName,
	}

	resp, err := e.llmClient.GenerateEmbedding(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM client failed to generate embeddings: %w", err)
	}

	return resp.Embeddings, nil
}

//Personal.AI order the ending
