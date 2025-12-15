package context

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisSessionStore implements SessionStore using Redis.
type RedisSessionStore struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

// NewRedisSessionStore creates a new RedisSessionStore.
func NewRedisSessionStore(addr, password string, db int, prefix string, ttl time.Duration) *RedisSessionStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})

	if prefix == "" {
		prefix = "ksa:session"
	}

	return &RedisSessionStore{
		client:    rdb,
		keyPrefix: prefix,
		ttl:       ttl,
	}
}

func (s *RedisSessionStore) sessionKey(sessionID string) string {
	return fmt.Sprintf("%s:%s", s.keyPrefix, sessionID)
}

func (s *RedisSessionStore) Save(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := s.sessionKey(session.ID)
	// Use session TTL or default
	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisSessionStore) Load(ctx context.Context, sessionID string) (*Session, error) {
	key := s.sessionKey(sessionID)
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	key := s.sessionKey(sessionID)
	return s.client.Del(ctx, key).Err()
}

func (s *RedisSessionStore) Cleanup(ctx context.Context, olderThan time.Duration) (int, error) {
	// Redis handles expiration automatically via TTL.
	// We don't need manual cleanup for basic usage.
	return 0, nil
}
