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

// Package parser provides helper functions for parsing and validating CLI arguments and flags.
// While Cobra and Viper handle most basic parsing, this package provides a centralized
// location for custom application-specific validation logic.
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// Parser provides centralized validation and parsing logic for CLI inputs.
// It is used to ensure that user-provided arguments and flags conform to
// application-specific rules that go beyond basic type checking.
type Parser struct {
	// This struct could hold configuration for validation rules in the future,
	// allowing for more dynamic validation behavior.
}

// NewParser creates and returns a new instance of the Parser.
// This constructor allows for a consistent way to initialize the parser.
//
// Returns:
//   *Parser: A pointer to a new Parser object.
func NewParser() *Parser {
	return &Parser{}
}

// ValidateMiddlewareType checks if the given string is a supported middleware type
// and returns the corresponding enum value. This centralizes the list of supported types.
//
// Parameters:
//   arg (string): The user-provided middleware type string (e.g., "redis", "mysql").
//
// Returns:
//   enum.MiddlewareType: The corresponding enum value if the type is valid.
//   error: An error if the provided string is not a supported middleware type.
func (p *Parser) ValidateMiddlewareType(arg string) (enum.MiddlewareType, error) {
	// A map-based approach is scalable and clear.
	supportedTypes := map[string]enum.MiddlewareType{
		"redis":         enum.Redis,
		"mysql":         enum.MySQL,
		"kafka":         enum.Kafka,
		"elasticsearch": enum.Elasticsearch,
		"postgresql":    enum.PostgreSQL,
		"mongodb":       enum.MongoDB,
		"rabbitmq":      enum.RabbitMQ,
		"minio":         enum.MinIO,
		"prometheus":    enum.Prometheus,
		"clickhouse":    enum.ClickHouse,
	}

	mwType, ok := supportedTypes[strings.ToLower(arg)]
	if !ok {
		return -1, fmt.Errorf("unsupported middleware type: '%s'", arg)
	}
	return mwType, nil
}

// ValidateOutputFormat checks if the given string is a supported output format.
//
// Parameters:
//   arg (string): The user-provided output format string (e.g., "json", "yaml").
//
// Returns:
//   error: An error if the format is not supported, otherwise nil.
func (p *Parser) ValidateOutputFormat(arg string) error {
	supported := map[string]bool{
		"text": true,
		"json": true,
		"yaml": true,
	}
	if !supported[strings.ToLower(arg)] {
		return fmt.Errorf("unsupported output format: '%s'. Supported formats are: text, json, yaml", arg)
	}
	return nil
}

// SanitizeInput performs basic sanitization on a user-provided string to remove
// characters that are often used in shell command injection attacks.
//
// NOTE: This provides a basic layer of defense and should not be the sole security measure.
// The primary security strategy should be to avoid executing shell commands from user input.
//
// Parameters:
//   input (string): The raw string to be sanitized.
//
// Returns:
//   string: The sanitized string with potentially harmful characters removed.
func (p *Parser) SanitizeInput(input string) string {
	// NOTE: This is a basic sanitizer and should not be solely relied upon for security.
	// The best practice is to avoid executing arbitrary strings and instead use
	// structured commands and parameters. This serves as a defense-in-depth measure.
	// This example removes common shell metacharacters: ; & | ( ) < > `
	re := regexp.MustCompile(`[;,&|()<>` + "`" + `]`)
	return re.ReplaceAllString(input, "")
}

// TODO: Implement functions for more complex validation logic, such as:
// - Validating mutually exclusive flags.
// - Checking for conditional dependencies between flags.
// - Parsing complex argument formats (e.g., "key=value,key2=value2").

//Personal.AI order the ending
