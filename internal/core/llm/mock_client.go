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

package llm

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// MockClient is a mock implementation of the LLMClient interface for testing.
// It returns predetermined responses based on configured behavior.
type MockClient struct {
	// Response is the static response to return from SendMessage calls.
	Response string

	// Error is the error to return from SendMessage calls.
	Error error

	// CallCount tracks the number of times SendMessage was called.
	CallCount int

	// LastRequest stores the last request received.
	LastRequest *interfaces.LLMRequest
}

// NewMockClient creates a new MockClient with default values.
func NewMockClient() *MockClient {
	return &MockClient{
		Response:  defaultMockResponse(),
		Error:     nil,
		CallCount: 0,
	}
}

// SendMessage implements the LLMClient interface.
// It returns the configured Response or Error.
func (m *MockClient) SendMessage(ctx context.Context, req *interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	m.CallCount++
	m.LastRequest = req

	if m.Error != nil {
		return nil, m.Error
	}

	return &interfaces.LLMResponse{
		Message: interfaces.Message{
			Role:    "assistant",
			Content: m.Response,
		},
		Usage: interfaces.UsageStats{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

// SendStreamingMessage implements the LLMClient interface.
// For the mock, it's not supported and returns an error.
func (m *MockClient) SendStreamingMessage(ctx context.Context, req *interfaces.LLMRequest) (<-chan interfaces.StreamingChunk, error) {
	return nil, fmt.Errorf("streaming not supported in mock client")
}

// GenerateEmbedding implements the LLMClient interface.
// For the mock, it returns dummy embeddings.
func (m *MockClient) GenerateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	embeddings := make([][]float32, len(req.Input))
	for i := range embeddings {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}
	return &interfaces.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: interfaces.UsageStats{
			PromptTokens: 10,
			TotalTokens:  10,
		},
	}, nil
}

// Complete implements the LLMClient interface for backward compatibility.
// It delegates to SendMessage internally.
func (m *MockClient) Complete(ctx context.Context, prompt string, options ...interfaces.LLMOption) (string, error) {
	req := &interfaces.LLMRequest{
		Messages: []interfaces.Message{
			{Role: "user", Content: prompt},
		},
	}

	for _, opt := range options {
		opt(req)
	}

	resp, err := m.SendMessage(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

// SetResponse sets the response that will be returned by the mock.
func (m *MockClient) SetResponse(response string) {
	m.Response = response
}

// SetError sets the error that will be returned by the mock.
func (m *MockClient) SetError(err error) {
	m.Error = err
}

// Reset resets the mock client to its initial state.
func (m *MockClient) Reset() {
	m.Response = defaultMockResponse()
	m.Error = nil
	m.CallCount = 0
	m.LastRequest = nil
}

// defaultMockResponse returns a default valid JSON response.
func defaultMockResponse() string {
	return `{
  "summary": "Mock analysis completed successfully",
  "issues": [
    {
      "id": "mock-001",
      "source": "AI",
      "title": "Mock Issue for Testing",
      "severity": "Low",
      "description": "This is a mock issue generated for testing purposes.",
      "evidence": "Mock evidence data",
      "recommendations": [
        {
          "id": "mock-rec-001",
          "description": "Mock recommendation",
          "canAutoFix": false,
          "priority": 0
        }
      ]
    }
  ]
}`
}
