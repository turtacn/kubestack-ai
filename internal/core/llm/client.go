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

// Package llm provides the LLM client contract for the core analysis layer.
// This package re-exports the LLMClient interface from internal/llm/interfaces
// to establish a stable contract boundary between the core analysis layer and
// the LLM implementation layer.
package llm

import (
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// Client is the LLM client interface used by the core analysis layer.
// It is an alias to the actual LLMClient interface to maintain clear
// architectural boundaries and allow for future flexibility.
type Client interface {
	interfaces.LLMClient
}

// Message is an alias to the LLM message type for convenience.
type Message = interfaces.Message

// LLMRequest is an alias to the LLM request type for convenience.
type LLMRequest = interfaces.LLMRequest

// LLMResponse is an alias to the LLM response type for convenience.
type LLMResponse = interfaces.LLMResponse

// UsageStats is an alias to the usage statistics type for convenience.
type UsageStats = interfaces.UsageStats
