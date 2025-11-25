package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/monitor/model"
	_ "github.com/mattn/go-sqlite3"
)

// Query represents a query for metrics
type Query struct {
	Metric string            // Metric name pattern
	Labels map[string]string // Label matchers
	Start  time.Time
	End    time.Time
}

// TimeseriesStore defines the interface for storing and querying metrics
type TimeseriesStore interface {
	Write(ctx context.Context, points []*model.MetricPoint) error
	Query(ctx context.Context, query *Query) ([]*model.MetricPoint, error)
	Close() error
}

// SQLiteTimeseriesStore implements TimeseriesStore using SQLite
type SQLiteTimeseriesStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteTimeseriesStore creates a new SQLite store
func NewSQLiteTimeseriesStore(path string) (*SQLiteTimeseriesStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	query := `
    CREATE TABLE IF NOT EXISTS metrics (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        value REAL NOT NULL,
        timestamp DATETIME NOT NULL,
        labels TEXT -- JSON encoded labels
    );
    CREATE INDEX IF NOT EXISTS idx_metrics_name_ts ON metrics(name, timestamp);
    `
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init db: %w", err)
	}

	return &SQLiteTimeseriesStore{db: db}, nil
}

// Write writes metrics to SQLite
func (s *SQLiteTimeseriesStore) Write(ctx context.Context, points []*model.MetricPoint) error {
	if len(points) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO metrics (name, value, timestamp, labels) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		labelsJSON, _ := json.Marshal(p.Labels)
		if _, err := stmt.ExecContext(ctx, p.Name, p.Value, p.Timestamp, string(labelsJSON)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Query queries metrics from SQLite
func (s *SQLiteTimeseriesStore) Query(ctx context.Context, q *Query) ([]*model.MetricPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryParts := []string{"SELECT name, value, timestamp, labels FROM metrics WHERE timestamp BETWEEN ? AND ?"}
	args := []interface{}{q.Start, q.End}

	// Pattern matching for metric name (support * wildcard)
	if q.Metric != "" {
		if strings.Contains(q.Metric, "*") {
			queryParts = append(queryParts, "AND name LIKE ?")
			args = append(args, strings.ReplaceAll(q.Metric, "*", "%"))
		} else {
			queryParts = append(queryParts, "AND name = ?")
			args = append(args, q.Metric)
		}
	}

	// NOTE: filtering by labels in SQLite efficiently is hard with JSON column.
	// For MVP, we will fetch and filter in Go if labels are provided.
	// Or we can assume labels are not heavily used for filtering in DB yet.

	rows, err := s.db.QueryContext(ctx, strings.Join(queryParts, " "), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.MetricPoint
	for rows.Next() {
		var name string
		var value float64
		var ts time.Time
		var labelsJSON string

		if err := rows.Scan(&name, &value, &ts, &labelsJSON); err != nil {
			return nil, err
		}

		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			continue // Skip malformed labels
		}

		// Client-side filtering for labels
		match := true
		for k, v := range q.Labels {
			if val, ok := labels[k]; !ok || val != v {
				match = false
				break
			}
		}

		if match {
			results = append(results, &model.MetricPoint{
				Name:      name,
				Value:     value,
				Timestamp: ts,
				Labels:    labels,
			})
		}
	}

	return results, nil
}

func (s *SQLiteTimeseriesStore) Close() error {
	return s.db.Close()
}
