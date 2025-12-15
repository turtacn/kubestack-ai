package indexer

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
)

// IndexerStore defines the interface for underlying storage operations.
type IndexerStore interface {
	Add(ctx context.Context, docs []models.Document) error
	Delete(ctx context.Context, docID string) error
	// Update usually involves delete then add in vector stores
}

type Indexer struct {
	store IndexerStore
}

func NewIndexer(store IndexerStore) *Indexer {
	return &Indexer{store: store}
}

func (i *Indexer) UpdateDocument(ctx context.Context, doc *models.Document) error {
	// 1. Delete old vector/doc
	if err := i.store.Delete(ctx, doc.ID); err != nil {
		return fmt.Errorf("failed to delete old document: %w", err)
	}

	// 2. Add new document
	if err := i.store.Add(ctx, []models.Document{*doc}); err != nil {
		return fmt.Errorf("failed to add new document: %w", err)
	}

	return nil
}

func (i *Indexer) DeleteDocument(ctx context.Context, docID string) error {
	return i.store.Delete(ctx, docID)
}

func (i *Indexer) BatchUpdate(ctx context.Context, docs []*models.Document) error {
	// Naive implementation: process sequentially or in groups
	// Real implementation should probably batch deletes and adds if the store supports it.

	for _, doc := range docs {
		if err := i.UpdateDocument(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}
