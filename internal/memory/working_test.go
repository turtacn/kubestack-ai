package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkingMemory_AddAndRetrieve(t *testing.T) {
	wm := NewWorkingMemory(10)

	entries := []MemoryEntry{
		{ID: "1", Role: "user", Content: "Hello", Timestamp: time.Now()},
		{ID: "2", Role: "assistant", Content: "Hi there", Timestamp: time.Now()},
		{ID: "3", Role: "user", Content: "How are you?", Timestamp: time.Now()},
	}

	for _, entry := range entries {
		err := wm.Add(entry)
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, wm.Size())

	retrieved := wm.GetAll()
	assert.Equal(t, 3, len(retrieved))
	assert.Equal(t, entries[0].ID, retrieved[0].ID)
	assert.Equal(t, entries[1].ID, retrieved[1].ID)
	assert.Equal(t, entries[2].ID, retrieved[2].ID)
}

func TestWorkingMemory_WindowLimit(t *testing.T) {
	maxSize := 3
	wm := NewWorkingMemory(maxSize)

	for i := 0; i < 5; i++ {
		entry := MemoryEntry{
			ID:        string(rune('A' + i)),
			Role:      "user",
			Content:   "Message " + string(rune('A'+i)),
			Timestamp: time.Now(),
		}
		err := wm.Add(entry)
		assert.NoError(t, err)
	}

	assert.Equal(t, maxSize, wm.Size())

	retrieved := wm.GetAll()
	assert.Equal(t, "C", retrieved[0].ID)
	assert.Equal(t, "D", retrieved[1].ID)
	assert.Equal(t, "E", retrieved[2].ID)
}

func TestWorkingMemory_Clear(t *testing.T) {
	wm := NewWorkingMemory(10)

	entry := MemoryEntry{ID: "1", Role: "user", Content: "Test", Timestamp: time.Now()}
	err := wm.Add(entry)
	assert.NoError(t, err)
	assert.Equal(t, 1, wm.Size())

	wm.Clear()
	assert.Equal(t, 0, wm.Size())

	retrieved := wm.GetAll()
	assert.Empty(t, retrieved)
}

func TestWorkingMemory_GetRecent(t *testing.T) {
	wm := NewWorkingMemory(10)

	for i := 0; i < 5; i++ {
		entry := MemoryEntry{
			ID:        string(rune('A' + i)),
			Role:      "user",
			Content:   "Message " + string(rune('A'+i)),
			Timestamp: time.Now(),
		}
		err := wm.Add(entry)
		assert.NoError(t, err)
	}

	recent := wm.GetRecent(3)
	assert.Equal(t, 3, len(recent))
	assert.Equal(t, "C", recent[0].ID)
	assert.Equal(t, "D", recent[1].ID)
	assert.Equal(t, "E", recent[2].ID)

	all := wm.GetRecent(10)
	assert.Equal(t, 5, len(all))

	none := wm.GetRecent(0)
	assert.Empty(t, none)
}
