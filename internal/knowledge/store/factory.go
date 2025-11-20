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

// NewVectorStoreFromConfig creates a new vector store based on the provided configuration.
func NewVectorStoreFromConfig(cfg *config.KnowledgeConfig) (VectorStore, error) {
	switch cfg.Retrieval.Semantic.Provider {
	case "in-memory":
		return NewInMemoryVectorStore()
	case "chroma":
		return NewChromaVectorStore(cfg.Retrieval.Semantic.Model, "default")
	default:
		return nil, fmt.Errorf("unsupported vector store provider: %s", cfg.Retrieval.Semantic.Provider)
	}
}

// NewDocumentStoreFromConfig creates a new document store based on the provided configuration.
func NewDocumentStoreFromConfig(cfg *config.KnowledgeConfig) (DocumentStore, error) {
	switch cfg.Retrieval.Keyword.Engine {
	case "in-memory":
		return newInMemoryDocumentStore()
	default:
		return nil, fmt.Errorf("unsupported document store provider: %s", cfg.Retrieval.Keyword.Engine)
	}
}
