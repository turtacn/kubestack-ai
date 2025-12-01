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
	"os"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestLoadRAGConfig(t *testing.T) {
	// Create a temporary config file
	configFile, err := os.CreateTemp("", "config_test_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(configFile.Name())

	// Use map to ensure keys match mapstructure tags explicitly
	cfgMap := map[string]interface{}{
		"knowledge": map[string]interface{}{
			"retrieval": map[string]interface{}{
				"mode": "hybrid",
				"semantic": map[string]interface{}{
					"enabled":         true,
					"provider":        "chroma",
					"model":           "text-embedding-3-small",
					"top_k":           10,
					"score_threshold": 0.6,
				},
				"keyword": map[string]interface{}{
					"enabled":  true,
					"engine":   "bleve",
					"analyzer": "standard",
					"top_k":    10,
				},
				"fusion": map[string]interface{}{
					"strategy": "weighted",
					"weighted": map[string]interface{}{
						"semantic_weight": 0.7,
						"keyword_weight":  0.3,
					},
				},
				"reranker": map[string]interface{}{
					"enabled":         true,
					"provider":        "cohere",
					"model":           "rerank-english-v3.0",
					"top_k":           5,
					"score_threshold": 0.7,
					"timeout":         "2s",
				},
			},
			"rag": map[string]interface{}{
				"engine": map[string]interface{}{
					"max_context_tokens": 4000,
					"max_chunks":         5,
				},
			},
		},
	}

	encoder := yaml.NewEncoder(configFile)
	err = encoder.Encode(cfgMap)
	assert.NoError(t, err)
	configFile.Close()

	// Load the config
	loadedCfg, err := config.LoadConfig(configFile.Name())
	assert.NoError(t, err)

	// Verify loaded values
	assert.Equal(t, "hybrid", loadedCfg.Knowledge.Retrieval.Mode)
	assert.Equal(t, "chroma", loadedCfg.Knowledge.Retrieval.Semantic.Provider)
	assert.Equal(t, 0.6, loadedCfg.Knowledge.Retrieval.Semantic.ScoreThreshold)
	assert.Equal(t, "bleve", loadedCfg.Knowledge.Retrieval.Keyword.Engine)
	assert.Equal(t, "weighted", loadedCfg.Knowledge.Retrieval.Fusion.Strategy)
	assert.Equal(t, 0.7, loadedCfg.Knowledge.Retrieval.Fusion.Weighted.SemanticWeight)
	assert.Equal(t, "cohere", loadedCfg.Knowledge.Retrieval.Reranker.Provider)
	assert.Equal(t, 4000, loadedCfg.Knowledge.RAG.Engine.MaxContextTokens)
}
