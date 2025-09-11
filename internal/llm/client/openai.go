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

// Package client provides concrete implementations of the LLMClient interface for various providers.
package client

import (
	"context"
	"errors"
	"io"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/sashabaranov/go-openai"
)

// openAIClient is the concrete implementation of the LLMClient for OpenAI's API.
// It wraps the 'go-openai' library to conform to the application's standard interface.
type openAIClient struct {
	client *openai.Client
	log    logger.Logger
}

// NewOpenAIClient creates a new client for interacting with OpenAI.
// It requires an API key and allows for an optional API base URL for proxies.
func NewOpenAIClient(apiKey string, apiBaseURL ...string) (interfaces.LLMClient, error) {
	if apiKey == "" {
		return nil, errors.New("OpenAI API key cannot be empty")
	}

	config := openai.DefaultConfig(apiKey)
	// Allow overriding the base URL for proxies or custom endpoints like local LLMs.
	if len(apiBaseURL) > 0 && apiBaseURL[0] != "" {
		config.BaseURL = apiBaseURL[0]
	}

	// The go-openai client has some default retry logic. For more advanced strategies,
	// a library like 'cenkalti/backoff' could be wrapped around these calls.

	return &openAIClient{
		client: openai.NewClientWithConfig(config),
		log:    logger.NewLogger("openai-client"),
	}, nil
}

// SendMessage sends a standard, non-streaming chat completion request.
func (c *openAIClient) SendMessage(ctx context.Context, req *interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	c.log.Debugf("Sending chat completion request to model %s", req.Model)
	resp, err := c.client.CreateChatCompletion(ctx, toOpenAIChatRequest(req))
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("openai response contained no choices")
	}

	return &interfaces.LLMResponse{
		Message: interfaces.Message{
			Role:    resp.Choices[0].Message.Role,
			Content: resp.Choices[0].Message.Content,
		},
		Usage: interfaces.UsageStats{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// SendStreamingMessage sends a request and returns a channel for streaming the response.
func (c *openAIClient) SendStreamingMessage(ctx context.Context, req *interfaces.LLMRequest) (<-chan interfaces.StreamingChunk, error) {
	c.log.Debugf("Sending streaming chat completion request to model %s", req.Model)
	req.Stream = true // Ensure stream is enabled in the request.

	stream, err := c.client.CreateChatCompletionStream(ctx, toOpenAIChatRequest(req))
	if err != nil {
		return nil, err
	}

	chunkChan := make(chan interfaces.StreamingChunk)

	// Start a goroutine to process the stream from OpenAI and send it to our channel.
	go func() {
		defer stream.Close()
		defer close(chunkChan)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				c.log.Debug("Stream finished.")
				return
			}
			if err != nil {
				c.log.Errorf("Error receiving stream chunk: %v", err)
				chunkChan <- interfaces.StreamingChunk{Err: err}
				return
			}

			chunk := interfaces.StreamingChunk{Content: response.Choices[0].Delta.Content}
			select {
			case chunkChan <- chunk:
				// chunk sent successfully
			case <-ctx.Done():
				c.log.Warn("Context cancelled during stream, stopping goroutine.")
				return
			}
		}
	}()

	return chunkChan, nil
}

// GenerateEmbedding converts text to vector embeddings using a specified model.
func (c *openAIClient) GenerateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Default to the recommended embedding model if not specified.
	model := openai.AdaEmbeddingV2
	if req.Model != "" {
		model = openai.EmbeddingModel(req.Model)
	}

	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: req.Input,
		Model: model,
	})
	if err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		embeddings[i] = d.Embedding
	}

	return &interfaces.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: interfaces.UsageStats{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}

// toOpenAIChatRequest is a helper to convert our internal request format to the go-openai format.
func toOpenAIChatRequest(req *interfaces.LLMRequest) openai.ChatCompletionRequest {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}
}

//Personal.AI order the ending
