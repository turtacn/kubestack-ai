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

package interfaces

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/rag"
)

// KnowledgeManager defines the contract for the component that provides a unified
// interface to the application's knowledge base. It abstracts away the complexities
// of different storage and search strategies (e.g., vector vs. keyword search).
type KnowledgeManager interface {
	// Search queries the knowledge base and returns the most relevant documents.
	Search(ctx context.Context, query string) ([]rag.Document, error)

	// TODO: Add methods for knowledge ingestion and management.
	// Ingest(ctx context.Context, source models.DataSource) error
	// Delete(ctx context.Context, documentID string) error
}