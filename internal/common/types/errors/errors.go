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

// Package errors defines the structured error handling mechanism for KubeStack-AI.
// It provides a base error interface, specific error types for different domains,
// and helper functions for error wrapping, checking, and classification.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType defines the category of an error, allowing for high-level classification.
type ErrorType string

const (
	// PluginErrorType indicates an error related to the plugin system (e.g., loading, execution).
	PluginErrorType ErrorType = "PluginError"
	// DiagnosisErrorType indicates an error during the diagnosis process.
	DiagnosisErrorType ErrorType = "DiagnosisError"
	// ExecutionErrorType indicates an error during the execution of a fix or action plan.
	ExecutionErrorType ErrorType = "ExecutionError"
	// ConfigErrorType indicates an error related to loading, parsing, or validating configuration.
	ConfigErrorType ErrorType = "ConfigError"
	// LLMErrorType indicates an error when interacting with a Large Language Model provider.
	LLMErrorType ErrorType = "LLMError"
	// KnowledgeErrorType indicates an error related to the knowledge base (e.g., storage, retrieval).
	KnowledgeErrorType ErrorType = "KnowledgeError"
	// UnknownErrorType indicates an error that has not been classified.
	UnknownErrorType ErrorType = "UnknownError"
)

// KubeStackError is the base interface for all custom errors in the project.
// It extends the standard `error` interface with structured information like
// an error code, type, and recovery suggestions, enabling more robust error handling
// and user-friendly feedback.
type KubeStackError interface {
	error
	// Code returns the unique numerical code for the error.
	Code() int
	// Type returns the category of the error.
	Type() ErrorType
	// Message returns the core, user-facing error message.
	Message() string
	// Suggestion provides a hint or action for the user to resolve the error.
	Suggestion() string
	// Unwrap returns the underlying wrapped error, satisfying the Go 1.13 error wrapping interface.
	Unwrap() error
}

// baseError is the concrete implementation of the KubeStackError interface.
type baseError struct {
	code       int
	errorType  ErrorType
	message    string
	suggestion string
	cause      error
}

// New creates a new KubeStackError.
func newError(code int, errorType ErrorType, message, suggestion string) KubeStackError {
	return &baseError{
		code:       code,
		errorType:  errorType,
		message:    message,
		suggestion: suggestion,
	}
}

// Wrap creates a new KubeStackError that wraps an existing error.
func wrapError(err error, code int, errorType ErrorType, message, suggestion string) KubeStackError {
	return &baseError{
		code:       code,
		errorType:  errorType,
		message:    message,
		suggestion: suggestion,
		cause:      err,
	}
}

// Error returns a user-friendly, formatted string representation of the error.
// It includes the error type, code, and message, and will also include the
// suggestion and the underlying cause if they exist. This method satisfies the
// standard `error` interface.
//
// Returns:
//   string: The formatted error string.
func (e *baseError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s:%d] %s", e.errorType, e.code, e.message))
	if e.suggestion != "" {
		sb.WriteString(fmt.Sprintf(". Suggestion: %s", e.suggestion))
	}
	if e.cause != nil {
		sb.WriteString(fmt.Sprintf("\n  └─ Caused by: %s", e.cause.Error()))
	}
	return sb.String()
}

// Code returns the numerical code of the error, which can be used for programmatic error handling.
func (e *baseError) Code() int { return e.code }

// Type returns the category of the error as an ErrorType constant.
func (e *baseError) Type() ErrorType { return e.errorType }

// Message returns the short, user-facing description of the error.
func (e *baseError) Message() string { return e.message }

// Suggestion returns a helpful, actionable suggestion for the user to resolve the error.
func (e *baseError) Suggestion() string { return e.suggestion }

// Unwrap provides access to the underlying error that was wrapped by this KubeStackError.
// It allows for compatibility with `errors.Is` and `errors.As`.
func (e *baseError) Unwrap() error { return e.cause }

// --- Error Definitions ---

// PluginError definitions include codes for issues like a plugin not being found,
// failing to load, or an action within a plugin failing.
const (
	// PluginNotFoundCode indicates that a requested plugin could not be found in the configured directory.
	PluginNotFoundCode = 1001
	// PluginLoadFailedCode indicates a failure while trying to load a plugin file (e.g., a .so file).
	PluginLoadFailedCode = 1002
	// PluginUnloadFailedCode indicates a failure during the process of unloading a plugin.
	PluginUnloadFailedCode = 1003
	// PluginInvalidCode indicates that a loaded plugin is invalid (e.g., missing required symbols).
	PluginInvalidCode = 1004
	// PluginActionFailedCode indicates that an action invoked within a plugin has failed.
	PluginActionFailedCode = 1005
)

// NewPluginError creates a new KubeStackError with the PluginErrorType.
func NewPluginError(code int, msg, sug string) KubeStackError {
	return newError(code, PluginErrorType, msg, sug)
}

// WrapPluginError wraps an existing error in a new KubeStackError with the PluginErrorType.
func WrapPluginError(e error, c int, msg, sug string) KubeStackError {
	return wrapError(e, c, PluginErrorType, msg, sug)
}

// DiagnosisError definitions cover failures during the data collection, analysis,
// or report generation phases of a diagnosis.
const (
	// DiagnosisFailedCode is a general code for when a diagnosis fails for an unspecified reason.
	DiagnosisFailedCode = 2001
	// DataCollectionErrorCode indicates that an error occurred while collecting data for a diagnosis.
	DataCollectionErrorCode = 2002
	// AnalysisErrorCode indicates an error during the analysis of collected data.
	AnalysisErrorCode = 2003
	// ReportGenerationErrorCode indicates a failure while generating the final diagnosis report.
	ReportGenerationErrorCode = 2004
)

