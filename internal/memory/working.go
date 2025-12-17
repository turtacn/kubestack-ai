package memory

import (
	"sync"
)

// WorkingMemory manages current session memory in RAM
type WorkingMemory struct {
	entries []MemoryEntry
	maxSize int
	mu      sync.RWMutex
}

// NewWorkingMemory creates a new WorkingMemory instance
func NewWorkingMemory(maxSize int) *WorkingMemory {
	return &WorkingMemory{
		entries: make([]MemoryEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a new memory entry
func (w *WorkingMemory) Add(entry MemoryEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries = append(w.entries, entry)

	if len(w.entries) > w.maxSize {
		w.entries = w.entries[len(w.entries)-w.maxSize:]
	}

	return nil
}

// GetRecent retrieves the most recent n entries
func (w *WorkingMemory) GetRecent(n int) []MemoryEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if n <= 0 {
		return []MemoryEntry{}
	}

	start := len(w.entries) - n
	if start < 0 {
		start = 0
	}

	result := make([]MemoryEntry, len(w.entries)-start)
	copy(result, w.entries[start:])
	return result
}

// GetAll retrieves all entries
func (w *WorkingMemory) GetAll() []MemoryEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make([]MemoryEntry, len(w.entries))
	copy(result, w.entries)
	return result
}

// Clear removes all entries
func (w *WorkingMemory) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries = make([]MemoryEntry, 0, w.maxSize)
}

// Size returns the current number of entries
func (w *WorkingMemory) Size() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.entries)
}
