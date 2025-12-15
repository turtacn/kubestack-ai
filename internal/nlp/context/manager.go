package context

import (
	"context"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/intent"
)

// Manager manages conversation contexts.
type Manager interface {
	GetContext(ctx context.Context, sessionID string) (*ConversationContext, error)
	SaveContext(ctx context.Context, sessionID string, convCtx *ConversationContext) error
	ClearContext(ctx context.Context, sessionID string) error
}

// ConversationContext holds the state of a conversation.
type ConversationContext struct {
	SessionID    string
	UserID       string
	Turns        []*Turn                                     `json:"turns"`
	ActiveEntity map[entity.EntityType]entity.Entity         `json:"active_entities"`
	CreatedAt    time.Time                                   `json:"created_at"`
	UpdatedAt    time.Time                                   `json:"updated_at"`
	MaxTurns     int                                         `json:"max_turns"`
}

// Turn represents a single turn in the conversation.
type Turn struct {
	Text     string         `json:"text"`
	Intent   *intent.Intent `json:"intent"`
	Entities []entity.Entity `json:"entities"`
	Response string         `json:"response"`
	Time     time.Time      `json:"time"`
}

// AddTurn adds a new turn to the conversation.
func (c *ConversationContext) AddTurn(turn *Turn) {
	c.Turns = append(c.Turns, turn)

	// Keep max turns limit
	if c.MaxTurns > 0 && len(c.Turns) > c.MaxTurns {
		c.Turns = c.Turns[len(c.Turns)-c.MaxTurns:]
	}

	// Update active entities
	if c.ActiveEntity == nil {
		c.ActiveEntity = make(map[entity.EntityType]entity.Entity)
	}
	for _, e := range turn.Entities {
		c.ActiveEntity[e.Type] = e
	}

	c.UpdatedAt = time.Now()
}

// RecentIntents returns the most recent intents.
func (c *ConversationContext) RecentIntents() []*intent.Intent {
	intents := make([]*intent.Intent, 0, len(c.Turns))
	for _, t := range c.Turns {
		if t.Intent != nil {
			intents = append(intents, t.Intent)
		}
	}
	return intents
}

// GetActiveEntity gets the currently active entity of a given type.
func (c *ConversationContext) GetActiveEntity(entityType entity.EntityType) (entity.Entity, bool) {
	if c.ActiveEntity == nil {
		return entity.Entity{}, false
	}
	e, ok := c.ActiveEntity[entityType]
	return e, ok
}

// InMemoryContextManager is a simple in-memory implementation of Manager.
type InMemoryContextManager struct {
	contexts   map[string]*ConversationContext
	mu         sync.RWMutex
	maxTurns   int
	sessionTTL time.Duration
	store      SessionStore // Optional backing store
}

// NewInMemoryContextManager creates a new InMemoryContextManager.
func NewInMemoryContextManager(maxTurns int, sessionTTL time.Duration, store SessionStore) *InMemoryContextManager {
	return &InMemoryContextManager{
		contexts:   make(map[string]*ConversationContext),
		maxTurns:   maxTurns,
		sessionTTL: sessionTTL,
		store:      store,
	}
}

func (m *InMemoryContextManager) GetContext(ctx context.Context, sessionID string) (*ConversationContext, error) {
	m.mu.RLock()
	convCtx, ok := m.contexts[sessionID]
	m.mu.RUnlock()

	if ok {
		// Check expiry
		if time.Since(convCtx.UpdatedAt) > m.sessionTTL {
			m.ClearContext(ctx, sessionID) // Remove from memory
			// Try load from store if available?
			// If store exists, maybe it has fresh data?
		} else {
			return convCtx, nil
		}
	}

	// Try loading from persistent store
	if m.store != nil {
		session, err := m.store.Load(ctx, sessionID)
		if err == nil && session != nil && session.Context != nil {
			// Restore context
			m.mu.Lock()
			m.contexts[sessionID] = session.Context
			m.mu.Unlock()
			return session.Context, nil
		}
	}

	return m.newContext(sessionID), nil
}

func (m *InMemoryContextManager) SaveContext(ctx context.Context, sessionID string, convCtx *ConversationContext) error {
	m.mu.Lock()
	m.contexts[sessionID] = convCtx
	m.mu.Unlock()

	// Persist asynchronously or synchronously? Sync for safety.
	if m.store != nil {
		session := &Session{
			ID:        sessionID,
			UserID:    convCtx.UserID,
			Context:   convCtx,
			CreatedAt: convCtx.CreatedAt,
			UpdatedAt: convCtx.UpdatedAt,
		}
		return m.store.Save(ctx, session)
	}
	return nil
}

func (m *InMemoryContextManager) ClearContext(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	delete(m.contexts, sessionID)
	m.mu.Unlock()

	if m.store != nil {
		return m.store.Delete(ctx, sessionID)
	}
	return nil
}

func (m *InMemoryContextManager) newContext(sessionID string) *ConversationContext {
	return &ConversationContext{
		SessionID:    sessionID,
		Turns:        make([]*Turn, 0),
		ActiveEntity: make(map[entity.EntityType]entity.Entity),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		MaxTurns:     m.maxTurns,
	}
}
