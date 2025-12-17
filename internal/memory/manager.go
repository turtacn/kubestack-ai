package memory

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/memory/store"
)

// MemoryManager orchestrates all memory layers
type MemoryManager struct {
	working   *WorkingMemory
	shortTerm *ShortTermMemory
	longTerm  LongTermMemory
	config    MemoryConfig
}

// NewMemoryManager creates a new MemoryManager instance
func NewMemoryManager(cfg MemoryConfig) (*MemoryManager, error) {
	badgerStore, err := store.NewBadgerStore(cfg.StorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create badger store: %w", err)
	}

	return &MemoryManager{
		working:   NewWorkingMemory(cfg.WorkingWindowSize),
		shortTerm: NewShortTermMemory(badgerStore, cfg.ShortTermTTL),
		longTerm:  NewNoOpLongTermMemory(),
		config:    cfg,
	}, nil
}

// RecordMessage records a new message in memory
func (m *MemoryManager) RecordMessage(sessionID string, entry MemoryEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.SessionID == "" {
		entry.SessionID = sessionID
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if err := m.working.Add(entry); err != nil {
		return fmt.Errorf("failed to add to working memory: %w", err)
	}

	if err := m.shortTerm.Append(sessionID, entry); err != nil {
		return fmt.Errorf("failed to append to short-term memory: %w", err)
	}

	return nil
}

// GetContext retrieves conversation context with optional token limit
func (m *MemoryManager) GetContext(sessionID string, maxTokens int) ([]MemoryEntry, error) {
	entries := m.working.GetAll()
	
	if len(entries) == 0 {
		loadedEntries, err := m.shortTerm.Load(sessionID)
		if err == nil {
			entries = loadedEntries
		}
	}

	if maxTokens > 0 {
		entries = m.truncateByTokens(entries, maxTokens)
	}

	return entries, nil
}

// LoadSession loads a session from short-term memory into working memory
func (m *MemoryManager) LoadSession(sessionID string) error {
	entries, err := m.shortTerm.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	m.working.Clear()
	for _, entry := range entries {
		if err := m.working.Add(entry); err != nil {
			return fmt.Errorf("failed to add entry to working memory: %w", err)
		}
	}

	return nil
}

// SaveSession saves current working memory to short-term memory
func (m *MemoryManager) SaveSession(sessionID string) error {
	entries := m.working.GetAll()
	return m.shortTerm.Save(sessionID, entries)
}

// ClearWorking clears the working memory
func (m *MemoryManager) ClearWorking() {
	m.working.Clear()
}

// Close closes all storage connections
func (m *MemoryManager) Close() error {
	if closer, ok := m.shortTerm.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// truncateByTokens approximates token count and truncates entries
func (m *MemoryManager) truncateByTokens(entries []MemoryEntry, maxTokens int) []MemoryEntry {
	estimatedTokens := 0
	result := make([]MemoryEntry, 0, len(entries))

	for i := len(entries) - 1; i >= 0; i-- {
		entryTokens := len(entries[i].Content) / 4
		if estimatedTokens+entryTokens > maxTokens {
			break
		}
		estimatedTokens += entryTokens
		result = append([]MemoryEntry{entries[i]}, result...)
	}

	return result
}
