package store

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBadgerStore_CRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "badger-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store, err := NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer store.Close()

	key := "test-key"
	value := []byte("test-value")

	err = store.Set(key, value)
	assert.NoError(t, err)

	retrieved, err := store.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	err = store.Delete(key)
	assert.NoError(t, err)

	_, err = store.Get(key)
	assert.Error(t, err)
}

func TestBadgerStore_TTL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "badger-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store, err := NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer store.Close()

	key := "ttl-key"
	value := []byte("ttl-value")
	ttl := 2 * time.Second

	err = store.SetWithTTL(key, value, ttl)
	assert.NoError(t, err)

	retrieved, err := store.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	time.Sleep(3 * time.Second)

	_, err = store.Get(key)
	assert.Error(t, err)
}

func TestBadgerStore_Concurrent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "badger-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store, err := NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer store.Close()

	var wg sync.WaitGroup
	numOps := 100

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key-" + string(rune('0'+idx%10))
			value := []byte("value-" + string(rune('0'+idx%10)))
			
			err := store.Set(key, value)
			assert.NoError(t, err)
			
			_, err = store.Get(key)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestBadgerStore_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "badger-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	key := "persist-key"
	value := []byte("persist-value")

	store1, err := NewBadgerStore(tempDir)
	require.NoError(t, err)

	err = store1.Set(key, value)
	assert.NoError(t, err)

	err = store1.Close()
	assert.NoError(t, err)

	store2, err := NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer store2.Close()

	retrieved, err := store2.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)
}
