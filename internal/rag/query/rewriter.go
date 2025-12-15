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
	// Improve tokenization: handle punctuation slightly better
	// For this phase, we'll just check if the query contains the typo directly
	// or stick to fields but clean punctuation.

	// Simple whitespace tokenization
	tokens := strings.Fields(query)
	var rewrites []string

	for _, token := range tokens {
		// Clean punctuation for lookup
		cleanToken := strings.TrimRight(token, ".,?!:;")
		lowerToken := strings.ToLower(cleanToken)

		if fixed, ok := r.typoDict[lowerToken]; ok {
			// Preserve punctuation
			punctuation := token[len(cleanToken):]
			rewrites = append(rewrites, fixed+punctuation)
			continue
		}

		rewrites = append(rewrites, token)
	}

	return strings.Join(rewrites, " "), nil
}
