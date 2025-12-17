package memory

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryManager_RecordAndRecall(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := MemoryConfig{
		WorkingWindowSize: 10,
		ShortTermTTL:      24 * time.Hour,
		StorePath:         tempDir,
	}

	manager, err := NewMemoryManager(cfg)
	require.NoError(t, err)
	defer manager.Close()

	sessionID := "test-session"

	entry1 := MemoryEntry{Role: "user", Content: "Hello"}
	err = manager.RecordMessage(sessionID, entry1)
	assert.NoError(t, err)

	entry2 := MemoryEntry{Role: "assistant", Content: "Hi there"}
	err = manager.RecordMessage(sessionID, entry2)
	assert.NoError(t, err)

	context, err := manager.GetContext(sessionID, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(context))
	assert.Equal(t, "Hello", context[0].Content)
	assert.Equal(t, "Hi there", context[1].Content)
}

func TestMemoryManager_ContextBuilding(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := MemoryConfig{
		WorkingWindowSize: 10,
		ShortTermTTL:      24 * time.Hour,
		StorePath:         tempDir,
	}

	manager, err := NewMemoryManager(cfg)
	require.NoError(t, err)
	defer manager.Close()

	sessionID := "context-session"

	for i := 0; i < 10; i++ {
		entry := MemoryEntry{
			Role:    "user",
			Content: "Message with approximately twenty tokens to test the context truncation mechanism properly",
		}
		err = manager.RecordMessage(sessionID, entry)
		assert.NoError(t, err)
	}

	context, err := manager.GetContext(sessionID, 100)
	assert.NoError(t, err)
	assert.True(t, len(context) < 10, "Context should be truncated based on token limit")
	assert.True(t, len(context) > 0, "Context should not be empty")
}

func TestMemoryManager_LoadSaveSession(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := MemoryConfig{
		WorkingWindowSize: 10,
		ShortTermTTL:      24 * time.Hour,
		StorePath:         tempDir,
	}

	manager, err := NewMemoryManager(cfg)
	require.NoError(t, err)

	sessionID := "load-save-session"

	entry1 := MemoryEntry{Role: "user", Content: "First message"}
	err = manager.RecordMessage(sessionID, entry1)
	assert.NoError(t, err)

	entry2 := MemoryEntry{Role: "assistant", Content: "First response"}
	err = manager.RecordMessage(sessionID, entry2)
	assert.NoError(t, err)

	err = manager.SaveSession(sessionID)
	assert.NoError(t, err)

	manager.ClearWorking()

	err = manager.LoadSession(sessionID)
	assert.NoError(t, err)

	context, err := manager.GetContext(sessionID, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(context))
	assert.Equal(t, "First message", context[0].Content)
	assert.Equal(t, "First response", context[1].Content)

	manager.Close()
}

func TestMemoryManager_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := MemoryConfig{
		WorkingWindowSize: 10,
		ShortTermTTL:      24 * time.Hour,
		StorePath:         tempDir,
	}

	sessionID := "persist-session"

	manager1, err := NewMemoryManager(cfg)
	require.NoError(t, err)

	entry := MemoryEntry{Role: "user", Content: "Persistent message"}
	err = manager1.RecordMessage(sessionID, entry)
	assert.NoError(t, err)

	manager1.Close()

	manager2, err := NewMemoryManager(cfg)
	require.NoError(t, err)
	defer manager2.Close()

	err = manager2.LoadSession(sessionID)
	assert.NoError(t, err)

	context, err := manager2.GetContext(sessionID, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(context))
	assert.Equal(t, "Persistent message", context[0].Content)
}

func TestMemoryManager_ClearWorking(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := MemoryConfig{
		WorkingWindowSize: 10,
		ShortTermTTL:      24 * time.Hour,
		StorePath:         tempDir,
	}

	manager, err := NewMemoryManager(cfg)
	require.NoError(t, err)
	defer manager.Close()

	sessionID := "clear-session"

	entry := MemoryEntry{Role: "user", Content: "Test"}
	err = manager.RecordMessage(sessionID, entry)
	assert.NoError(t, err)

	context, err := manager.GetContext(sessionID, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(context))

	manager.ClearWorking()

	// Note: GetContext falls back to short-term memory when working memory is empty
	// So we should still have the entry from short-term memory
	context, err = manager.GetContext(sessionID, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(context), "Context should be loaded from short-term memory")
}
