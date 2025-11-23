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

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
)

// KnowledgeManager defines the interface for the knowledge base, providing a unified
// entry point for searching and managing knowledge.
type KnowledgeManager interface {
	Search(ctx context.Context, query string) ([]search.Document, error)
}