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

// ErrorType defines the category of an error.
type ErrorType string

const (
	PluginErrorType      ErrorType = "PluginError"
	DiagnosisErrorType   ErrorType = "DiagnosisError"
	ExecutionErrorType   ErrorType = "ExecutionError"
	ConfigErrorType      ErrorType = "ConfigError"
	LLMErrorType         ErrorType = "LLMError"
	KnowledgeErrorType   ErrorType = "KnowledgeError"
	UnknownErrorType     ErrorType = "UnknownError"
)

// KubeStackError is the base interface for all custom errors in the project.
// It extends the standard `error` interface with structured information like
// error code, type, and recovery suggestions.
type KubeStackError interface {
	error
	Code() int
	Type() ErrorType
	Message() string
	Suggestion() string
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

// Code returns the error code.
func (e *baseError) Code() int { return e.code }

// Type returns the error type.
func (e *baseError) Type() ErrorType { return e.errorType }

// Message returns the core error message.
func (e *baseError) Message() string { return e.message }

// Suggestion returns a user-friendly suggestion for resolving the error.
func (e *baseError) Suggestion() string { return e.suggestion }

// Unwrap returns the wrapped error, allowing for error chain inspection.
func (e *baseError) Unwrap() error { return e.cause }

// --- Error Definitions ---

// PluginError definitions
const (
	PluginNotFoundCode       = 1001
	PluginLoadFailedCode     = 1002
	PluginUnloadFailedCode   = 1003
	PluginInvalidCode        = 1004
	PluginActionFailedCode   = 1005
)
func NewPluginError(code int, msg, sug string) KubeStackError { return newError(code, PluginErrorType, msg, sug) }
func WrapPluginError(e error, c int, msg, sug string) KubeStackError { return wrapError(e, c, PluginErrorType, msg, sug) }

// DiagnosisError definitions
const (
	DiagnosisFailedCode       = 2001
	DataCollectionErrorCode   = 2002
	AnalysisErrorCode         = 2003
	ReportGenerationErrorCode = 2004
)
func NewDiagnosisError(c int, msg, sug string) KubeStackError { return newError(c, DiagnosisErrorType, msg, sug) }
func WrapDiagnosisError(e error, c int, msg, sug string) KubeStackError { return wrapError(e, c, DiagnosisErrorType, msg, sug) }

// ExecutionError definitions
const (
	ExecutionPlanFailedCode   = 3001
	ActionExecutionFailedCode = 3002
	RollbackFailedCode        = 3003
	ValidationFailedCode      = 3004
)
func NewExecutionError(c int, msg, sug string) KubeStackError { return newError(c, ExecutionErrorType, msg, sug) }
func WrapExecutionError(e error, c int, msg, sug string) KubeStackError { return wrapError(e, c, ExecutionErrorType, msg, sug) }

// ConfigError definitions
const (
	ConfigLoadFailedCode       = 4001
	ConfigValidationFailedCode = 4002
	ConfigSaveFailedCode       = 4003
	ConfigNotFoundCode         = 4004
)
func NewConfigError(c int, msg, sug string) KubeStackError { return newError(c, ConfigErrorType, msg, sug) }
func WrapConfigError(e error, c int, msg, sug string) KubeStackError { return wrapError(e, c, ConfigErrorType, msg, sug) }

// LLMError definitions
const (
	LLMRequestFailedCode       = 5001
	LLMResponseParseFailedCode = 5002
	LLMApiKeyInvalidCode       = 5003
	LLMQuotaExceededCode       = 5004
)
func NewLLMError(c int, msg, sug string) KubeStackError { return newError(c, LLMErrorType, msg, sug) }
func WrapLLMError(e error, c int, msg, sug string) KubeStackError { return wrapError(e, c, LLMErrorType, msg, sug) }

// KnowledgeError definitions
const (
	KnowledgeStoreFailedCode     = 6001
	KnowledgeRetrievalFailedCode = 6002
	KnowledgeEmbeddingFailedCode = 6003
	KnowledgeCrawlFailedCode     = 6004
)
func NewKnowledgeError(c int, msg, sug string) KubeStackError { return newError(c, KnowledgeErrorType, msg, sug) }
func WrapKnowledgeError(e error, c int, msg, sug string) KubeStackError { return wrapError(e, c, KnowledgeErrorType, msg, sug) }


// --- Helper Functions ---

// IsKubeStackError checks if an error is a KubeStackError and returns it.
func IsKubeStackError(err error) (KubeStackError, bool) {
	ke, ok := err.(KubeStackError)
	return ke, ok
}

// GetType classifies an error and returns its ErrorType.
// If the error is not a KubeStackError, it returns UnknownErrorType.
func GetType(err error) ErrorType {
	if ke, ok := IsKubeStackError(err); ok {
		return ke.Type()
	}
	return UnknownErrorType
}

//Personal.AI order the ending
