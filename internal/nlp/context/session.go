package context

import (
	"context"
	"time"
)

// Session represents a user session.
type Session struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Context   *ConversationContext `json:"context"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// SessionStore interface for persisting sessions.
type SessionStore interface {
	Save(ctx context.Context, session *Session) error
	Load(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
	Cleanup(ctx context.Context, olderThan time.Duration) (int, error)
}
