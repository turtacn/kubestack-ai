// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package knowledge

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
)

// Manager defines the interface for the knowledge base, providing a unified
// entry point for searching and managing knowledge.
type Manager interface {
	Search(ctx context.Context, query string) ([]search.Document, error)
}

// knowledgeManager is the concrete implementation of the Manager interface.
type knowledgeManager struct {
	log      logger.Logger
	searcher search.Searcher
}

// NewManager creates a new instance of the knowledge manager.
func NewManager(searcher search.Searcher) (Manager, error) {
	return &knowledgeManager{
		log:      logger.NewLogger("knowledge-manager"),
		searcher: searcher,
	}, nil
}

// Search performs a search against the knowledge base.
func (m *knowledgeManager) Search(ctx context.Context, query string) ([]search.Document, error) {
	m.log.Infof("Searching knowledge base for query: %.50s...", query)
	return m.searcher.Search(ctx, query)
}
