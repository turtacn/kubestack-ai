package planning

import (
	"encoding/json"
	"fmt"
	"sync"
)

// StateStore is the interface for persisting execution states
type StateStore interface {
	Save(state *ExecutionState) error
	Load(planID string) (*ExecutionState, error)
	Delete(planID string) error
	List() ([]*ExecutionState, error)
}

// MemoryStateStore is an in-memory implementation of StateStore
type MemoryStateStore struct {
	states map[string]*ExecutionState
	mu     sync.RWMutex
}

// NewMemoryStateStore creates a new MemoryStateStore
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		states: make(map[string]*ExecutionState),
	}
}

// Save saves an execution state
func (s *MemoryStateStore) Save(state *ExecutionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy to avoid external modifications
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	var copied ExecutionState
	if err := json.Unmarshal(data, &copied); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	s.states[state.PlanID] = &copied
	return nil
}

// Load loads an execution state
func (s *MemoryStateStore) Load(planID string) (*ExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.states[planID]
	if !exists {
		return nil, fmt.Errorf("state not found for plan: %s", planID)
	}

	// Deep copy to avoid external modifications
	data, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	var copied ExecutionState
	if err := json.Unmarshal(data, &copied); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &copied, nil
}

// Delete deletes an execution state
func (s *MemoryStateStore) Delete(planID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.states[planID]; !exists {
		return fmt.Errorf("state not found for plan: %s", planID)
	}

	delete(s.states, planID)
	return nil
}

// List returns all execution states
func (s *MemoryStateStore) List() ([]*ExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]*ExecutionState, 0, len(s.states))
	for _, state := range s.states {
		// Deep copy
		data, err := json.Marshal(state)
		if err != nil {
			continue
		}
		var copied ExecutionState
		if err := json.Unmarshal(data, &copied); err != nil {
			continue
		}
		states = append(states, &copied)
	}

	return states, nil
}

// PersistentStateStore is a persistent implementation using memory.Store
type PersistentStateStore struct {
	store interface{} // memory.Store interface
}

// NewPersistentStateStore creates a new PersistentStateStore
func NewPersistentStateStore(store interface{}) *PersistentStateStore {
	return &PersistentStateStore{
		store: store,
	}
}

// Save saves an execution state to persistent storage
func (s *PersistentStateStore) Save(state *ExecutionState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Use type assertion to call Set method
	type setter interface {
		Set(key string, value []byte) error
	}

	if st, ok := s.store.(setter); ok {
		key := fmt.Sprintf("plan:%s", state.PlanID)
		return st.Set(key, data)
	}

	return fmt.Errorf("store does not implement Set method")
}

// Load loads an execution state from persistent storage
func (s *PersistentStateStore) Load(planID string) (*ExecutionState, error) {
	type getter interface {
		Get(key string) ([]byte, error)
	}

	if st, ok := s.store.(getter); ok {
		key := fmt.Sprintf("plan:%s", planID)
		data, err := st.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get state: %w", err)
		}

		var state ExecutionState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		return &state, nil
	}

	return nil, fmt.Errorf("store does not implement Get method")
}

// Delete deletes an execution state from persistent storage
func (s *PersistentStateStore) Delete(planID string) error {
	type deleter interface {
		Delete(key string) error
	}

	if st, ok := s.store.(deleter); ok {
		key := fmt.Sprintf("plan:%s", planID)
		return st.Delete(key)
	}

	return fmt.Errorf("store does not implement Delete method")
}

// List returns all execution states from persistent storage
func (s *PersistentStateStore) List() ([]*ExecutionState, error) {
	type lister interface {
		List(prefix string) ([][]byte, error)
	}

	if st, ok := s.store.(lister); ok {
		values, err := st.List("plan:")
		if err != nil {
			return nil, fmt.Errorf("failed to list states: %w", err)
		}

		states := make([]*ExecutionState, 0, len(values))
		for _, data := range values {
			var state ExecutionState
			if err := json.Unmarshal(data, &state); err != nil {
				continue
			}
			states = append(states, &state)
		}

		return states, nil
	}

	return nil, fmt.Errorf("store does not implement List method")
}
