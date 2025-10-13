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
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// elasticsearchDocumentStore is a production-grade implementation of DocumentStore that uses
// Elasticsearch as its backend. It handles the connection to Elasticsearch, management of
// indices, and the translation between the application's data models and the Elasticsearch API.
type elasticsearchDocumentStore struct {
	log       logger.Logger
	client    *elasticsearch.Client
	indexName string
}

// NewElasticsearchDocumentStore creates a new document store that connects to an Elasticsearch instance.
func NewElasticsearchDocumentStore(cfg *config.ElasticsearchConfig) (DocumentStore, error) {
	log := logger.NewLogger("es-doc-store")

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new elasticsearch client: %w", err)
	}

	res, err := esClient.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer res.Body.Close()
	log.Infof("Successfully connected to Elasticsearch: %s", res.String())

	store := &elasticsearchDocumentStore{
		log:       log,
		client:    esClient,
		indexName: cfg.IndexName,
	}

	if err := store.ensureIndex(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure elasticsearch index exists: %w", err)
	}

	return store, nil
}

// Add saves a new document to the Elasticsearch index.
func (s *elasticsearchDocumentStore) Add(ctx context.Context, doc *RawDocument) (string, error) {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      s.indexName,
		DocumentID: doc.ID,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return "", fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("failed to index document: %s", res.String())
	}

	s.log.Infof("Added document with ID: %s", doc.ID)
	return doc.ID, nil
}

// Get retrieves a document by its unique ID from Elasticsearch.
func (s *elasticsearchDocumentStore) Get(ctx context.Context, id string) (*RawDocument, error) {
	req := esapi.GetRequest{
		Index:      s.indexName,
		DocumentID: id,
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to get document: %s", res.String())
	}

	var esResponse struct {
		Source *RawDocument `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode get response: %w", err)
	}

	return esResponse.Source, nil
}

// Update modifies an existing document in the Elasticsearch index.
func (s *elasticsearchDocumentStore) Update(ctx context.Context, doc *RawDocument) error {
	doc.UpdatedAt = time.Now().UTC()

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document for update: %w", err)
	}

	req := esapi.UpdateRequest{
		Index:      s.indexName,
		DocumentID: doc.ID,
		Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, body))),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update document: %s", res.String())
	}
	return nil
}

// Delete removes a document from the Elasticsearch index by its ID.
func (s *elasticsearchDocumentStore) Delete(ctx context.Context, id string) error {
	req := esapi.DeleteRequest{
		Index:      s.indexName,
		DocumentID: id,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to delete document: %s", res.String())
	}
	return nil
}

// Search performs a keyword search and/or tag-based filtering in Elasticsearch.
func (s *elasticsearchDocumentStore) Search(ctx context.Context, query string, tags []string) ([]*RawDocument, error) {
	var esQuery map[string]interface{}
	if query != "" {
		esQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  query,
					"fields": []string{"Content", "Source"},
				},
			},
		}
	} else {
		esQuery = map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}}
	}

	// This is a simplified filter for tags. A more robust implementation would use a proper filter clause.
	if len(tags) > 0 {
		esQuery["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": esQuery["query"],
				"filter": map[string]interface{}{
					"terms": map[string]interface{}{
						"Tags": tags,
					},
				},
			},
		}
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(esQuery); err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{s.indexName},
		Body:  &buf,
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to search documents: %s", res.String())
	}

	var esResponse struct {
		Hits struct {
			Hits []struct {
				Source *RawDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	var results []*RawDocument
	for _, hit := range esResponse.Hits.Hits {
		results = append(results, hit.Source)
	}

	return results, nil
}

// ensureIndex checks if the target index exists, and if not, creates it.
func (s *elasticsearchDocumentStore) ensureIndex(ctx context.Context) error {
	req := esapi.IndicesExistsRequest{
		Index: []string{s.indexName},
	}
	res, err := req.Do(ctx, s.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		s.log.Debugf("Index '%s' already exists.", s.indexName)
		return nil
	}

	if res.StatusCode == http.StatusNotFound {
		s.log.Infof("Index '%s' not found, creating it.", s.indexName)
		createReq := esapi.IndicesCreateRequest{
			Index: s.indexName,
		}
		createRes, err := createReq.Do(ctx, s.client)
		if err != nil {
			return err
		}
		defer createRes.Body.Close()
		if createRes.IsError() {
			return fmt.Errorf("failed to create index: %s", createRes.String())
		}
		return nil
	}

	return fmt.Errorf("failed to check for index '%s', status: %s", s.indexName, res.Status())
}