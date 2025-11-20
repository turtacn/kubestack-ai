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

// RawDocument represents an original, unprocessed document, such as the full
// content of a webpage or a markdown file. It is the source material from which
// smaller, embeddable chunks are created.
type RawDocument struct {
	// ID is the unique identifier for the document.
	ID string
	// Content is the full, unprocessed text content of the document.
	Content string
	// Source is the original location of the document (e.g., a URL or file path).
	Source string
	// Tags provide metadata for filtering and categorization.
	Tags []string
	// CreatedAt is the timestamp when the document was first added to the store.
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the document was last modified.
	UpdatedAt time.Time
}

// DocumentStore defines the interface for storing, retrieving, and searching
// original, full-text documents. This serves as the "source of truth" and
// complements the VectorStore, which only stores document chunks for semantic search.
// A production implementation might use a database like MongoDB or a search engine
// like Elasticsearch.
type DocumentStore interface {
	// Add saves a new document to the store and returns its unique ID.
	Add(ctx context.Context, doc *RawDocument) (string, error)
	// Get retrieves a document by its unique ID.
	Get(ctx context.Context, id string) (*RawDocument, error)
	// Update modifies an existing document in the store.
	Update(ctx context.Context, doc *RawDocument) error
	// Delete removes a document from the store by its unique ID.
	Delete(ctx context.Context, id string) error
	// Search performs a keyword search and/or tag-based filtering to find relevant documents.
	Search(ctx context.Context, query string, tags []string) ([]*RawDocument, error)
}

// --- In-Memory Document Store Implementation ---

// inMemoryDocumentStore is a simple, non-production implementation of DocumentStore.
type inMemoryDocumentStore struct {
	log       logger.Logger
	documents map[string]*RawDocument
	mu        sync.RWMutex
}

// newInMemoryDocumentStore creates a new, empty in-memory document store.
// This implementation is simple and useful for testing or small-scale deployments,
// but it is not durable and will lose all data when the application restarts.
//
// Returns:
//   DocumentStore: A new instance of an in-memory document store.
//   error: An error if initialization fails (nil in this implementation).
func newInMemoryDocumentStore() (DocumentStore, error) {
	return &inMemoryDocumentStore{
		log:       logger.NewLogger("in-memory-doc-store"),
		documents: make(map[string]*RawDocument),
	}, nil
}

// Add saves a new document to the in-memory map. It automatically assigns a new
// UUID if the document's ID is empty and sets the creation and update timestamps.
// This operation is thread-safe.
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

// Get retrieves a document by its unique ID from the in-memory map.
// This operation is thread-safe.
func (s *inMemoryDocumentStore) Get(_ context.Context, id string) (*RawDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, fmt.Errorf("document with ID '%s' not found", id)
	}
	return doc, nil
}

// Update modifies an existing document in the in-memory map. It finds the document
// by its ID and replaces it, updating the `UpdatedAt` timestamp.
// This operation is thread-safe.
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

// Delete removes a document from the in-memory map by its ID.
// This operation is thread-safe.
func (s *inMemoryDocumentStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[id]; !ok {
		return fmt.Errorf("document with ID '%s' not found for deletion", id)
	}
	delete(s.documents, id)
	return nil
}

// Search performs a simple, case-insensitive keyword search across the document
// content and/or filters by tags.
// NOTE: This is a basic implementation for demonstration purposes. A production
// system would use a dedicated full-text search engine like Elasticsearch for this.
// This operation is thread-safe.
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
