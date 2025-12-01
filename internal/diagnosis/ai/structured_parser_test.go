package ai

import (
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/stretchr/testify/assert"
)

func TestStructuredParser_ParseDiagnosisResult(t *testing.T) {
	p := NewStructuredParser()

	t.Run("Valid JSON with markdown", func(t *testing.T) {
		// Use lowercase severity to match oneof validator
		raw := "```json\n" +
			`{
			  "severity": "high",
			  "category": "Performance",
			  "root_cause": "The database is running slow due to a high number of concurrent connections.",
			  "affected_components": ["database", "api-server"],
			  "confidence": 0.85
			}` + "\n```"

		var result parser.DiagnosisResult
		err := p.Parse(raw, &result)
		assert.NoError(t, err)
		assert.Equal(t, "high", result.Severity)
		assert.Equal(t, 0.85, result.Confidence)
	})

	t.Run("Valid raw JSON without markdown", func(t *testing.T) {
		raw := `{
			  "severity": "critical",
			  "category": "Availability",
			  "root_cause": "The main server process has crashed unexpectedly.",
			  "affected_components": ["api-server"],
			  "confidence": 0.99
			}`

		var result parser.DiagnosisResult
		err := p.Parse(raw, &result)
		assert.NoError(t, err)
		assert.Equal(t, "critical", result.Severity)
	})

	t.Run("Invalid JSON syntax", func(t *testing.T) {
		raw := `{"severity": "high",` // Missing closing brace
		var result parser.DiagnosisResult
		err := p.Parse(raw, &result)
		assert.Error(t, err)
	})
}
