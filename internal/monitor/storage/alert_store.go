package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
	_ "github.com/mattn/go-sqlite3"
)

// AlertQuery represents a query for alerts
type AlertQuery struct {
    Severity string
    Limit    string // Using string to match API params, but internally should probably be int
    Status   string
    Start    time.Time
    End      time.Time
}

// AlertStore defines the interface for storing alerts
type AlertStore interface {
    Save(ctx context.Context, alert *types.Alert) error
    Query(ctx context.Context, query *AlertQuery) ([]*types.Alert, error)
    Close() error
}

// SQLiteAlertStore implements AlertStore using SQLite
type SQLiteAlertStore struct {
    db *sql.DB
    mu sync.RWMutex
}

// NewSQLiteAlertStore creates a new SQLite alert store
func NewSQLiteAlertStore(path string) (*SQLiteAlertStore, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }

    query := `
    CREATE TABLE IF NOT EXISTS alerts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        rule_name TEXT NOT NULL,
        severity TEXT NOT NULL,
        status TEXT NOT NULL,
        labels TEXT,
        annotations TEXT,
        value REAL,
        fired_at DATETIME NOT NULL,
        resolved_at DATETIME
    );
    CREATE INDEX IF NOT EXISTS idx_alerts_fired_at ON alerts(fired_at);
    `
    if _, err := db.Exec(query); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to init alert db: %w", err)
    }

    return &SQLiteAlertStore{db: db}, nil
}

func (s *SQLiteAlertStore) Save(ctx context.Context, a *types.Alert) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    labelsJSON, _ := json.Marshal(a.Labels)
    annotationsJSON, _ := json.Marshal(a.Annotations)

    query := `INSERT INTO alerts (rule_name, severity, status, labels, annotations, value, fired_at, resolved_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

    _, err := s.db.ExecContext(ctx, query,
        a.RuleName, a.Severity, a.Status,
        string(labelsJSON), string(annotationsJSON), a.Value,
        a.FiredAt, a.ResolvedAt)
    return err
}

func (s *SQLiteAlertStore) Query(ctx context.Context, q *AlertQuery) ([]*types.Alert, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    queryParts := []string{"SELECT rule_name, severity, status, labels, annotations, value, fired_at, resolved_at FROM alerts WHERE 1=1"}
    var args []interface{}

    if q.Severity != "" {
        queryParts = append(queryParts, "AND severity = ?")
        args = append(args, q.Severity)
    }
    if q.Status != "" {
        queryParts = append(queryParts, "AND status = ?")
        args = append(args, q.Status)
    }
    if !q.Start.IsZero() {
        queryParts = append(queryParts, "AND fired_at >= ?")
        args = append(args, q.Start)
    }
    if !q.End.IsZero() {
        queryParts = append(queryParts, "AND fired_at <= ?")
        args = append(args, q.End)
    }

    queryParts = append(queryParts, "ORDER BY fired_at DESC")

    if q.Limit != "" {
        queryParts = append(queryParts, "LIMIT ?")
        args = append(args, q.Limit)
    }

    rows, err := s.db.QueryContext(ctx, strings.Join(queryParts, " "), args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var alerts []*types.Alert
    for rows.Next() {
        var a types.Alert
        var labelsJSON, annotationsJSON string
        var resolvedAt sql.NullTime

        if err := rows.Scan(&a.RuleName, &a.Severity, &a.Status, &labelsJSON, &annotationsJSON, &a.Value, &a.FiredAt, &resolvedAt); err != nil {
            return nil, err
        }

        if resolvedAt.Valid {
            a.ResolvedAt = resolvedAt.Time
        }
        _ = json.Unmarshal([]byte(labelsJSON), &a.Labels)
        _ = json.Unmarshal([]byte(annotationsJSON), &a.Annotations)

        alerts = append(alerts, &a)
    }

    return alerts, nil
}

func (s *SQLiteAlertStore) Close() error {
    return s.db.Close()
}
