package store

import "time"

// Store is the abstraction for key-value storage
type Store interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	SetWithTTL(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Close() error
}
