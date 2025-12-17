package memory

// LongTermMemory is the interface for long-term vector storage
type LongTermMemory interface {
	Store(entry MemoryEntry) error
	Search(query string, topK int) ([]MemoryEntry, error)
	Delete(id string) error
}

// NoOpLongTermMemory is a placeholder implementation
type NoOpLongTermMemory struct{}

// NewNoOpLongTermMemory creates a new no-op long-term memory
func NewNoOpLongTermMemory() *NoOpLongTermMemory {
	return &NoOpLongTermMemory{}
}

// Store does nothing
func (n *NoOpLongTermMemory) Store(entry MemoryEntry) error {
	return nil
}

// Search returns empty results
func (n *NoOpLongTermMemory) Search(query string, topK int) ([]MemoryEntry, error) {
	return []MemoryEntry{}, nil
}

// Delete does nothing
func (n *NoOpLongTermMemory) Delete(id string) error {
	return nil
}
