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

// Package interfaces defines the contracts for LLM and RAG components.
package interfaces

import (
	"context"
)

// Message represents a single message in a conversation, following the common role-based structure.
type Message struct {
	Role    string `json:"role"` // e.g., "system", "user", "assistant"
	Content string `json:"content"`
}

// LLMRequest encapsulates all parameters for a request to an LLM.
type LLMRequest struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	Temperature  float32   `json:"temperature,omitempty"`
	MaxTokens    int       `json:"max_tokens,omitempty"`
	Stream       bool      `json:"stream,omitempty"` // Indicates if a streaming response is requested.
}

// UsageStats contains information about token usage for a request, useful for cost tracking.
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMResponse contains the complete response from a non-streaming LLM call.
type LLMResponse struct {
	Message Message    `json:"message"`
	Usage   UsageStats `json:"usage"`
}

// StreamingChunk represents a single piece of data from a streaming LLM response.
// The channel will be closed when the stream is complete.
type StreamingChunk struct {
	Content string `json:"content"`
	Err     error  // Used to propagate any errors that occur mid-stream.
}

// EmbeddingRequest encapsulates a request for generating text embeddings.
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model,omitempty"` // Optional: specify embedding model
}

// EmbeddingResponse contains the vector embeddings for the input text.
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Usage      UsageStats  `json:"usage"`
}

// LLMClient defines the standard interface for interacting with any Large Language Model provider.
// It abstracts away the specific details of different providers' APIs (e.g., OpenAI, Gemini),
// allowing the application to switch between them easily.
type LLMClient interface {
	// SendMessage sends a request to the LLM and gets a complete response back.
	// This is suitable for tasks where the full response is needed before proceeding.
	SendMessage(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// SendStreamingMessage sends a request and gets a stream of response chunks back via a channel.
	// This is useful for interactive, real-time applications like a CLI 'ask' command.
	SendStreamingMessage(ctx context.Context, req *LLMRequest) (<-chan StreamingChunk, error)

	// GenerateEmbedding converts one or more strings of text into numerical vector embeddings.
	// This is a core component of the RAG (Retrieval-Augmented Generation) system.
	GenerateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}

//Personal.AI order the ending
