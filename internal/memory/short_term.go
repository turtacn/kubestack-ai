package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/memory/store"
)

// ShortTermMemory manages cross-session memory with persistence
type ShortTermMemory struct {
	store store.Store
	ttl   time.Duration
}

// NewShortTermMemory creates a new ShortTermMemory instance
func NewShortTermMemory(store store.Store, ttl time.Duration) *ShortTermMemory {
	return &ShortTermMemory{
		store: store,
		ttl:   ttl,
	}
}

// Save stores all entries for a session
func (s *ShortTermMemory) Save(sessionID string, entries []MemoryEntry) error {
	key := s.makeKey(sessionID)
	
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	return s.store.SetWithTTL(key, data, s.ttl)
}

// Load retrieves all entries for a session
func (s *ShortTermMemory) Load(sessionID string) ([]MemoryEntry, error) {
	key := s.makeKey(sessionID)
	
	data, err := s.store.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries: %w", err)
	}

	var entries []MemoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entries: %w", err)
	}

	return entries, nil
}

// Append adds a new entry to existing session data
func (s *ShortTermMemory) Append(sessionID string, entry MemoryEntry) error {
	entries, err := s.Load(sessionID)
	if err != nil {
		entries = []MemoryEntry{}
	}

	entries = append(entries, entry)
	return s.Save(sessionID, entries)
}

// Delete removes all entries for a session
func (s *ShortTermMemory) Delete(sessionID string) error {
	key := s.makeKey(sessionID)
	return s.store.Delete(key)
}

func (s *ShortTermMemory) makeKey(sessionID string) string {
	return "session:" + sessionID
}
