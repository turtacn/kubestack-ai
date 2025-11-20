package ai

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type MultiTurnManager struct {
	sessions      sync.Map // SessionID -> *ConversationSession
	maxTurns      int
	clarifyThresh float64
}

func NewMultiTurnManager(maxTurns int, clarifyThresh float64) *MultiTurnManager {
	return &MultiTurnManager{
		maxTurns:      maxTurns,
		clarifyThresh: clarifyThresh,
	}
}

type ConversationSession struct {
	ID         string
	History    []*Turn
	Context    map[string]interface{}
	LastActive time.Time
	mu         sync.Mutex
}

type Turn struct {
	Role     string
	Content  string
	Metadata map[string]interface{}
}

func (m *MultiTurnManager) ProcessTurn(sessionID string, userInput string, aiResponse *DiagnosisResult) (*Turn, bool, error) {
	session := m.getOrCreateSession(sessionID)
	session.mu.Lock()
	defer session.mu.Unlock()

	session.History = append(session.History, &Turn{Role: "user", Content: userInput})
	session.LastActive = time.Now()

	needsClarification := aiResponse.Confidence < m.clarifyThresh
	if needsClarification {
		clarifyPrompt, err := m.generateClarifyPrompt(aiResponse, session.Context)
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate clarification prompt: %w", err)
		}
		assistantTurn := &Turn{Role: "assistant", Content: clarifyPrompt, Metadata: map[string]interface{}{"type": "clarification_request"}}
		session.History = append(session.History, assistantTurn)
		m.compressHistory(session)
		return assistantTurn, true, nil
	}

	responseContent, err := json.Marshal(aiResponse)
	if err != nil {
		return nil, false, fmt.Errorf("failed to serialize AI response: %w", err)
	}
	assistantTurn := &Turn{Role: "assistant", Content: string(responseContent), Metadata: map[string]interface{}{"type": "diagnosis_result"}}
	session.History = append(session.History, assistantTurn)
	m.compressHistory(session)
	return assistantTurn, false, nil
}

func (m *MultiTurnManager) getOrCreateSession(sessionID string) *ConversationSession {
	val, ok := m.sessions.Load(sessionID)
	if ok {
		return val.(*ConversationSession)
	}
	session := &ConversationSession{
		ID:         sessionID,
		History:    []*Turn{},
		Context:    make(map[string]interface{}),
		LastActive: time.Now(),
	}
	m.sessions.Store(sessionID, session)
	return session
}

func (m *MultiTurnManager) generateClarifyPrompt(result *DiagnosisResult, context map[string]interface{}) (string, error) {
	var questions []string
	switch result.Category {
	case "Performance":
		questions = append(questions, "Can you provide the output of `redis-cli --latency-history`?")
		questions = append(questions, "Have there been any recent changes to client-side query patterns?")
	case "Resource":
		questions = append(questions, "What is the output of `kubectl top pod <pod-name>` for the affected instance?")
		questions = append(questions, "Are there any resource quotas or limits applied to the namespace?")
	default:
		questions = append(questions, fmt.Sprintf("Can you provide more logs from the time of the issue related to the root cause: '%s'?", result.RootCause))
	}

	prompt := fmt.Sprintf("The current diagnosis has a low confidence score of %.2f. To help me narrow down the issue, please provide the following information:\n", result.Confidence)
	for i, q := range questions {
		prompt += fmt.Sprintf("%d. %s\n", i+1, q)
	}
	return prompt, nil
}

func (m *MultiTurnManager) compressHistory(session *ConversationSession) {
	if len(session.History) > m.maxTurns*2 { // Each turn has a user and an assistant message
		// Keep the first turn (initial user prompt) and the last `maxTurns - 1` turns
		firstTurn := session.History[0]
		lastTurns := session.History[len(session.History)-(m.maxTurns-1)*2:]

		// Create a summary of the truncated turns
		summary := fmt.Sprintf("... (details of %d intermediate turns truncated) ...", (len(session.History) - len(lastTurns) - 1)/2)

		newHistory := []*Turn{firstTurn}
		newHistory = append(newHistory, &Turn{Role:"system", Content: summary})
		newHistory = append(newHistory, lastTurns...)

		session.History = newHistory
	}
}
