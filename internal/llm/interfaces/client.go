// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package interfaces defines the contracts for LLM and RAG components.
package interfaces

import (
	"context"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMRequest encapsulates parameters for LLM request.
type LLMRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Temperature    float32   `json:"temperature,omitempty"`
	MaxTokens      int       `json:"max_tokens,omitempty"`
	Stream         bool      `json:"stream,omitempty"`
	ResponseFormat string    `json:"response_format,omitempty"`
}

// UsageStats contains token usage info.
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMResponse contains the response.
type LLMResponse struct {
	Message Message    `json:"message"`
	Usage   UsageStats `json:"usage"`
}

// StreamingChunk represents a streaming chunk.
type StreamingChunk struct {
	Content string `json:"content"`
	Err     error  `json:"-"`
}

// EmbeddingRequest for embeddings.
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model,omitempty"`
}

// EmbeddingResponse for embeddings.
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Usage      UsageStats  `json:"usage"`
}

// LLMOption defines functional options for LLM calls (Legacy support).
type LLMOption func(*LLMRequest)

// LLMClient defines the interface for interacting with LLM providers.
type LLMClient interface {
	SendMessage(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	SendStreamingMessage(ctx context.Context, req *LLMRequest) (<-chan StreamingChunk, error)
	GenerateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// Complete is kept for backward compatibility if other components rely on it,
	// but it should map to SendMessage internally.
	// However, if we remove it from interface, we must update all callers.
	// Since ai_analyzer uses SendMessage now, we can probably remove it,
	// BUT manager.go error suggested something expects Complete.
	// I will include it to satisfy any legacy constraints for now.
	Complete(ctx context.Context, prompt string, options ...LLMOption) (string, error)
}
