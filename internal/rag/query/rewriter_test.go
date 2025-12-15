package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryRewriter_Rewrite(t *testing.T) {
	typoDict := map[string]string{
		"reddis": "redis",
		"mysqll": "mysql",
	}
	rewriter := NewQueryRewriter(typoDict, nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"reddis is fast", "redis is fast"},
		{"mysqll connection", "mysql connection"},
		{"normal query", "normal query"},
	}

	for _, tt := range tests {
		res, err := rewriter.Rewrite(context.Background(), tt.input)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, res)
	}
}
