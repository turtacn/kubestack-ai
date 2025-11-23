package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// DiagnosisResult defines the structured output for diagnosis.
type DiagnosisResult struct {
	RootCause           string   `json:"root_cause" validate:"required"`
	Severity            string   `json:"severity" validate:"required,oneof=low medium high critical"`
	Confidence          float64  `json:"confidence" validate:"required,gte=0,lte=1"`
	ContributingFactors []string `json:"contributing_factors"`
	AffectedComponents  []string `json:"affected_components"`
	Evidence            []string `json:"evidence"`
	NextSteps           []string `json:"next_steps"`
}

// StructuredOutputParser handles parsing and validation of LLM JSON outputs.
type StructuredOutputParser struct {
	validator *validator.Validate
}

// NewStructuredOutputParser creates a new parser instance.
func NewStructuredOutputParser() *StructuredOutputParser {
	return &StructuredOutputParser{
		validator: validator.New(),
	}
}

// Parse parses the raw string output from LLM into a DiagnosisResult.
func (p *StructuredOutputParser) Parse(llmOutput string) (*DiagnosisResult, error) {
	cleanJSON := p.cleanOutput(llmOutput)

	var result DiagnosisResult
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := p.validator.Struct(&result); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &result, nil
}

// cleanOutput removes potential markdown code blocks from the output.
func (p *StructuredOutputParser) cleanOutput(output string) string {
	cleaned := strings.TrimSpace(output)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}
