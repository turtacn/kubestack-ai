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
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// --- Direct HTTP Client Implementation for ChromaDB V2 API ---

// ChromaVectorStore is a production-ready implementation of VectorStore that uses
// direct HTTP calls to the ChromaDB V2 REST API. This approach avoids issues with
// the Go client library.
type ChromaVectorStore struct {
	log          logger.Logger
	httpClient   *http.Client
	apiBaseURL   string
	collectionID string
}

// Chroma API V2 Request/Response Structs
type chromaAddRequest struct {
	IDs        []string                 `json:"ids"`
	Embeddings [][]float32              `json:"embeddings"`
	Metadatas  []map[string]interface{} `json:"metadatas"`
	Documents  []string                 `json:"documents"`
}

type chromaQueryRequest struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings"`
	NResults        int         `json:"n_results"`
	Include         []string    `json:"include"`
}

type chromaQueryResponse struct {
	IDs        [][]string                 `json:"ids"`
	Distances  [][]float32                `json:"distances"`
	Metadatas  [][]map[string]interface{} `json:"metadatas"`
	Embeddings [][][]float32              `json:"embeddings"`
	Documents  [][]string                 `json:"documents"`
}

type chromaCollection struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewChromaVectorStore creates a new vector store that connects directly to the
// ChromaDB V2 API.
func NewChromaVectorStore(url string, collectionName string) (VectorStore, error) {
	log := logger.NewLogger("chroma-http-store")
	client := &http.Client{Timeout: 30 * time.Second}
	apiBaseURL := fmt.Sprintf("%s/api/v2", url)

	// Get or create the collection
	collection, err := getOrCreateCollection(client, apiBaseURL, collectionName)
	if err != nil {
		return nil, err
	}

	log.Infof("Using ChromaDB collection '%s' with ID: %s", collection.Name, collection.ID)

	return &ChromaVectorStore{
		log:          log,
		httpClient:   client,
		apiBaseURL:   apiBaseURL,
		collectionID: collection.ID,
	}, nil
}

// AddDocuments adds a batch of documents to the ChromaDB collection via the REST API.
func (c *ChromaVectorStore) AddDocuments(ctx context.Context, docs []StoreDocument) error {
	if len(docs) == 0 {
		return nil
	}
	c.log.Infof("Adding %d documents to collection ID '%s'", len(docs), c.collectionID)

	addReq := chromaAddRequest{
		IDs:        getDocIDs(docs),
		Embeddings: getDocVectors(docs),
		Metadatas:  getDocMetadatas(docs),
		Documents:  getDocContents(docs),
	}

	endpoint := fmt.Sprintf("/collections/%s/add", c.collectionID)
	_, err := c.doRequest(ctx, http.MethodPost, endpoint, addReq)
	if err != nil {
		return fmt.Errorf("failed to add documents via Chroma API: %w", err)
	}

	return nil
}

// SimilaritySearch performs a similarity search on the ChromaDB collection via the REST API.
func (c *ChromaVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error) {
	c.log.Debugf("Performing similarity search for top %d results in collection ID '%s'", topK, c.collectionID)

	queryReq := chromaQueryRequest{
		QueryEmbeddings: [][]float32{queryVector},
		NResults:        topK,
		Include:         []string{"Metadatas", "Documents", "Distances", "Embeddings"},
	}

	endpoint := fmt.Sprintf("/collections/%s/query", c.collectionID)
	respBody, err := c.doRequest(ctx, http.MethodPost, endpoint, queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to query Chroma API: %w", err)
	}

	var queryResp chromaQueryResponse
	if err := json.Unmarshal(respBody, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Chroma query response: %w", err)
	}

	return convertChromaHTTPResultsToStoreDocuments(queryResp)
}

// --- Helper Functions ---

func (c *ChromaVectorStore) doRequest(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	url := c.apiBaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("chroma api request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func getOrCreateCollection(client *http.Client, baseURL, name string) (*chromaCollection, error) {
	// Try to get the collection first
	getEndpoint := fmt.Sprintf("%s/collections/%s", baseURL, name)
	req, _ := http.NewRequest(http.MethodGet, getEndpoint, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send get collection request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var collection chromaCollection
		if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
			return nil, fmt.Errorf("failed to decode existing collection: %w", err)
		}
		return &collection, nil
	}

	// If it doesn't exist (e.g., 404), create it
	if resp.StatusCode == http.StatusNotFound {
		createEndpoint := fmt.Sprintf("%s/collections", baseURL)
		createPayload := map[string]string{"name": name}
		payloadBytes, _ := json.Marshal(createPayload)
		req, _ = http.NewRequest(http.MethodPost, createEndpoint, bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		createResp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send create collection request: %w", err)
		}
		defer createResp.Body.Close()

		if createResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(createResp.Body)
			return nil, fmt.Errorf("failed to create collection, status %d: %s", createResp.StatusCode, string(body))
		}
		var collection chromaCollection
		if err := json.NewDecoder(createResp.Body).Decode(&collection); err != nil {
			return nil, fmt.Errorf("failed to decode created collection: %w", err)
		}
		return &collection, nil
	}

	// Handle other unexpected status codes
	body, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("unexpected status code when getting collection: %d, body: %s", resp.StatusCode, string(body))
}

func getDocIDs(docs []StoreDocument) []string {
	ids := make([]string, len(docs))
	for i, doc := range docs {
		if doc.ID != "" {
			ids[i] = doc.ID
		} else {
			ids[i] = uuid.New().String()
		}
	}
	return ids
}

func getDocVectors(docs []StoreDocument) [][]float32 {
	vectors := make([][]float32, len(docs))
	for i, doc := range docs {
		vectors[i] = doc.Vector
	}
	return vectors
}

func getDocMetadatas(docs []StoreDocument) []map[string]interface{} {
	metadatas := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		metadatas[i] = doc.Metadata
	}
	return metadatas
}

func getDocContents(docs []StoreDocument) []string {
	contents := make([]string, len(docs))
	for i, doc := range docs {
		contents[i] = doc.Content
	}
	return contents
}

func convertChromaHTTPResultsToStoreDocuments(resp chromaQueryResponse) ([]StoreDocument, error) {
	if len(resp.IDs) == 0 || len(resp.IDs[0]) == 0 {
		return []StoreDocument{}, nil
	}

	numDocs := len(resp.IDs[0])
	docs := make([]StoreDocument, numDocs)

	for i := 0; i < numDocs; i++ {
		docs[i] = StoreDocument{
			ID:       resp.IDs[0][i],
			Content:  resp.Documents[0][i],
			Metadata: resp.Metadatas[0][i],
			Score:    1.0 - resp.Distances[0][i],
			Vector:   resp.Embeddings[0][i],
		}
	}

	return docs, nil
}