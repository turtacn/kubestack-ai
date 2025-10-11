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

package store

import (
	"fmt"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
)

// NewVectorStoreFromConfig is a factory function that creates a VectorStore instance
// based on the provided configuration. It acts as a selector to switch between
// different vector store implementations like in-memory, ChromaDB, etc.
//
// Parameters:
//   - cfg: The vector store configuration containing the provider and its settings.
//
// Returns:
//   - VectorStore: An initialized vector store implementation.
//   - error: An error if the provider is unknown or if initialization fails.
func NewVectorStoreFromConfig(cfg *config.VectorStoreConfig) (VectorStore, error) {
	switch cfg.Provider {
	case "in-memory":
		return NewInMemoryVectorStore()
	case "chroma":
		return NewChromaVectorStore(cfg.Chroma.URL, cfg.Chroma.CollectionName)
	default:
		return nil, fmt.Errorf("unknown vector store provider: %s", cfg.Provider)
	}
}

// NewDocumentStoreFromConfig is a factory function that creates a DocumentStore instance
// based on the provided configuration. It acts as a selector to switch between
// different document store implementations like in-memory and Elasticsearch.
//
// Parameters:
//   - cfg: The document store configuration containing the provider and its settings.
//
// Returns:
//   - DocumentStore: An initialized document store implementation.
//   - error: An error if the provider is unknown or if initialization fails.
func NewDocumentStoreFromConfig(cfg *config.DocumentStoreConfig) (DocumentStore, error) {
	switch cfg.Provider {
	case "in-memory":
		return NewInMemoryDocumentStore()
	case "elasticsearch":
		return NewElasticsearchDocumentStore(cfg.Elasticsearch.Addresses, cfg.Elasticsearch.IndexName)
	default:
		return nil, fmt.Errorf("unknown document store provider: %s", cfg.Provider)
	}
}