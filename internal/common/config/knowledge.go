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

package config

import (
	"fmt"
	"time"
)

// KnowledgeConfig is the top-level configuration for all knowledge-base related operations.
type KnowledgeConfig struct {
	DefaultIndex string          `yaml:"default_index"`
	Language     string          `yaml:"language"`
	Retrieval    RetrievalConfig `yaml:"retrieval"`
	RAG          RAGConfig       `yaml:"rag"`
}

// RetrievalConfig holds settings for the retrieval process.
type RetrievalConfig struct {
	Mode     string         `yaml:"mode"`
	Semantic SemanticConfig `yaml:"semantic"`
	Keyword  KeywordConfig  `yaml:"keyword"`
	Fusion   FusionConfig   `yaml:"fusion"`
	Reranker RerankerConfig `yaml:"reranker"`
}

// SemanticConfig holds settings for semantic search.
type SemanticConfig struct {
	Enabled        bool    `yaml:"enabled"`
	Provider       string  `yaml:"provider"`
	Model          string  `yaml:"model"`
	TopK           int     `yaml:"top_k"`
	ScoreThreshold float64 `yaml:"score_threshold"`
}

// KeywordConfig holds settings for keyword search.
type KeywordConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Engine   string `yaml:"engine"`
	Analyzer string `yaml:"analyzer"`
	TopK     int    `yaml:"top_k"`
}

// FusionConfig holds settings for combining search results.
type FusionConfig struct {
	Strategy string         `yaml:"strategy"`
	RRF      RRFConfig      `yaml:"rrf"`
	Weighted WeightedConfig `yaml:"weighted"`
}

// RRFConfig holds settings for Reciprocal Rank Fusion.
type RRFConfig struct {
	K int `yaml:"k"`
}

// WeightedConfig holds settings for weighted sum fusion.
type WeightedConfig struct {
	SemanticWeight float64 `yaml:"semantic_weight"`
	KeywordWeight  float64 `yaml:"keyword_weight"`
}

// RerankerConfig holds settings for the reranking process.
type RerankerConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Provider       string        `yaml:"provider"`
	Model          string        `yaml:"model"`
	TopK           int           `yaml:"top_k"`
	ScoreThreshold float64       `yaml:"score_threshold"`
	Timeout        time.Duration `yaml:"timeout"`
}

// RAGConfig holds settings for the Retrieval-Augmented Generation process.
type RAGConfig struct {
	Engine RAGEngineConfig `yaml:"engine"`
}

// RAGEngineConfig holds settings for the RAG engine.
type RAGEngineConfig struct {
	MaxContextTokens int `yaml:"max_context_tokens"`
	MaxChunks        int `yaml:"max_chunks"`
}

// Validate checks the configuration for common errors.
func (c *KnowledgeConfig) Validate() error {
	if c.Retrieval.Mode == "hybrid" && c.Retrieval.Fusion.Strategy == "weighted" {
		if c.Retrieval.Fusion.Weighted.SemanticWeight+c.Retrieval.Fusion.Weighted.KeywordWeight != 1.0 {
			return fmt.Errorf("semantic_weight and keyword_weight must sum to 1.0")
		}
	}
	if c.Retrieval.Semantic.TopK <= 0 {
		return fmt.Errorf("semantic top_k must be greater than 0")
	}
	if c.Retrieval.Keyword.TopK <= 0 {
		return fmt.Errorf("keyword top_k must be greater than 0")
	}
	if c.Retrieval.Reranker.TopK <= 0 {
		return fmt.Errorf("reranker top_k must be greater than 0")
	}
	return nil
}
