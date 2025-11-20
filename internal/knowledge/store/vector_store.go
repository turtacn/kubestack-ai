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

// Package store provides interfaces and implementations for different knowledge storage backends.
package store

import (
	"context"
)

// StoreDocument represents a single, embeddable unit of text (a "chunk") and its
// corresponding vector representation. This is the core data model for storage
// and retrieval in a vector database.
type StoreDocument struct {
	// ID is the unique identifier for this document chunk.
	ID string
	// Content is the text content of the chunk.
	Content string
	// Vector is the numerical representation of the content's semantics.
	Vector []float32
	// Metadata contains additional information about the chunk, such as its source document.
	Metadata map[string]interface{}
	// Score is populated during a search operation and indicates the relevance of the
	// document to the query. It is not a stored field.
	Score float32
}

// VectorStore defines the interface for a vector database. It abstracts the
// storage and retrieval of vector embeddings, allowing for different backends
// (e.g., in-memory, Chroma, Qdrant) to be used interchangeably.
type VectorStore interface {
	// AddDocuments adds a batch of documents (chunks and their vectors) to the store.
	AddDocuments(ctx context.Context, docs []StoreDocument) error
	// SimilaritySearch finds the top K documents in the store that are most
	// semantically similar to a given query vector.
	SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error)
}
