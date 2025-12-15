package tokenizer

import (
	"context"
	"regexp"
	"strings"
)

// Tokenizer is the interface for text tokenization.
type Tokenizer interface {
	Tokenize(ctx context.Context, text string) ([]string, error)
	TokenizeWithPos(ctx context.Context, text string) ([]Token, error)
}

// Token represents a token with position information.
type Token struct {
	Text     string
	StartPos int
	EndPos   int
	POS      string // Part of speech (optional)
}

// SimpleTokenizer is a basic tokenizer based on spaces and punctuation.
type SimpleTokenizer struct {
	stopwords map[string]bool
}

// NewSimpleTokenizer creates a new SimpleTokenizer.
func NewSimpleTokenizer(stopwords []string) *SimpleTokenizer {
	sw := make(map[string]bool)
	for _, s := range stopwords {
		sw[s] = true
	}
	// Add default stopwords if none provided
	if len(stopwords) == 0 {
		for s := range defaultStopwords {
			sw[s] = true
		}
	}
	return &SimpleTokenizer{
		stopwords: sw,
	}
}

func (t *SimpleTokenizer) Tokenize(ctx context.Context, text string) ([]string, error) {
	// Split by whitespace and punctuation
	separators := regexp.MustCompile(`[\s\p{P}]+`)
	rawTokens := separators.Split(text, -1)

	var tokens []string
	for _, token := range rawTokens {
		token = strings.TrimSpace(token)
		if token == "" || t.stopwords[token] {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (t *SimpleTokenizer) TokenizeWithPos(ctx context.Context, text string) ([]Token, error) {
	// Simple implementation doesn't support accurate position tracking easily with split
	// This is a placeholder for interface satisfaction if needed, or we can implement a scanner.
	// For now, we will just return tokens with dummy positions or implement a basic scanner.

	// Better implementation for position tracking
	var tokens []Token
	re := regexp.MustCompile(`[^\s\p{P}]+`)
	matches := re.FindAllStringIndex(text, -1)

	for _, loc := range matches {
		word := text[loc[0]:loc[1]]
		if !t.stopwords[word] {
			tokens = append(tokens, Token{
				Text:     word,
				StartPos: loc[0],
				EndPos:   loc[1],
			})
		}
	}
	return tokens, nil
}

// defaultStopwords contains a small set of common Chinese/English stopwords.
var defaultStopwords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true,
	"我": true, "有": true, "和": true, "就": true,
	"不": true, "人": true, "都": true, "一": true,
	"一个": true, "上": true, "也": true, "很": true,
	"到": true, "说": true, "要": true, "去": true,
	"你": true, "会": true, "着": true, "没有": true,
	"看": true, "好": true, "自己": true, "这": true,
    "the": true, "is": true, "in": true, "at": true, "of": true,
    "and": true, "a": true, "an": true, "to": true, "for": true,
}
