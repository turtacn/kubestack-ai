package nlp

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	ncontext "github.com/kubestack-ai/kubestack-ai/internal/nlp/context"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/intent"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/tokenizer"
)

// Config represents NLP configuration.
type Config struct {
	TokenizerType string // "simple", "jieba"
	EnableLLM     bool
	MaxTurns      int
	SessionTTL    time.Duration
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		TokenizerType: "simple",
		EnableLLM:     false,
		MaxTurns:      10,
		SessionTTL:    30 * time.Minute,
	}
}

// NLPProcessor orchestrates NLP tasks.
type NLPProcessor struct {
	tokenizer        tokenizer.Tokenizer
	intentRecognizer intent.Recognizer
	entityExtractor  entity.Extractor
	contextManager   ncontext.Manager
	log              logger.Logger
}

// ProcessRequest represents an input request for processing.
type ProcessRequest struct {
	Text      string
	SessionID string
	UserID    string
}

// ProcessResult represents the outcome of NLP processing.
type ProcessResult struct {
	Intent        *intent.Intent
	Entities      []entity.Entity
	Context       *ncontext.ConversationContext
	Tokens        []string
	ProcessedText string
}

// NewNLPProcessor creates a new NLP processor.
func NewNLPProcessor(cfg *Config, llmClient interfaces.LLMClient) *NLPProcessor {
	// 1. Tokenizer
	var tok tokenizer.Tokenizer
	if cfg.TokenizerType == "simple" {
		tok = tokenizer.NewSimpleTokenizer(nil)
	} else {
		tok = tokenizer.NewSimpleTokenizer(nil)
	}

	// 2. Entity Extractor
	entExtractor := entity.BuildDefaultExtractor()

	// 3. Intent Recognizer
	ruleRecognizer := intent.NewRuleBasedRecognizer()
	var finalRecognizer intent.Recognizer = ruleRecognizer

	if cfg.EnableLLM && llmClient != nil {
		llmRecognizer := intent.NewLLMIntentRecognizer(llmClient, ruleRecognizer)
		// Hybrid: Rule first, fallback to LLM if needed (implemented in HybridRecognizer)
		// Or LLM first? Usually Hybrid means Rule first for speed/cost, LLM for coverage.
		// My HybridRecognizer impl logic: Rule -> if low conf -> LLM.
		finalRecognizer = intent.NewHybridRecognizer(ruleRecognizer, llmRecognizer, 0.7) // 0.7 threshold
	}

	// 4. Context Manager
	// For now, defaulting to InMemory.
	// Future: use Redis based on config.
	ctxManager := ncontext.NewInMemoryContextManager(cfg.MaxTurns, cfg.SessionTTL)

	return &NLPProcessor{
		tokenizer:        tok,
		intentRecognizer: finalRecognizer,
		entityExtractor:  entExtractor,
		contextManager:   ctxManager,
	}
}

// Process processes the user input.
func (p *NLPProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	result := &ProcessResult{}

	// 1. Preprocessing
	processedText := p.preprocess(req.Text)
	result.ProcessedText = processedText

	// 2. Tokenization
	tokens, err := p.tokenizer.Tokenize(ctx, processedText)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}
	result.Tokens = tokens

	// 3. Entity Extraction
	entities, err := p.entityExtractor.Extract(ctx, processedText, tokens)
	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}
	result.Entities = entities

	// 4. Load Context
	convCtx, err := p.contextManager.GetContext(ctx, req.SessionID)
	if err != nil {
		// Log error, proceed with new/empty context if needed?
		// For now, error likely means storage failure, but we can continue statelessly.
		// But GetContext returns new context on miss usually.
	}

	// 5. Intent Recognition
	intentReq := &intent.RecognizeRequest{
		Text:     processedText,
		Tokens:   tokens,
		Entities: entities,
		History:  convCtx.RecentIntents(),
	}
	recIntent, err := p.intentRecognizer.Recognize(ctx, intentReq)
	if err != nil {
		return nil, fmt.Errorf("intent recognition failed: %w", err)
	}
	result.Intent = recIntent

	// 6. Update Context
	convCtx.AddTurn(&ncontext.Turn{
		Text:     req.Text,
		Intent:   recIntent,
		Entities: entities,
		Time:     time.Now(),
	})
	if err := p.contextManager.SaveContext(ctx, req.SessionID, convCtx); err != nil {
		// Log error
	}
	result.Context = convCtx

	return result, nil
}

func (p *NLPProcessor) preprocess(text string) string {
	return text
}
