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

package search

import (
	"context"
)

// Document represents a single chunk of retrieved information that is considered
// relevant to a query. It is ready to be injected into an LLM prompt to provide context.
type Document struct {
	// Content is the text of the document chunk.
	Content string `json:"content"`
	// Metadata contains additional information about the document, such as its source URL.
	Metadata map[string]interface{} `json:"metadata"`
	// Score indicates the relevance of the document to the query, as determined by the search mechanism.
	Score float32 `json:"score"`
}

// RetrieveOptions holds options for a retrieval operation.
type RetrieveOptions struct {
	TopK int
}

// Retriever defines the interface for components that retrieve relevant documents
// from a knowledge base in response to a user query. This is a core part of the
// Retrieval-Augmented Generation (RAG) pattern.
type Retriever interface {
	// Retrieve finds the top K most relevant documents for a given query using a single method (e.g., semantic).
	Retrieve(ctx context.Context, query string, topK int) ([]Document, error)
	// HybridRetrieve finds the top K most relevant documents using a hybrid approach (e.g., semantic + keyword).
	HybridRetrieve(ctx context.Context, query string, opts *RetrieveOptions) ([]Document, error)
}
