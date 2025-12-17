package memory

import (
	"os"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/memory/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortTermMemory_Persist(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "short-term-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	badgerStore, err := store.NewBadgerStore(tempDir)
	require.NoError(t, err)

	stm := NewShortTermMemory(badgerStore, 24*time.Hour)
	sessionID := "test-session"

	entries := []MemoryEntry{
		{ID: "1", SessionID: sessionID, Role: "user", Content: "Hello", Timestamp: time.Now()},
		{ID: "2", SessionID: sessionID, Role: "assistant", Content: "Hi", Timestamp: time.Now()},
		{ID: "3", SessionID: sessionID, Role: "user", Content: "Bye", Timestamp: time.Now()},
	}

	err = stm.Save(sessionID, entries)
	assert.NoError(t, err)

	err = badgerStore.Close()
	assert.NoError(t, err)

	badgerStore2, err := store.NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer badgerStore2.Close()

	stm2 := NewShortTermMemory(badgerStore2, 24*time.Hour)
	loaded, err := stm2.Load(sessionID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(loaded))
	assert.Equal(t, entries[0].ID, loaded[0].ID)
	assert.Equal(t, entries[1].ID, loaded[1].ID)
	assert.Equal(t, entries[2].ID, loaded[2].ID)
}

func TestShortTermMemory_TTL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "short-term-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	badgerStore, err := store.NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer badgerStore.Close()

	stm := NewShortTermMemory(badgerStore, 2*time.Second)
	sessionID := "ttl-session"

	entries := []MemoryEntry{
		{ID: "1", SessionID: sessionID, Role: "user", Content: "Test", Timestamp: time.Now()},
	}

	err = stm.Save(sessionID, entries)
	assert.NoError(t, err)

	loaded, err := stm.Load(sessionID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(loaded))

	time.Sleep(3 * time.Second)

	_, err = stm.Load(sessionID)
	assert.Error(t, err)
}

func TestShortTermMemory_SessionIsolation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "short-term-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	badgerStore, err := store.NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer badgerStore.Close()

	stm := NewShortTermMemory(badgerStore, 24*time.Hour)

	session1 := "session-1"
	entries1 := []MemoryEntry{
		{ID: "1", SessionID: session1, Role: "user", Content: "Session 1", Timestamp: time.Now()},
	}

	session2 := "session-2"
	entries2 := []MemoryEntry{
		{ID: "2", SessionID: session2, Role: "user", Content: "Session 2", Timestamp: time.Now()},
	}

	err = stm.Save(session1, entries1)
	assert.NoError(t, err)

	err = stm.Save(session2, entries2)
	assert.NoError(t, err)

	loaded1, err := stm.Load(session1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(loaded1))
	assert.Equal(t, "Session 1", loaded1[0].Content)

	loaded2, err := stm.Load(session2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(loaded2))
	assert.Equal(t, "Session 2", loaded2[0].Content)
}

func TestShortTermMemory_Append(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "short-term-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	badgerStore, err := store.NewBadgerStore(tempDir)
	require.NoError(t, err)
	defer badgerStore.Close()

	stm := NewShortTermMemory(badgerStore, 24*time.Hour)
	sessionID := "append-session"

	entry1 := MemoryEntry{ID: "1", SessionID: sessionID, Role: "user", Content: "First", Timestamp: time.Now()}
	err = stm.Append(sessionID, entry1)
	assert.NoError(t, err)

	entry2 := MemoryEntry{ID: "2", SessionID: sessionID, Role: "assistant", Content: "Second", Timestamp: time.Now()}
	err = stm.Append(sessionID, entry2)
	assert.NoError(t, err)

	loaded, err := stm.Load(sessionID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(loaded))
	assert.Equal(t, "First", loaded[0].Content)
	assert.Equal(t, "Second", loaded[1].Content)
}
