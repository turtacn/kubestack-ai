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
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// ElasticsearchDocumentStore is a production-ready implementation of DocumentStore that
// uses Elasticsearch as the backend. It handles the connection to an Elasticsearch
// cluster and the translation of store operations to Elasticsearch API calls.
type ElasticsearchDocumentStore struct {
	log    logger.Logger
	client *elasticsearch.Client
	index  string
}

// NewElasticsearchDocumentStore creates a new document store connected to an Elasticsearch instance.
// It initializes the client and ensures that the target index exists.
func NewElasticsearchDocumentStore(addresses []string, indexName string) (DocumentStore, error) {
	log := logger.NewLogger("es-doc-store")
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Ping the server to verify the connection
	if _, err := client.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping elasticsearch: %w", err)
	}

	// Check if the index exists
	res, err := client.Indices.Exists([]string{indexName})
	if err != nil {
		return nil, fmt.Errorf("failed to check if index exists: %w", err)
	}
	// The status code for a missing index is 404.
	if res.StatusCode == 404 {
		log.Infof("Index '%s' not found, creating it now.", indexName)
		// Create the index with a simple mapping
		createRes, err := client.Indices.Create(indexName)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
		if createRes.IsError() {
			return nil, fmt.Errorf("failed to create index: %s", createRes.String())
		}
	}

	log.Infof("Elasticsearch document store initialized for index '%s'", indexName)
	return &ElasticsearchDocumentStore{
		log:    log,
		client: client,
		index:  indexName,
	}, nil
}

// Add saves a new document to the Elasticsearch index.
func (s *ElasticsearchDocumentStore) Add(ctx context.Context, doc *RawDocument) (string, error) {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	res, err := s.client.Index(s.index, bytes.NewReader(body), s.client.Index.WithDocumentID(doc.ID), s.client.Index.WithRefresh("true"))
	if err != nil {
		return "", fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("failed to index document: %s", res.String())
	}

	return doc.ID, nil
}

// Get retrieves a document by its unique ID from the Elasticsearch index.
func (s *ElasticsearchDocumentStore) Get(ctx context.Context, id string) (*RawDocument, error) {
	res, err := s.client.Get(s.index, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, fmt.Errorf("document with ID '%s' not found", id)
		}
		return nil, fmt.Errorf("failed to get document: %s", res.String())
	}

	var esResponse struct {
		Source *RawDocument `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode document response: %w", err)
	}

	return esResponse.Source, nil
}

// Update modifies an existing document in the Elasticsearch index by re-indexing it.
func (s *ElasticsearchDocumentStore) Update(ctx context.Context, doc *RawDocument) error {
	doc.UpdatedAt = time.Now().UTC()

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document for update: %w", err)
	}

	res, err := s.client.Index(s.index, bytes.NewReader(body), s.client.Index.WithDocumentID(doc.ID), s.client.Index.WithRefresh("true"))
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update document: %s", res.String())
	}
	return nil
}

// Delete removes a document from the Elasticsearch index by its unique ID.
func (s *ElasticsearchDocumentStore) Delete(ctx context.Context, id string) error {
	res, err := s.client.Delete(s.index, id, s.client.Delete.WithRefresh("true"))
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return fmt.Errorf("document with ID '%s' not found for deletion", id)
		}
		return fmt.Errorf("failed to delete document: %s", res.String())
	}
	return nil
}

// Search performs a keyword search and/or tag-based filtering on the Elasticsearch index.
func (s *ElasticsearchDocumentStore) Search(ctx context.Context, query string, tags []string) ([]*RawDocument, error) {
	var mustClauses []interface{}
	if query != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"Content": query,
			},
		})
	}

	var filterClauses []interface{}
	if len(tags) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"Tags": tags,
			},
		})
	}

	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   mustClauses,
				"filter": filterClauses,
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(esQuery); err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(s.index),
		s.client.Search.WithBody(&buf),
		s.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch search returned an error: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	var results []*RawDocument
	hits, ok := r["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected search response format")
	}

	for _, hit := range hits {
		source, ok := hit.(map[string]interface{})["_source"]
		if !ok {
			continue
		}
		sourceBytes, err := json.Marshal(source)
		if err != nil {
			continue
		}
		var doc RawDocument
		if err := json.Unmarshal(sourceBytes, &doc); err == nil {
			results = append(results, &doc)
		}
	}

	return results, nil
}