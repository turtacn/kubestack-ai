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

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// geminiClient is the concrete implementation of the LLMClient for Google's Gemini API.
// It wraps the 'generative-ai-go' library.
type geminiClient struct {
	model     *genai.GenerativeModel
	embModel  *genai.EmbeddingModel
	log       logger.Logger
}

// NewGeminiClient creates a new client for interacting with Google Gemini.
func NewGeminiClient(ctx context.Context, apiKey string) (interfaces.LLMClient, error) {
	if apiKey == "" {
		return nil, errors.New("Google Gemini API key cannot be empty")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// In a real app, the model names would come from config.
	model := client.GenerativeModel("gemini-pro")
	embModel := client.EmbeddingModel("embedding-001")

	return &geminiClient{
		model:    model,
		embModel: embModel,
		log:      logger.NewLogger("gemini-client"),
	}, nil
}

// SendMessage sends a standard, non-streaming request.
func (c *geminiClient) SendMessage(ctx context.Context, req *interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	c.log.Debugf("Sending chat completion request to Gemini model")

	session := c.model.StartChat()
	// All but the last message is considered history.
	session.History = toGenaiContent(req.Messages[:len(req.Messages)-1])

	lastMessage := req.Messages[len(req.Messages)-1].Content
	// TODO: Add support for multimodal input (images, etc.) by adding other genai.Part types.
	resp, err := session.SendMessage(ctx, genai.Text(lastMessage))
	if err != nil {
		return nil, err
	}

	fullText, err := extractTextFromResponse(resp)
	if err != nil {
		return nil, err
	}

	// NOTE: Gemini API does not provide detailed token usage stats for chat sessions in the same way as OpenAI.
	return &interfaces.LLMResponse{
		Message: interfaces.Message{
			Role:    "assistant", // Gemini uses "model", we map to "assistant" for consistency.
			Content: fullText,
		},
	}, nil
}

// SendStreamingMessage sends a request and returns a channel for streaming the response.
func (c *geminiClient) SendStreamingMessage(ctx context.Context, req *interfaces.LLMRequest) (<-chan interfaces.StreamingChunk, error) {
	c.log.Debugf("Sending streaming chat completion request to Gemini model")

	session := c.model.StartChat()
	session.History = toGenaiContent(req.Messages[:len(req.Messages)-1])

	lastMessage := req.Messages[len(req.Messages)-1].Content
	iter := session.SendMessageStream(ctx, genai.Text(lastMessage))

	chunkChan := make(chan interfaces.StreamingChunk)

	go func() {
		defer close(chunkChan)
		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				c.log.Debug("Gemini stream finished.")
				return
			}
			if err != nil {
				c.log.Errorf("Error receiving Gemini stream chunk: %v", err)
				chunkChan <- interfaces.StreamingChunk{Err: err}
				return
			}

			fullText, _ := extractTextFromResponse(resp)
			chunk := interfaces.StreamingChunk{Content: fullText}
			select {
			case chunkChan <- chunk:
				// chunk sent
			case <-ctx.Done():
				c.log.Warn("Context cancelled during stream, stopping goroutine.")
				return
			}
		}
	}()

	return chunkChan, nil
}

// GenerateEmbedding converts text to vector embeddings.
func (c *geminiClient) GenerateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	batch := c.embModel.NewBatch()
	for _, input := range req.Input {
		batch.AddContent(genai.Text(input))
	}

	res, err := c.embModel.BatchEmbedContents(ctx, batch)
	if err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(res.Embeddings))
	for i, e := range res.Embeddings {
		embeddings[i] = e.Values
	}

	// NOTE: Gemini API does not provide token usage stats for embeddings.
	return &interfaces.EmbeddingResponse{
		Embeddings: embeddings,
	}, nil
}

// toGenaiContent converts our internal message format to Gemini's []*genai.Content format.
func toGenaiContent(messages []interfaces.Message) []*genai.Content {
	var history []*genai.Content
	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		// The "system" role is handled differently in Gemini (via `system_instruction`).
		// For simplicity in this chat-based implementation, we treat it as a user message.
		history = append(history, &genai.Content{
			Parts: []genai.Part{genai.Text(msg.Content)},
			Role:  role,
		})
	}
	return history
}

// extractTextFromResponse consolidates text parts from a Gemini response.
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", errors.New("gemini response contained no valid candidates")
	}
	var fullText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			fullText += string(txt)
		}
	}
	return fullText, nil
}

//Personal.AI order the ending
