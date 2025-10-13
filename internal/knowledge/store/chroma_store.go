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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// chromaVectorStore is a production-grade implementation of VectorStore that uses
// ChromaDB as its backend. It interacts directly with the ChromaDB REST API.
type chromaVectorStore struct {
	log            logger.Logger
	httpClient     *http.Client
	url            string
	collectionName string
}

// NewChromaVectorStore creates a new vector store that connects to a ChromaDB instance via its REST API.
func NewChromaVectorStore(cfg *config.ChromaConfig) (VectorStore, error) {
	log := logger.NewLogger("chroma-http-store")
	collectionName := fmt.Sprintf("%s-%s", cfg.Namespace, cfg.CollectionName)

	store := &chromaVectorStore{
		log:            log,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		url:            cfg.URL,
		collectionName: collectionName,
	}

	if err := store.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure chroma collection exists: %w", err)
	}

	log.Infof("Successfully connected to ChromaDB at %s and ensured collection '%s' exists", cfg.URL, collectionName)
	return store, nil
}

// AddDocuments adds a batch of documents to the ChromaDB collection via the REST API.
func (s *chromaVectorStore) AddDocuments(ctx context.Context, docs []StoreDocument) error {
	if len(docs) == 0 {
		return nil
	}
	s.log.Infof("Adding %d documents to Chroma collection '%s'", len(docs), s.collectionName)

	ids := make([]string, len(docs))
	embeddings := make([][]float32, len(docs))
	metadatas := make([]map[string]interface{}, len(docs))
	documents := make([]string, len(docs))

	for i, doc := range docs {
		if doc.ID == "" {
			ids[i] = uuid.New().String()
		} else {
			ids[i] = doc.ID
		}
		embeddings[i] = doc.Vector
		metadatas[i] = doc.Metadata
		documents[i] = doc.Content
	}

	payload := map[string]interface{}{
		"ids":         ids,
		"embeddings":  embeddings,
		"metadatas":   metadatas,
		"documents":   documents,
	}

	endpoint := fmt.Sprintf("%s/api/v1/collections/%s/add", s.url, s.collectionName)
	_, err := s.doRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to add documents to chroma: %w", err)
	}
	return nil
}

// SimilaritySearch performs a similarity search in the ChromaDB collection via the REST API.
func (s *chromaVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error) {
	s.log.Debugf("Performing similarity search in Chroma for top %d results", topK)

	payload := map[string]interface{}{
		"query_embeddings": [][]float32{queryVector},
		"n_results":        topK,
		"include":          []string{"metadatas", "documents", "distances"},
	}

	endpoint := fmt.Sprintf("%s/api/v1/collections/%s/query", s.url, s.collectionName)
	body, err := s.doRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to query chroma: %w", err)
	}

	var results struct {
		IDs       [][]string                 `json:"ids"`
		Distances [][]float32              `json:"distances"`
		Metadatas [][]map[string]interface{} `json:"metadatas"`
		Documents [][]string                 `json:"documents"`
	}

	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chroma query response: %w", err)
	}

	if len(results.IDs) == 0 {
		return []StoreDocument{}, nil
	}

	var storeDocs []StoreDocument
	for i := 0; i < len(results.IDs[0]); i++ {
		storeDocs = append(storeDocs, StoreDocument{
			ID:       results.IDs[0][i],
			Content:  results.Documents[0][i],
			Metadata: results.Metadatas[0][i],
			Score:    results.Distances[0][i],
		})
	}

	return storeDocs, nil
}

// ensureCollection checks if the target collection exists, and if not, creates it.
func (s *chromaVectorStore) ensureCollection(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/api/v1/collections/%s", s.url, s.collectionName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		s.log.Debugf("Collection '%s' already exists.", s.collectionName)
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		s.log.Infof("Collection '%s' not found, creating it.", s.collectionName)
		createEndpoint := fmt.Sprintf("%s/api/v1/collections", s.url)
		payload := map[string]interface{}{"name": s.collectionName}
		_, err := s.doRequest(ctx, http.MethodPost, createEndpoint, payload)
		return err
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to check for collection '%s', status: %s, body: %s", s.collectionName, resp.Status, string(body))
}

// doRequest is a helper function to handle boilerplate for making HTTP requests.
func (s *chromaVectorStore) doRequest(ctx context.Context, method, url string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http request failed with status %s: %s", resp.Status, string(respBody))
	}

	return respBody, nil
}