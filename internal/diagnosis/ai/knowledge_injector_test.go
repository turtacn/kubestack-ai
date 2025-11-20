package ai

import (
	"testing"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/stretchr/testify/assert"
)

func TestKnowledgeInjector_InjectKnowledge(t *testing.T) {
	injector := NewKnowledgeInjector(30) // Max 30 tokens, should only fit the first two docs

	docs := []search.Document{
		{Content: "This is a high-priority document with lots of useful information.", Score: 0.9}, // ~16 tokens
		{Content: "This is a medium-priority document.", Score: 0.5}, // ~8 tokens
		{Content: "This is a low-priority document that should be truncated.", Score: 0.2}, // ~14 tokens
	}

	result := injector.InjectKnowledge(docs, "test query")

	assert.Contains(t, result, "This is a high-priority document")
	assert.Contains(t, result, "This is a medium-priority document")
	assert.NotContains(t, result, "This is a low-priority document that should be truncated.")
}
