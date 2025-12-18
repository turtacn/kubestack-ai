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

// Package validator provides validation functions for CLI commands and parameters
package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/spf13/cobra"
)

// ValidateCommand validates that a cobra command has all required fields populated
func ValidateCommand(cmd *cobra.Command) error {
	if cmd.Use == "" {
		return fmt.Errorf("command must have Use field")
	}
	if cmd.Short == "" {
		return fmt.Errorf("command '%s' must have Short description", cmd.Use)
	}
	return nil
}

// ValidateCommandTree recursively validates all commands in a command tree
func ValidateCommandTree(cmd *cobra.Command) []error {
	var errors []error
	
	if err := ValidateCommand(cmd); err != nil {
		errors = append(errors, err)
	}
	
	// Validate flags
	cmd.Flags().VisitAll(func(flag *cobra.Flag) {
		if flag.Usage == "" {
			errors = append(errors, fmt.Errorf("flag '%s' in command '%s' must have usage text", flag.Name, cmd.Use))
		}
	})
	
	// Recursively validate subcommands
	for _, subCmd := range cmd.Commands() {
		if subErrs := ValidateCommandTree(subCmd); len(subErrs) > 0 {
			errors = append(errors, subErrs...)
		}
	}
	
	return errors
}

// ValidateMiddlewareType validates that a string is a valid middleware type
func ValidateMiddlewareType(middlewareType string) error {
	_, err := enum.ParseMiddlewareType(middlewareType)
	if err != nil {
		return fmt.Errorf("invalid middleware type '%s', must be one of: %s", 
			middlewareType, strings.Join(enum.AllowedMiddlewareTypes(), ", "))
	}
	return nil
}

// ValidateOutputFormat validates that the output format is supported
func ValidateOutputFormat(format string) error {
	validFormats := []string{"text", "json", "yaml", "table"}
	format = strings.ToLower(format)
	
	for _, valid := range validFormats {
		if format == valid {
			return nil
		}
	}
	
	return fmt.Errorf("invalid output format '%s', must be one of: %s", 
		format, strings.Join(validFormats, ", "))
}

// ValidateConnectionString validates the format of a connection string
func ValidateConnectionString(connStr string) error {
	if connStr == "" {
		return fmt.Errorf("connection string cannot be empty")
	}
	
	// Check if it's a URL format
	if strings.Contains(connStr, "://") {
		_, err := url.Parse(connStr)
		if err != nil {
			return fmt.Errorf("invalid connection string URL format: %w", err)
		}
		return nil
	}
	
	// Check if it's a host:port format
	hostPortRegex := regexp.MustCompile(`^[a-zA-Z0-9\.\-]+:\d+$`)
	if hostPortRegex.MatchString(connStr) {
		return nil
	}
	
	// Check if it's just a hostname/IP (port might be optional)
	hostRegex := regexp.MustCompile(`^[a-zA-Z0-9\.\-]+$`)
	if hostRegex.MatchString(connStr) {
		return nil
	}
	
	return fmt.Errorf("invalid connection string format, expected URL, host:port, or hostname")
}

// ValidateFlagsCompatibility validates that flag combinations are compatible
func ValidateFlagsCompatibility(flags map[string]interface{}) error {
	// Example: Check that if --dry-run is set, certain other flags might not make sense
	if dryRun, ok := flags["dry-run"].(bool); ok && dryRun {
		if autoFix, ok := flags["auto-fix"].(bool); ok && autoFix {
			return fmt.Errorf("--dry-run and --auto-fix cannot be used together")
		}
	}
	
	return nil
}

// ValidateInstanceName validates that an instance name follows naming conventions
func ValidateInstanceName(name string) error {
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}
	
	// Allow alphanumeric, hyphens, dots, and colons (for host:port)
	validNameRegex := regexp.MustCompile(`^[a-zA-Z0-9\.\-:]+$`)
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("invalid instance name '%s', must contain only alphanumeric characters, dots, hyphens, and colons", name)
	}
	
	return nil
}
