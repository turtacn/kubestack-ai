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

package manager

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/rag"
)

// manager is the concrete implementation of the interfaces.KnowledgeManager.
type manager struct {
	log      logger.Logger
	searcher search.Searcher
}

// NewManager creates a new instance of the knowledge manager.
func NewManager(searcher search.Searcher) interfaces.KnowledgeManager {
	return &manager{
		log:      logger.NewLogger("knowledge-manager"),
		searcher: searcher,
	}
}

// Search delegates the search operation to the underlying searcher component.
func (m *manager) Search(ctx context.Context, query string) ([]rag.Document, error) {
	m.log.Infof("Performing knowledge search for query: %.50s...", query)
	return m.searcher.Search(ctx, query)
}