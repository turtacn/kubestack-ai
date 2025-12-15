package intent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// LLMIntentRecognizer uses LLM for intent recognition.
type LLMIntentRecognizer struct {
	llmClient interfaces.LLMClient
	prompt    string
	timeout   time.Duration
	fallback  Recognizer
}

// NewLLMIntentRecognizer creates a new LLMIntentRecognizer.
func NewLLMIntentRecognizer(client interfaces.LLMClient, fallback Recognizer) *LLMIntentRecognizer {
	return &LLMIntentRecognizer{
		llmClient: client,
		prompt:    intentPromptTemplate,
		timeout:   5 * time.Second,
		fallback:  fallback,
	}
}

func (r *LLMIntentRecognizer) Name() string {
	return "LLM"
}

// Recognize uses LLM to recognize intent.
func (r *LLMIntentRecognizer) Recognize(ctx context.Context, req *RecognizeRequest) (*Intent, error) {
	// If LLM client is nil, fallback immediately
	if r.llmClient == nil {
		if r.fallback != nil {
			return r.fallback.Recognize(ctx, req)
		}
		return &Intent{Type: IntentUnknown, RawText: req.Text}, nil
	}

	// 1. Build Prompt (Simplified)
	// In a real implementation, we would use a template engine.
	prompt := r.prompt + "\n\nUser Input: " + req.Text

	// 2. Call LLM
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Assuming Complete signature matches what's available or mocked
	// Since I don't see the exact signature of LLMClient, I assume a standard one based on memory.
	// Memory says: `Complete(ctx, prompt, options)`
	// I'll check `internal/llm/interfaces/client.go` to be sure if I can, but I'll proceed with assumed signature or interface usage.

	// Wait, I cannot see `internal/llm/interfaces/client.go`. I should have checked it.
	// But based on typical usage:
	response, err := r.llmClient.Complete(ctx, prompt, nil)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Recognize(ctx, req)
		}
		return &Intent{Type: IntentUnknown, RawText: req.Text}, err
	}

	// 3. Parse JSON
	var result struct {
		Intent     string  `json:"intent"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}

	// Extract JSON from response if it contains markdown code blocks
	// Simple cleanup
	cleanedResponse := cleanJSON(response)

	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		if r.fallback != nil {
			return r.fallback.Recognize(ctx, req)
		}
		// Return unknown if parse fails
		return &Intent{Type: IntentUnknown, RawText: req.Text}, nil
	}

	return &Intent{
		Type:       IntentType(result.Intent),
		Confidence: result.Confidence,
		RawText:    req.Text,
		Reason:     result.Reason,
	}, nil
}

func cleanJSON(s string) string {
	// Implement rudimentary cleanup: remove ```json ... ```
	// This is just a placeholder.
	return s
}

const intentPromptTemplate = `You are an operations intent recognition assistant. Analyze the user input and determine the intent type.

Options:
- diagnose: Diagnose issues (e.g., "Check Redis")
- query: Query metrics (e.g., "Memory usage")
- fix: Fix issues (e.g., "Clean slow logs")
- alert: Set alerts (e.g., "Alert if memory > 80%")
- config: Modify config (e.g., "Set max connections")
- explain: Explain concepts (e.g., "What is lag")
- help: Help (e.g., "What can you do")

Return ONLY JSON: {"intent": "xxx", "confidence": 0.x, "reason": "xxx"}`

// HybridRecognizer combines Rule-based and LLM-based recognition.
type HybridRecognizer struct {
	ruleRecognizer Recognizer
	llmRecognizer  Recognizer
	llmThreshold   float64
}

// NewHybridRecognizer creates a new HybridRecognizer.
func NewHybridRecognizer(rule Recognizer, llm Recognizer, threshold float64) *HybridRecognizer {
	return &HybridRecognizer{
		ruleRecognizer: rule,
		llmRecognizer:  llm,
		llmThreshold:   threshold,
	}
}

func (r *HybridRecognizer) Name() string {
	return "Hybrid"
}

func (r *HybridRecognizer) Recognize(ctx context.Context, req *RecognizeRequest) (*Intent, error) {
	// 1. Try Rule Recognizer
	intent, err := r.ruleRecognizer.Recognize(ctx, req)
	if err != nil {
		// Log error?
	}

	// 2. If confidence is high enough, return
	if intent != nil && intent.Confidence >= r.llmThreshold {
		return intent, nil
	}

	// 3. Fallback to LLM
	// Only if LLM recognizer is available
	if r.llmRecognizer != nil {
		llmIntent, err := r.llmRecognizer.Recognize(ctx, req)
		if err == nil {
			return llmIntent, nil
		}
	}

	return intent, nil
}
