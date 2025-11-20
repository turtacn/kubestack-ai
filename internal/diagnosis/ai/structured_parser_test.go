package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructuredParser_ParseDiagnosisResult(t *testing.T) {
	parser := NewStructuredParser()

	t.Run("Valid JSON with markdown", func(t *testing.T) {
		raw := "```json\n" +
			`{
			  "severity": "High",
			  "category": "Performance",
			  "root_cause": "The database is running slow due to a high number of concurrent connections.",
			  "affected_components": ["database", "api-server"],
			  "confidence": 0.85
			}` + "\n```"

		result, err := parser.ParseDiagnosisResult(raw)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "High", result.Severity)
		assert.Equal(t, "Performance", result.Category)
		assert.Equal(t, 0.85, result.Confidence)
	})

	t.Run("Valid raw JSON without markdown", func(t *testing.T) {
		raw := `{
			  "severity": "Critical",
			  "category": "Availability",
			  "root_cause": "The main server process has crashed unexpectedly.",
			  "affected_components": ["api-server"],
			  "confidence": 0.99
			}`

		result, err := parser.ParseDiagnosisResult(raw)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Critical", result.Severity)
	})

	t.Run("Invalid JSON syntax", func(t *testing.T) {
		raw := `{"severity": "High",` // Missing closing brace
		_, err := parser.ParseDiagnosisResult(raw)
		assert.Error(t, err)
		_, ok := err.(*ParseError)
		assert.True(t, ok, "Expected a *ParseError for invalid JSON")
	})

	t.Run("Validation Error on fields", func(t *testing.T) {
		raw := "```json\n" +
			`{
			  "severity": "Unknown",
			  "category": "Performance",
			  "root_cause": "short",
			  "affected_components": [],
			  "confidence": 1.1
			}` + "\n```"
		_, err := parser.ParseDiagnosisResult(raw)
		assert.Error(t, err)
		_, ok := err.(*ValidationError)
		assert.True(t, ok, "Expected a *ValidationError")
	})

	t.Run("No JSON block found", func(t *testing.T) {
		raw := `This is just some text from the AI without a json block.`
		_, err := parser.ParseDiagnosisResult(raw)
		assert.Error(t, err)
		_, ok := err.(*ParseError)
		assert.True(t, ok, "Expected a *ParseError for no JSON block")
	})

	t.Run("Fuzzy matching for severity", func(t *testing.T) {
		raw := `{
			  "severity": "critical",
			  "category": "Availability",
			  "root_cause": "The main server process has crashed unexpectedly.",
			  "affected_components": ["api-server"],
			  "confidence": 0.99
			}`
		result, err := parser.ParseDiagnosisResult(raw)
		assert.NoError(t, err)
		assert.Equal(t, "Critical", result.Severity, "Severity should be corrected to title case")
	})
}
