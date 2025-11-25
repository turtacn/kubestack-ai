package alert

import (
	"context"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

type SilenceManager struct {
	silences map[string]*types.Silence
	mu       sync.RWMutex
	store    storage.SilenceStore
}

// NewSilenceManager creates a new silence manager
func NewSilenceManager(store storage.SilenceStore) *SilenceManager {
	m := &SilenceManager{
		silences: make(map[string]*types.Silence),
		store:    store,
	}
	// Load active silences
	if store != nil {
		ctx := context.Background()
		active, err := store.ListActive(ctx, time.Now())
		if err == nil {
			for _, s := range active {
				m.silences[s.ID] = s
			}
		}
	}
	return m
}

// Add adds a silence rule
func (m *SilenceManager) Add(ctx context.Context, silence *types.Silence) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if silence.ID == "" {
		silence.ID = uuid.New().String()
	}
	m.silences[silence.ID] = silence

	if m.store != nil {
		return m.store.Save(ctx, silence)
	}
	return nil
}

// IsSilenced checks if an alert is silenced
func (m *SilenceManager) IsSilenced(ruleName string, labels map[string]string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()

	for _, silence := range m.silences {
		// Check time window
		if now.Before(silence.StartTime) || now.After(silence.EndTime) {
			continue
		}

		// Check rule name matching
		if silence.RuleName != "" && silence.RuleName != ruleName {
			continue
		}

		// Check label matching (all silence labels must match alert labels)
		if !m.labelsMatch(silence.Labels, labels) {
			continue
		}

		return true
	}

	return false
}

func (m *SilenceManager) labelsMatch(silenceLabels, alertLabels map[string]string) bool {
	for key, value := range silenceLabels {
		if alertLabels[key] != value {
			return false
		}
	}
	return true
}

// GC removes expired silences
func (m *SilenceManager) GC(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.removeExpired()
		}
	}
}

func (m *SilenceManager) removeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, silence := range m.silences {
		if now.After(silence.EndTime) {
			delete(m.silences, id)
			if m.store != nil {
				_ = m.store.Delete(context.Background(), id)
			}
		}
	}
}
