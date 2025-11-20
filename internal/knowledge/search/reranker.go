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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

// Reranker defines the interface for a component that re-ranks a list of
// documents based on their relevance to a given query. This is typically done
// using a more powerful model (like a cross-encoder) than the one used for
// initial retrieval.
type Reranker interface {
	// Rerank takes a query and a list of candidate documents and returns a new,
	// re-ordered list of documents with updated relevance scores.
	Rerank(ctx context.Context, query string, candidates []*Document, topK int) ([]*Document, error)
}

// --- API-Based Reranker Implementation ---

// APIReranker implements the Reranker interface by calling an external reranking API.
type APIReranker struct {
	client   *http.Client
	endpoint string
	apiKey   string
}

// NewAPIReranker creates a new APIReranker.
func NewAPIReranker(endpoint, apiKey string) Reranker {
	return &APIReranker{
		client:   &http.Client{},
		endpoint: endpoint,
		apiKey:   apiKey,
	}
}

// Rerank sends the query and documents to a reranking API and re-orders the
// documents based on the returned scores.
func (r *APIReranker) Rerank(ctx context.Context, query string, candidates []*Document, topK int) ([]*Document, error) {
	texts := make([]string, len(candidates))
	for i, doc := range candidates {
		texts[i] = doc.Content
	}

	payload := map[string]interface{}{
		"query":     query,
		"documents": texts,
		"top_n":     topK,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reranker payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call reranker API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reranker API returned non-OK status: %s", resp.Status)
	}

	var rerankResponse struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rerankResponse); err != nil {
		return nil, fmt.Errorf("failed to decode reranker response: %w", err)
	}

	rerankedDocs := make([]*Document, 0, topK)
	for _, result := range rerankResponse.Results {
		if result.Index < len(candidates) {
			doc := candidates[result.Index]
			doc.Score = float32(result.Score)
			rerankedDocs = append(rerankedDocs, doc)
		}
	}

	// Sort by the new score in descending order
	sort.Slice(rerankedDocs, func(i, j int) bool {
		return rerankedDocs[i].Score > rerankedDocs[j].Score
	})

	return rerankedDocs, nil
}

// --- Local ONNX Reranker (Placeholder) ---

// LocalONNXReranker implements the Reranker interface using a local ONNX model.
type LocalONNXReranker struct {
	// session *ort.Session // ONNX Runtime session
}

// NewLocalONNXReranker creates a new LocalONNXReranker.
func NewLocalONNXReranker(modelPath string) (Reranker, error) {
	// Placeholder for ONNX model loading
	return &LocalONNXReranker{}, nil
}

// Rerank performs reranking using the local ONNX model.
func (r *LocalONNXReranker) Rerank(ctx context.Context, query string, candidates []*Document, topK int) ([]*Document, error) {
	// Placeholder for ONNX reranking logic
	// In a real implementation, you would:
	// 1. Tokenize the query and documents
	// 2. Run the model to get relevance scores
	// 3. Re-rank the documents based on the scores
	return candidates, nil // For now, just return the original candidates
}
