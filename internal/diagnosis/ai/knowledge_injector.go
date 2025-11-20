package ai

import (
    "fmt"
    "sort"
    "strings"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
)

type KnowledgeInjector struct {
    tokenCounter TokenCounter
    maxTokens    int
}

func NewKnowledgeInjector(maxTokens int) *KnowledgeInjector {
	return &KnowledgeInjector{
		tokenCounter: &ApproximateTokenCounter{},
		maxTokens:    maxTokens,
	}
}

func (k *KnowledgeInjector) InjectKnowledge(docs []search.Document, query string) string {
    sort.Slice(docs, func(i, j int) bool {
        return docs[i].Score > docs[j].Score
    })

    var injectedText strings.Builder
    currentTokens := 0
    for _, doc := range docs {
        docTokens := k.tokenCounter.Count(doc.Content)
        if currentTokens+docTokens > k.maxTokens {
            break
        }
        title := "Document"
        if t, ok := doc.Metadata["title"].(string); ok {
            title = t
        }
        injectedText.WriteString(fmt.Sprintf("### Document: %s\n", title))
        injectedText.WriteString(doc.Content)
        injectedText.WriteString("\n\n")
        currentTokens += docTokens
    }
    return injectedText.String()
}

type TokenCounter interface {
    Count(text string) int
}

type ApproximateTokenCounter struct{}

func (c *ApproximateTokenCounter) Count(text string) int {
    return len(text) / 4
}
