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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// RawDocument represents an original, unprocessed document, such as a webpage or a markdown file.
type RawDocument struct {
	ID        string
	Content   string
	Source    string // e.g., URL, file path
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DocumentStore is the interface for storing and retrieving original, full-text documents.
// This complements the VectorStore by holding the source material from which vectors are generated.
// A real implementation might use a database like MongoDB or Elasticsearch.
type DocumentStore interface {
	Add(ctx context.Context, doc *RawDocument) (string, error)
	Get(ctx context.Context, id string) (*RawDocument, error)
	Update(ctx context.Context, doc *RawDocument) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, tags []string) ([]*RawDocument, error)
}

// --- In-Memory Document Store Implementation ---

// inMemoryDocumentStore is a simple, non-production implementation of DocumentStore.
type inMemoryDocumentStore struct {
	log       logger.Logger
	documents map[string]*RawDocument
	mu        sync.RWMutex
}

// NewInMemoryDocumentStore creates a new, empty in-memory document store.
func NewInMemoryDocumentStore() (DocumentStore, error) {
	return &inMemoryDocumentStore{
		log:       logger.NewLogger("in-memory-doc-store"),
		documents: make(map[string]*RawDocument),
	}, nil
}

// Add saves a new document and assigns it a UUID if it doesn't have one.
func (s *inMemoryDocumentStore) Add(_ context.Context, doc *RawDocument) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt

	s.documents[doc.ID] = doc
	s.log.Infof("Added document with ID: %s from source: %s", doc.ID, doc.Source)
	return doc.ID, nil
}

// Get retrieves a document by its unique ID.
func (s *inMemoryDocumentStore) Get(_ context.Context, id string) (*RawDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, fmt.Errorf("document with ID '%s' not found", id)
	}
	return doc, nil
}

// Update modifies an existing document.
func (s *inMemoryDocumentStore) Update(_ context.Context, doc *RawDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[doc.ID]; !ok {
		return fmt.Errorf("document with ID '%s' not found for update", doc.ID)
	}

	doc.UpdatedAt = time.Now().UTC()
	s.documents[doc.ID] = doc
	return nil
}

// Delete removes a document from the store.
func (s *inMemoryDocumentStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[id]; !ok {
		return fmt.Errorf("document with ID '%s' not found for deletion", id)
	}
	delete(s.documents, id)
	return nil
}

// Search performs a simple, case-insensitive keyword search and/or tag filtering.
// A production system would use a dedicated search engine like Elasticsearch for this.
func (s *inMemoryDocumentStore) Search(_ context.Context, query string, tags []string) ([]*RawDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*RawDocument
	lowerQuery := strings.ToLower(query)

	for _, doc := range s.documents {
		// Keyword search: check if content contains the query string.
		if query != "" && !strings.Contains(strings.ToLower(doc.Content), lowerQuery) {
			continue // Doesn't match keyword query.
		}

		// Tag filtering: check if the document has at least one of the required tags.
		if len(tags) > 0 {
			tagMatch := false
			for _, docTag := range doc.Tags {
				for _, searchTag := range tags {
					if docTag == searchTag {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue // Doesn't match tags.
			}
		}
		results = append(results, doc)
	}
	return results, nil
}

//Personal.AI order the ending
