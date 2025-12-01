package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type StructuredParser struct {
	validator *validator.Validate
}

func NewStructuredParser() *StructuredParser {
	return &StructuredParser{
		validator: validator.New(),
	}
}

func (p *StructuredParser) Parse(output string, result interface{}) error {
	// 1. Clean up markdown code blocks if present
	cleanJSON := p.cleanMarkdown(output)

	// 2. Unmarshal JSON
	if err := json.Unmarshal([]byte(cleanJSON), result); err != nil {
		return fmt.Errorf("failed to parse JSON: %w\nOutput was: %s", err, output)
	}

	// 3. Validate structure
	if err := p.validator.Struct(result); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

func (p *StructuredParser) cleanMarkdown(output string) string {
	cleaned := strings.TrimSpace(output)
	if strings.HasPrefix(cleaned, "```") {
		// Remove first line (```json or just ```)
		lines := strings.Split(cleaned, "\n")
		if len(lines) > 1 {
			if strings.HasPrefix(lines[0], "```") {
				lines = lines[1:]
			}
			// Remove last line if it is ```
			if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			cleaned = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(cleaned)
}
