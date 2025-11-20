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
	"os"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/stretchr/testify/assert"
)

func TestBM25Searcher(t *testing.T) {
	indexPath := "./test_bm25.index"
	defer os.RemoveAll(indexPath)

	searcher, err := NewBM25Searcher(indexPath)
	assert.NoError(t, err)

	docs := []*store.StoreDocument{
		{ID: "1", Content: "Redis cluster"},
		{ID: "2", Content: "Redis sentinel"},
		{ID: "3", Content: "MongoDB sharding"},
	}

	err = searcher.IndexDocuments(docs)
	assert.NoError(t, err)

	results, err := searcher.Search("Redis", 2)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, []string{results[0].Content, results[1].Content}, "Redis cluster")
	assert.Contains(t, []string{results[0].Content, results[1].Content}, "Redis sentinel")
}
