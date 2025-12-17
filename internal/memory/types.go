package memory

import (
	"time"
)

// MemoryEntry represents a single memory entry
type MemoryEntry struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"` // "user", "assistant", "system"
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryQuery represents query parameters for retrieving memory
type MemoryQuery struct {
	SessionID string
	Limit     int
	Before    *time.Time
	After     *time.Time
	Role      string
}

// MemoryLayer represents the memory layer type
type MemoryLayer int

const (
	Working MemoryLayer = iota
	ShortTerm
	LongTerm
)

func (m MemoryLayer) String() string {
	switch m {
	case Working:
		return "working"
	case ShortTerm:
		return "short_term"
	case LongTerm:
		return "long_term"
	default:
		return "unknown"
	}
}

// MemoryConfig represents configuration for memory system
type MemoryConfig struct {
	WorkingWindowSize int           `yaml:"working_window_size" json:"working_window_size"`
	ShortTermTTL      time.Duration `yaml:"short_term_ttl" json:"short_term_ttl"`
	StorePath         string        `yaml:"store_path" json:"store_path"`
}

// DefaultMemoryConfig returns default memory configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		WorkingWindowSize: 20,
		ShortTermTTL:      24 * time.Hour * 7, // 7 days
		StorePath:         "./data/memory",
	}
}
