// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ui provides components for rendering CLI user interfaces, like formatters and progress bars.
package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"gopkg.in/yaml.v2"
)

// Formatter defines the interface for components that format data for CLI output.
// It provides a standardized way to print structured data in various formats.
type Formatter interface {
	// Print takes an arbitrary data structure and prints it to standard output
	// in a specific format (e.g., JSON, YAML, or human-readable text).
	//
	// Parameters:
	//   data (interface{}): The data to be formatted and printed.
	//
	// Returns:
	//   error: An error if formatting or printing fails.
	Print(data interface{}) error
}

// NewFormatter is a factory function that returns the appropriate formatter
// based on the requested output format (e.g., from the --output flag).
// It defaults to the 'text' formatter if the requested format is not recognized.
//
// Parameters:
//   format (string): The desired output format (e.g., "json", "yaml", "text").
//
// Returns:
//   Formatter: An implementation of the Formatter interface.
func NewFormatter(format string) Formatter {
	switch strings.ToLower(format) {
	case "json":
		return &jsonFormatter{}
	case "yaml":
		return &yamlFormatter{}
	default:
		return &textFormatter{}
	}
}

// --- JSON Formatter ---
type jsonFormatter struct{}

// Print formats the provided data as a pretty-printed JSON string and writes it to standard output.
//
// Parameters:
//   data (interface{}): The data to be marshaled into JSON.
//
// Returns:
//   error: An error if JSON marshaling fails.
func (f *jsonFormatter) Print(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to json: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// --- YAML Formatter ---
type yamlFormatter struct{}

// Print formats the provided data as a YAML string and writes it to standard output.
//
// Parameters:
//   data (interface{}): The data to be marshaled into YAML.
//
// Returns:
//   error: An error if YAML marshaling fails.
func (f *yamlFormatter) Print(data interface{}) error {
	b, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to yaml: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// --- Human-Readable Text Formatter ---
type textFormatter struct{}

// Print formats the data into a human-readable, colorized text format.
// It uses a type switch to delegate the actual formatting to type-specific
// helper methods. If a type is not explicitly supported, it prints a warning
// and falls back to the JSON formatter.
//
// Parameters:
//   data (interface{}): The data to be formatted, typically a struct like *models.DiagnosisResult.
//
// Returns:
//   error: An error if printing fails.
func (f *textFormatter) Print(data interface{}) error {
	switch v := data.(type) {
	case *models.DiagnosisResult:
		return f.printDiagnosisResult(v)
	// TODO: Add cases for other data types, like *models.ExecutionResult
	default:
		// Fallback to JSON for unknown types
		color.Yellow("Warning: Text formatter not implemented for this data type. Falling back to JSON.")
		return NewFormatter("json").Print(data)
	}
}

// printDiagnosisResult formats a diagnosis result into a colorized, text-based view.
func (f *textFormatter) printDiagnosisResult(result *models.DiagnosisResult) error {
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	var statusColored string
	switch result.Status {
	case enum.StatusCritical:
		statusColored = red(result.Status.String())
	case enum.StatusWarning:
		statusColored = yellow(result.Status.String())
	default:
		statusColored = green(result.Status.String())
	}

	fmt.Printf("%s: %s (%s)\n", bold("Diagnosis Report"), result.Summary, statusColored)

	if len(result.Issues) == 0 {
		fmt.Println(green("\n✅ No issues found."))
		return nil
	}

	fmt.Printf("\nFound %d issues:\n", len(result.Issues))

	for _, issue := range result.Issues {
		var sevColored string
		switch issue.Severity {
		case enum.SeverityCritical:
			sevColored = red(issue.Severity.String())
		case enum.SeverityHigh:
			sevColored = yellow(issue.Severity.String())
		case enum.SeverityWarning:
			sevColored = yellow(issue.Severity.String())
		default:
			sevColored = color.New(color.FgWhite).SprintFunc()(issue.Severity.String())
		}

		fmt.Printf("\n[%s] %s\n", sevColored, bold(issue.Title))
		fmt.Printf("  - Description: %s\n", issue.Description)
		fmt.Printf("  - Evidence:    %s\n", issue.Evidence)
		for _, r := range issue.Recommendations {
			fmt.Printf("  - Recommendation: %s\n", r.Description)
		}
	}

	return nil
}

//Personal.AI order the ending
