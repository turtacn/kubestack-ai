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
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

// RAGEngine orchestrates the Retrieval-Augmented Generation (RAG) pipeline.
type RAGEngine struct {
	searcher  search.Searcher
	llmClient llm.LLMClient
	cfg       config.RAGEngineConfig
}

// NewRAGEngine creates a new RAGEngine.
func NewRAGEngine(
	knowledgeCfg config.KnowledgeConfig,
	vectorRetriever search.Retriever,
	bm25Searcher *search.BM25Searcher,
	reranker search.Reranker,
	llmClient llm.LLMClient,
) (*RAGEngine, error) {
	var searcher search.Searcher
	var err error

	switch knowledgeCfg.Retrieval.Mode {
	case "hybrid":
		searcher, err = search.NewHybridSearcher(vectorRetriever, bm25Searcher, reranker, knowledgeCfg.Retrieval)
	case "semantic":
		searcher, err = search.NewSemanticSearcher(vectorRetriever)
	default:
		return nil, fmt.Errorf("unknown retrieval mode: %s", knowledgeCfg.Retrieval.Mode)
	}

	if err != nil {
		return nil, err
	}

	return &RAGEngine{
		searcher:  searcher,
		llmClient: llmClient,
		cfg:       knowledgeCfg.RAG.Engine,
	}, nil
}

// GenerateAnswer performs the end-to-end RAG process.
func (e *RAGEngine) GenerateAnswer(ctx context.Context, question string) (string, error) {
	// 1. Retrieve relevant documents
	docs, err := e.searcher.Search(ctx, question)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// 2. Build the prompt with the retrieved context
	tmpl := prompt.Template{
		Name:    "rag",
		Content: "Answer the following question based on the provided context:\n\n{{range .context}}Context: {{.}}\n{{end}}\nQuestion: {{.question}}",
	}
	builder, err := prompt.NewBuilder(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to create prompt builder: %w", err)
	}

	docContents := make([]string, len(docs))
	for i, doc := range docs {
		docContents[i] = doc.Content
	}

	prompt, err := builder.
		WithData("context", docContents).
		WithData("question", question).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	// 3. Generate the answer using the language model
	answer, err := e.llmClient.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, nil
}