// NewDiagnosisError creates a new KubeStackError with the DiagnosisErrorType.
func NewDiagnosisError(c int, msg, sug string) KubeStackError {
	return newError(c, DiagnosisErrorType, msg, sug)
}

// WrapDiagnosisError wraps an existing error in a new KubeStackError with the DiagnosisErrorType.
func WrapDiagnosisError(e error, c int, msg, sug string) KubeStackError {
	return wrapError(e, c, DiagnosisErrorType, msg, sug)
}

// ExecutionError definitions relate to failures in planning or executing automated fixes.
const (
	// ExecutionPlanFailedCode indicates an error while generating a fix execution plan.
	ExecutionPlanFailedCode = 3001
	// ActionExecutionFailedCode indicates that a specific action within an execution plan failed.
	ActionExecutionFailedCode = 3002
	// RollbackFailedCode indicates that an error occurred while trying to roll back a failed action.
	RollbackFailedCode = 3003
	// ValidationFailedCode indicates that the post-execution validation check failed.
	ValidationFailedCode = 3004
)

// NewExecutionError creates a new KubeStackError with the ExecutionErrorType.
func NewExecutionError(c int, msg, sug string) KubeStackError {
	return newError(c, ExecutionErrorType, msg, sug)
}

// WrapExecutionError wraps an existing error in a new KubeStackError with the ExecutionErrorType.
func WrapExecutionError(e error, c int, msg, sug string) KubeStackError {
	return wrapError(e, c, ExecutionErrorType, msg, sug)
}

// ConfigError definitions are for errors related to application configuration.
const (
	// ConfigLoadFailedCode indicates an error while reading or parsing the configuration file.
	ConfigLoadFailedCode = 4001
	// ConfigValidationFailedCode indicates that the loaded configuration failed a validation rule.
	ConfigValidationFailedCode = 4002
	// ConfigSaveFailedCode indicates an error while trying to save the configuration.
	ConfigSaveFailedCode = 4003
	// ConfigNotFoundCode indicates that a required configuration file was not found.
	ConfigNotFoundCode = 4004
)

// NewConfigError creates a new KubeStackError with the ConfigErrorType.
func NewConfigError(c int, msg, sug string) KubeStackError {
	return newError(c, ConfigErrorType, msg, sug)
}

// WrapConfigError wraps an existing error in a new KubeStackError with the ConfigErrorType.
func WrapConfigError(e error, c int, msg, sug string) KubeStackError {
	return wrapError(e, c, ConfigErrorType, msg, sug)
}

// LLMError definitions cover issues with Large Language Model interactions.
const (
	// LLMRequestFailedCode indicates a failure in the network request to the LLM provider.
	LLMRequestFailedCode = 5001
	// LLMResponseParseFailedCode indicates an error while parsing the response from the LLM.
	LLMResponseParseFailedCode = 5002
	// LLMApiKeyInvalidCode indicates that the provided API key is invalid or unauthorized.
	LLMApiKeyInvalidCode = 5003
	// LLMQuotaExceededCode indicates that the API quota for the LLM provider has been exceeded.
	LLMQuotaExceededCode = 5004
)

// NewLLMError creates a new KubeStackError with the LLMErrorType.
func NewLLMError(c int, msg, sug string) KubeStackError {
	return newError(c, LLMErrorType, msg, sug)
}

// WrapLLMError wraps an existing error in a new KubeStackError with the LLMErrorType.
func WrapLLMError(e error, c int, msg, sug string) KubeStackError {
	return wrapError(e, c, LLMErrorType, msg, sug)
}

// KnowledgeError definitions are for errors related to the knowledge base.
const (
	// KnowledgeStoreFailedCode indicates an error while storing data in the knowledge base.
	KnowledgeStoreFailedCode = 6001
	// KnowledgeRetrievalFailedCode indicates an error while retrieving data from the knowledge base.
	KnowledgeRetrievalFailedCode = 6002
	// KnowledgeEmbeddingFailedCode indicates an error during the text embedding process.
	KnowledgeEmbeddingFailedCode = 6003
	// KnowledgeCrawlFailedCode indicates an error while crawling a data source for the knowledge base.
	KnowledgeCrawlFailedCode = 6004
)

// NewKnowledgeError creates a new KubeStackError with the KnowledgeErrorType.
func NewKnowledgeError(c int, msg, sug string) KubeStackError {
	return newError(c, KnowledgeErrorType, msg, sug)
}

// WrapKnowledgeError wraps an existing error in a new KubeStackError with the KnowledgeErrorType.
func WrapKnowledgeError(e error, c int, msg, sug string) KubeStackError {
	return wrapError(e, c, KnowledgeErrorType, msg, sug)
}


// --- Helper Functions ---

// IsKubeStackError checks if an error is of the KubeStackError type.
// This is a type assertion helper that improves readability.
//
// Parameters:
//   err (error): The error to check.
//
// Returns:
//   KubeStackError: The error cast to KubeStackError if the assertion is successful.
//   bool: True if the error is a KubeStackError, false otherwise.
func IsKubeStackError(err error) (KubeStackError, bool) {
	ke, ok := err.(KubeStackError)
	return ke, ok
}

// GetType classifies an error and returns its ErrorType.
// If the error is a KubeStackError, its specific type is returned. Otherwise,
// it returns UnknownErrorType. This is useful for high-level error routing.
//
// Parameters:
//   err (error): The error to classify.
//
// Returns:
//   ErrorType: The specific type of the error, or UnknownErrorType.
func GetType(err error) ErrorType {
	if ke, ok := IsKubeStackError(err); ok {
		return ke.Type()
	}
	return UnknownErrorType
}

//Personal.AI order the ending
