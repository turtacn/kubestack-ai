package query

import (
	"context"
	"strings"
)

type QueryRewriter struct {
	typoDict    map[string]string
	synonymDict map[string][]string
}

func NewQueryRewriter(typoDict map[string]string, synonymDict map[string][]string) *QueryRewriter {
	if typoDict == nil {
		typoDict = make(map[string]string)
	}
	if synonymDict == nil {
		synonymDict = make(map[string][]string)
	}
	return &QueryRewriter{
		typoDict:    typoDict,
		synonymDict: synonymDict,
	}
}

func (r *QueryRewriter) Rewrite(ctx context.Context, query string) (string, error) {
	// Simple whitespace tokenization (replace with better tokenizer if needed)
	tokens := strings.Fields(query)
	var rewrites []string

	for _, token := range tokens {
		// 1. Check Typo Dict
		lowerToken := strings.ToLower(token)
		if fixed, ok := r.typoDict[lowerToken]; ok {
			rewrites = append(rewrites, fixed)
			continue
		}

		// 2. Synonyms expansion?
		// For rewriting, we usually just replace typos.
		// Synonym expansion is more for query expansion (multiple queries).
		// But if we want to normalize terms, we can do it here.

		rewrites = append(rewrites, token)
	}

	return strings.Join(rewrites, " "), nil
}
