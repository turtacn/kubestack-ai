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

// Message represents a single message in a conversation, following the common
// role-based structure used by most chat-based LLMs.
type Message struct {
	// Role is the originator of the message (e.g., "system", "user", "assistant").
	Role string `json:"role"`
	// Content is the text of the message.
	Content string `json:"content"`
}

// LLMRequest encapsulates all the parameters for a request to an LLM's chat
// completion endpoint.
type LLMRequest struct {
	// Model is the identifier of the specific model to use for the request.
	Model string `json:"model"`
	// Messages is the sequence of messages representing the conversation history.
	Messages []Message `json:"messages"`
	// Temperature controls the randomness of the output. Higher values result in more creative responses.
	Temperature float32 `json:"temperature,omitempty"`
	// MaxTokens is the maximum number of tokens to generate in the response.
	MaxTokens int `json:"max_tokens,omitempty"`
	// Stream indicates whether a streaming response is requested.
	Stream bool `json:"stream,omitempty"`
	// ResponseFormat specifies the format of the response (e.g., "json_object").
	ResponseFormat string `json:"response_format,omitempty"`
}

// UsageStats contains information about token usage for an API request, which is
// crucial for monitoring costs and rate limits.
type UsageStats struct {
	// PromptTokens is the number of tokens in the input prompt.
	PromptTokens int `json:"prompt_tokens"`
	// CompletionTokens is the number of tokens in the generated response.
	CompletionTokens int `json:"completion_tokens"`
	// TotalTokens is the sum of prompt and completion tokens.
	TotalTokens int `json:"total_tokens"`
}

// LLMResponse contains the complete response from a non-streaming LLM call,
// including the assistant's message and token usage statistics.
type LLMResponse struct {
	// Message is the response message from the assistant.
	Message Message `json:"message"`
	// Usage provides the token usage statistics for the request.
	Usage UsageStats `json:"usage"`
}

// StreamingChunk represents a single piece of data received from a streaming LLM
// response. The channel transmitting these chunks will be closed when the stream is complete.
type StreamingChunk struct {
	// Content is the text content of the response chunk.
	Content string `json:"content"`
	// Err is used to propagate any errors that occur mid-stream.
	Err error `json:"-"`
}

// EmbeddingRequest encapsulates a request for generating vector embeddings from text.
type EmbeddingRequest struct {
	// Input is a slice of strings to be converted into embeddings.
	Input []string `json:"input"`
	// Model is an optional identifier for the specific embedding model to use.
	Model string `json:"model,omitempty"`
}

// EmbeddingResponse contains the vector embeddings generated for the input text,
// along with token usage statistics.
type EmbeddingResponse struct {
	// Embeddings is a slice of vectors, where each vector corresponds to an input string.
	Embeddings [][]float32 `json:"embeddings"`
	// Usage provides the token usage statistics for the embedding request.
	Usage UsageStats `json:"usage"`
}

// LLMClient defines the standard interface for interacting with any Large Language
// Model (LLM) provider. It abstracts away the specific details of different
// providers' APIs (e.g., OpenAI, Gemini), allowing the application's core logic
// to remain agnostic of the underlying LLM implementation.
type LLMClient interface {
	// SendMessage sends a request to the LLM and waits for a complete response.
	// This method is suitable for tasks where the full response is needed before proceeding.
	SendMessage(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// SendStreamingMessage sends a request and returns a channel from which response
	// chunks can be read in real-time. This is ideal for interactive applications,
	// such as a CLI, where immediate feedback is important.
	SendStreamingMessage(ctx context.Context, req *LLMRequest) (<-chan StreamingChunk, error)

	// GenerateEmbedding converts one or more strings of text into their numerical
	// vector representations. This is a core component of Retrieval-Augmented
	// Generation (RAG) systems, enabling semantic search.
	GenerateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}

//Personal.AI order the ending
