package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Custom error types for better error handling
type ParseError struct {
	Raw    string
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse AI output: %s. Raw output: %s", e.Reason, e.Raw)
}

type ValidationError struct {
	Details error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("AI output failed validation: %v", e.Details)
}

// --- Parser ---

type StructuredParser struct {
	validator *validator.Validate
}

func NewStructuredParser() *StructuredParser {
	return &StructuredParser{
		validator: validator.New(),
	}
}

func (p *StructuredParser) parseAndValidate(rawOutput string, target interface{}) error {
	jsonStr := extractJSONBlock(rawOutput)
	if jsonStr == "" {
		return &ParseError{Raw: rawOutput, Reason: "no JSON block found"}
	}

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return &ParseError{Raw: rawOutput, Reason: fmt.Sprintf("invalid JSON: %v", err)}
	}

	// Fuzzy repair for Severity field
	if result, ok := target.(*DiagnosisResult); ok {
		result.Severity = strings.Title(strings.ToLower(result.Severity))
	}

	if err := p.validator.Struct(target); err != nil {
		return &ValidationError{Details: err}
	}

	return nil
}

func (p *StructuredParser) ParseDiagnosisResult(rawOutput string) (*DiagnosisResult, error) {
	var result DiagnosisResult
	if err := p.parseAndValidate(rawOutput, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *StructuredParser) ParseRootCause(rawOutput string) (*RootCause, error) {
	var result RootCause
	if err := p.parseAndValidate(rawOutput, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *StructuredParser) ParseRepairPlan(rawOutput string) (*RepairPlan, error) {
	var result RepairPlan
	if err := p.parseAndValidate(rawOutput, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func extractJSONBlock(text string) string {
	re := regexp.MustCompile("(?s)```json\n(.*?)\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	// Fallback for raw JSON without markdown
	if strings.HasPrefix(strings.TrimSpace(text), "{") {
		return text
	}
	return ""
}
