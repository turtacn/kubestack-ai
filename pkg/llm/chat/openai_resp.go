package chat

import (
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
)

// Helper structs for ChatResponse interface

type openAIChatResponse struct {
	openaiCompletion *openai.ChatCompletion
}

var _ ChatResponse = (*openAIChatResponse)(nil)

func (r *openAIChatResponse) UsageMetadata() any {
	// Check if the main completion object and Usage exist
	if r.openaiCompletion != nil && r.openaiCompletion.Usage.TotalTokens > 0 { // Check a field within Usage
		return r.openaiCompletion.Usage
	}
	return nil
}

func (r *openAIChatResponse) Candidates() []Candidate {
	if r.openaiCompletion == nil {
		return nil
	}
	candidates := make([]Candidate, len(r.openaiCompletion.Choices))
	for i, choice := range r.openaiCompletion.Choices {
		candidates[i] = &openAICandidate{openaiChoice: &choice}
	}
	return candidates

}

type openAICandidate struct {
	openaiChoice *openai.ChatCompletionChoice
}

var _ Candidate = (*openAICandidate)(nil)

func (c *openAICandidate) Parts() []Part {
	// Check if the choice exists before accessing Message
	if c.openaiChoice == nil {
		return nil
	}

	// OpenAI message can have Content AND ToolCalls
	var parts []Part
	if c.openaiChoice.Message.Content != "" {
		parts = append(parts, &openAIPart{content: c.openaiChoice.Message.Content})
	}
	if len(c.openaiChoice.Message.ToolCalls) > 0 {
		parts = append(parts, &openAIPart{toolCalls: c.openaiChoice.Message.ToolCalls})
	}
	return parts
}

// String provides a simple string representation for logging/debugging.
func (c *openAICandidate) String() string {
	if c.openaiChoice == nil {
		return "<nil candidate>"
	}
	content := "<no content>"
	if c.openaiChoice.Message.Content != "" {
		content = c.openaiChoice.Message.Content
	}
	toolCalls := len(c.openaiChoice.Message.ToolCalls)
	finishReason := string(c.openaiChoice.FinishReason)
	return fmt.Sprintf("Candidate(FinishReason: %s, ToolCalls: %d, Content: %q)", finishReason, toolCalls, content)
}

type openAIPart struct {
	content   string
	toolCalls []openai.ChatCompletionMessageToolCall // Correct type
}

var _ Part = (*openAIPart)(nil)

func (p *openAIPart) AsText() (string, bool) {
	return p.content, p.content != ""
}

func (p *openAIPart) AsFunctionCalls() ([]FunctionCall, bool) {
	return convertToolCallsToFunctionCalls(p.toolCalls)
}

// Update openAIChatStreamResponse to include accumulated content
type openAIChatStreamResponse struct {
	streamChunk openai.ChatCompletionChunk
	accumulator openai.ChatCompletionAccumulator
	content     string
	toolCalls   []openai.ChatCompletionMessageToolCall
}

// Update Candidates() to use accumulated content
func (r *openAIChatStreamResponse) Candidates() []Candidate {
	if len(r.streamChunk.Choices) == 0 {
		return nil
	}

	candidates := make([]Candidate, len(r.streamChunk.Choices))
	for i, choice := range r.streamChunk.Choices {
		candidates[i] = &openAIStreamCandidate{
			streamChoice: choice,
			content:      r.content,
			toolCalls:    r.toolCalls,
		}
	}
	return candidates
}

// Update openAIStreamCandidate to handle delta content
type openAIStreamCandidate struct {
	streamChoice openai.ChatCompletionChunkChoice
	content      string // This will now be just the delta content
	toolCalls    []openai.ChatCompletionMessageToolCall
}

// Update Parts() to handle delta content
func (c *openAIStreamCandidate) Parts() []Part {
	var parts []Part

	// Only include the delta content
	if c.content != "" {
		parts = append(parts, &openAIStreamPart{
			content: c.content,
		})
	}

	// Include accumulated tool calls
	if len(c.toolCalls) > 0 {
		parts = append(parts, &openAIStreamPart{
			toolCalls: c.toolCalls,
		})
	}

	return parts
}

// Add UsageMetadata implementation
func (r *openAIChatStreamResponse) UsageMetadata() any {
	if r.accumulator.Usage.TotalTokens > 0 {
		return r.accumulator.Usage
	}
	return nil
}

// Add String implementation
func (c *openAIStreamCandidate) String() string {
	return fmt.Sprintf("StreamingCandidate(Content: %q, ToolCalls: %d)",
		c.content, len(c.toolCalls))
}

// Define openAIStreamPart
type openAIStreamPart struct {
	content   string
	toolCalls []openai.ChatCompletionMessageToolCall
}

// Ensure openAIStreamPart implements Part interface
var _ Part = (*openAIStreamPart)(nil)

func (p *openAIStreamPart) AsText() (string, bool) {
	return p.content, p.content != ""
}

func (p *openAIStreamPart) AsFunctionCalls() ([]FunctionCall, bool) {
	return convertToolCallsToFunctionCalls(p.toolCalls)
}

// openAISchema wraps a llm Schema with OpenAI-specific marshaling behavior
type openAISchema struct {
	*Schema
}

// MarshalJSON provides OpenAI-specific JSON marshaling that ensures object schemas have properties
func (s openAISchema) MarshalJSON() ([]byte, error) {
	// Create a map to build the JSON representation
	result := make(map[string]interface{})

	if s.Type != "" {
		result["type"] = string(s.Type)
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	if len(s.Required) > 0 {
		result["required"] = s.Required
	}

	// For object types, always include properties (even if empty) to satisfy OpenAI
	if s.Schema.Type == string(TypeObject) {
		if s.Schema.Properties != nil {
			result["properties"] = s.Schema.Properties
		} else {
			result["properties"] = make(map[string]*Schema)
		}
	} else if s.Schema.Properties != nil && len(s.Schema.Properties) > 0 {
		// For non-object types, only include properties if they exist and are non-empty
		result["properties"] = s.Schema.Properties
	}

	if s.Items != nil {
		result["items"] = s.Items
	}

	return json.Marshal(result)
}
