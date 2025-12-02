package parser

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// StructuredParser parses LLM responses into structured data.
type StructuredParser struct{}

// NewStructuredParser creates a new StructuredParser.
func NewStructuredParser() *StructuredParser {
	return &StructuredParser{}
}

// Parse parses the LLM response string into an AIAnalysisResult.
func (p *StructuredParser) Parse(input string) (*models.AIAnalysisResult, error) {
	// 1. Try direct JSON unmarshal
	var result models.AIAnalysisResult
	if err := json.Unmarshal([]byte(input), &result); err == nil {
		return &result, nil
	}

	// 2. Try to extract JSON from markdown code block
	re := regexp.MustCompile("(?s)```json\\s*(.*?)\\s*```")
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		jsonStr := matches[1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return &result, nil
		}
	}

	// 3. Fallback: Heuristic parsing (Simple)
	// For now, let's return an error if JSON parsing fails.
	return nil, fmt.Errorf("failed to parse AI response as JSON")
}
