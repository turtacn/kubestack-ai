package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiTurnManager_ProcessTurn(t *testing.T) {
	manager := NewMultiTurnManager(3, 0.7)

	sessionID := "test-session"
	userInput := "My Redis is slow"

	t.Run("Clarification needed", func(t *testing.T) {
		aiResponse := &DiagnosisResult{
			Severity:           "High",
			Category:           "Performance",
			RootCause:          "Suspected high latency.",
			AffectedComponents: []string{"redis-master"},
			Confidence:         0.6,
		}

		turn, needsClarify, err := manager.ProcessTurn(sessionID, userInput, aiResponse)
		assert.NoError(t, err)
		assert.True(t, needsClarify)
		assert.Contains(t, turn.Content, "low confidence score")
	})

	t.Run("No clarification needed", func(t *testing.T) {
		aiResponse := &DiagnosisResult{
			Severity:           "Critical",
			Category:           "Resource",
			RootCause:          "OOMKilled due to memory pressure.",
			AffectedComponents: []string{"redis-master"},
			Confidence:         0.95,
		}

		_, needsClarify, err := manager.ProcessTurn(sessionID, userInput, aiResponse)
		assert.NoError(t, err)
		assert.False(t, needsClarify)
	})
}
