package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
	_ "github.com/mattn/go-sqlite3"
)

// SilenceStore defines the interface for storing silences
type SilenceStore interface {
	Save(ctx context.Context, silence *types.Silence) error
	Delete(ctx context.Context, id string) error
	ListActive(ctx context.Context, now time.Time) ([]*types.Silence, error)
	Close() error
}

// SQLiteSilenceStore implements SilenceStore using SQLite
type SQLiteSilenceStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteSilenceStore creates a new SQLite silence store
func NewSQLiteSilenceStore(path string) (*SQLiteSilenceStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	query := `
    CREATE TABLE IF NOT EXISTS silences (
        id TEXT PRIMARY KEY,
        rule_name TEXT,
        labels TEXT,
        start_time DATETIME NOT NULL,
        end_time DATETIME NOT NULL,
        created_by TEXT,
        comment TEXT
    );
    CREATE INDEX IF NOT EXISTS idx_silences_end_time ON silences(end_time);
    `
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init silence db: %w", err)
	}

	return &SQLiteSilenceStore{db: db}, nil
}

func (s *SQLiteSilenceStore) Save(ctx context.Context, silence *types.Silence) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	labelsJSON, _ := json.Marshal(silence.Labels)

	query := `INSERT OR REPLACE INTO silences (id, rule_name, labels, start_time, end_time, created_by, comment)
              VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		silence.ID, silence.RuleName, string(labelsJSON),
		silence.StartTime, silence.EndTime, silence.CreatedBy, silence.Comment)
	return err
}

func (s *SQLiteSilenceStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM silences WHERE id = ?", id)
	return err
}

func (s *SQLiteSilenceStore) ListActive(ctx context.Context, now time.Time) ([]*types.Silence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// List silences that are not expired (end_time > now)
	query := "SELECT id, rule_name, labels, start_time, end_time, created_by, comment FROM silences WHERE end_time > ?"
	rows, err := s.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var silences []*types.Silence
	for rows.Next() {
		var silence types.Silence
		var labelsJSON string

		if err := rows.Scan(&silence.ID, &silence.RuleName, &labelsJSON, &silence.StartTime, &silence.EndTime, &silence.CreatedBy, &silence.Comment); err != nil {
			return nil, err
		}

		_ = json.Unmarshal([]byte(labelsJSON), &silence.Labels)
		silences = append(silences, &silence)
	}

	return silences, nil
}

func (s *SQLiteSilenceStore) Close() error {
	return s.db.Close()
}
