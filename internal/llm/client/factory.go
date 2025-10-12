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
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// NewClient is a factory function that creates and returns an LLMClient
// implementation based on the provided configuration. This allows the application
// to easily switch between different LLM providers.
func NewClient(cfg *config.LLMConfig) (interfaces.LLMClient, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIClient(cfg.OpenAI.APIKey)
	case "gemini":
		// Gemini client needs a context for initialization.
		// We use context.Background() here because this happens at startup,
		// before any request-specific context is available.
		return NewGeminiClient(context.Background(), cfg.Gemini.APIKey)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}