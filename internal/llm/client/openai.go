package client

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the LLMClient interface using the OpenAI API.
type OpenAIClient struct {
	client *openai.Client
}

// NewOpenAIClient creates a new OpenAIClient.
func NewOpenAIClient(apiKey string) (*OpenAIClient, error) {
	client := openai.NewClient(apiKey)
	return &OpenAIClient{client: client}, nil
}

// SendMessage sends a request to the LLM and waits for a complete response.
func (c *OpenAIClient) SendMessage(ctx context.Context, req *interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(req.Temperature), // Cast might be needed depending on go-openai version, it usually takes float32
		MaxTokens:   req.MaxTokens,
	})

	if err != nil {
		return nil, fmt.Errorf("openai completion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
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

// SendStreamingMessage sends a request and returns a channel for response chunks.
func (c *OpenAIClient) SendStreamingMessage(ctx context.Context, req *interfaces.LLMRequest) (<-chan interfaces.StreamingChunk, error) {
	// TODO: Implement streaming
	return nil, fmt.Errorf("streaming not implemented")
}

// GenerateEmbedding generates vector embeddings.
func (c *OpenAIClient) GenerateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// TODO: Implement embedding
	return nil, fmt.Errorf("embedding not implemented")
}

// Legacy Complete method for backward compatibility if needed, but we should migrate.
func (c *OpenAIClient) Complete(ctx context.Context, prompt string, options ...interfaces.LLMOption) (string, error) {
	req := &interfaces.LLMRequest{
		Model: "gpt-4", // Default
		Messages: []interfaces.Message{
			{Role: "user", Content: prompt},
		},
	}
	resp, err := c.SendMessage(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}
