package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"k8s.io/klog/v2"
)

// Chat Session Implementation
type OpenAIChatSession struct {
	Client              openai.Client
	History             []openai.ChatCompletionMessageParamUnion
	Model               string
	FunctionDefinitions []*FunctionDefinition            // 封装的工具定义数组
	Tools               []openai.ChatCompletionToolParam // openai的工具定义数组
}

// Ensure OpenAIChatSession implements the Chat interface.
var _ Chat = (*OpenAIChatSession)(nil)

// SetFunctionDefinitions stores the function definitions and converts them to OpenAI format.
func (cs *OpenAIChatSession) SetFunctionDefinitions(defs []*FunctionDefinition) error {
	cs.FunctionDefinitions = defs
	cs.Tools = nil // Clear previous tools
	if len(defs) > 0 {
		cs.Tools = make([]openai.ChatCompletionToolParam, len(defs))
		for i, llmDef := range defs {
			klog.Infof("Processing function definition: %s", llmDef.Name)

			// Process function parameters
			params, err := cs.convertFunctionParameters(llmDef)
			if err != nil {
				return fmt.Errorf("failed to process parameters for function %s: %w", llmDef.Name, err)
			}

			cs.Tools[i] = openai.ChatCompletionToolParam{
				Function: openai.FunctionDefinitionParam{
					Name:        llmDef.Name,
					Description: openai.String(llmDef.Description),
					Parameters:  params,
				},
			}
		}
	}
	klog.V(1).Infof("Set %d function definitions for OpenAI chat session", len(cs.FunctionDefinitions))
	return nil
}

// Send sends the user message(s), appends to history, and gets the LLM response.
func (cs *OpenAIChatSession) Send(ctx context.Context, contents ...*Message) (ChatResponse, error) {
	klog.V(1).InfoS("OpenAIChatSession.Send called", "model", cs.Model, "history_len", len(cs.History))

	// Process and append messages to history
	if err := cs.addContentsToHistory(contents); err != nil {
		return nil, err
	}

	// Prepare and send API request
	chatReq := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(cs.Model),
		Messages: cs.History,
	}
	if len(cs.Tools) > 0 {
		chatReq.Tools = cs.Tools
	}

	// Call the OpenAI API
	klog.V(1).InfoS("Sending request to OpenAI Chat API", "model", cs.Model, "messages", len(chatReq.Messages), "tools", len(chatReq.Tools))
	completion, err := cs.Client.Chat.Completions.New(ctx, chatReq)
	if err != nil {
		// TODO: Check if error is retryable using cs.IsRetryableError
		klog.Errorf("OpenAI ChatCompletion API error: %v", err)
		return nil, fmt.Errorf("OpenAI chat completion failed: %w", err)
	}
	klog.V(1).InfoS("Received response from OpenAI Chat API", "id", completion.ID, "choices", len(completion.Choices))

	// Process the response
	if len(completion.Choices) == 0 {
		klog.Warning("Received response with no choices from OpenAI")
		return nil, errors.New("received empty response from OpenAI (no choices)")
	}

	assistantMsg := completion.Choices[0].Message
	// Convert to param type before appending to history
	cs.History = append(cs.History, assistantMsg.ToParam())
	klog.V(2).InfoS("Added assistant message to history", "content_present", assistantMsg.Content != "", "tool_calls", len(assistantMsg.ToolCalls))

	// Wrap the response
	resp := &openAIChatResponse{
		openaiCompletion: completion,
	}

	return resp, nil
}

// SendStreaming sends the user message(s) and returns an iterator for the LLM response stream.
func (cs *OpenAIChatSession) SendStreaming(ctx context.Context, contents ...*Message) (ChatResponseIterator, error) {
	klog.V(1).InfoS("Starting OpenAI streaming request", "model", cs.Model)

	// Process and append messages to history
	if err := cs.addContentsToHistory(contents); err != nil {
		return nil, err
	}

	// Prepare and send API request
	chatReq := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(cs.Model),
		Messages: cs.History,
	}
	if len(cs.Tools) > 0 {
		chatReq.Tools = cs.Tools
	}

	// Start the OpenAI streaming request
	klog.V(1).InfoS("Sending streaming request to OpenAI API",
		"model", cs.Model,
		"messageCount", len(chatReq.Messages),
		"toolCount", len(chatReq.Tools))

	stream := cs.Client.Chat.Completions.NewStreaming(ctx, chatReq)

	// Create an accumulator to track the full response
	acc := openai.ChatCompletionAccumulator{}

	// Create and return the stream iterator
	return func(yield func(ChatResponse, error) bool) {
		defer stream.Close()

		var lastResponseChunk *openAIChatStreamResponse
		var currentContent strings.Builder
		var currentToolCalls []openai.ChatCompletionMessageToolCall

		// Process stream chunks
		for stream.Next() {
			chunk := stream.Current()

			// Update the accumulator with the new chunk
			acc.AddChunk(chunk)

			// Handle content completion
			if _, ok := acc.JustFinishedContent(); ok {
				klog.V(2).Info("Content stream finished")
			}

			// Handle refusal completion
			if refusal, ok := acc.JustFinishedRefusal(); ok {
				klog.V(2).Infof("Refusal stream finished: %v", refusal)
				yield(nil, fmt.Errorf("model refused to respond: %v", refusal))
				return
			}

			// Handle tool call completion
			var toolCallsForThisChunk []openai.ChatCompletionMessageToolCall
			if tool, ok := acc.JustFinishedToolCall(); ok {
				klog.V(2).Infof("Tool call finished: %s %s", tool.Name, tool.Arguments)
				newToolCall := openai.ChatCompletionMessageToolCall{
					ID: tool.ID,
					Function: openai.ChatCompletionMessageToolCallFunction{
						Name:      tool.Name,
						Arguments: tool.Arguments,
					},
				}
				currentToolCalls = append(currentToolCalls, newToolCall)
				// Only include the newly finished tool call in this chunk
				toolCallsForThisChunk = []openai.ChatCompletionMessageToolCall{newToolCall}
			}

			streamResponse := &openAIChatStreamResponse{
				streamChunk: chunk,
				accumulator: acc,
				content:     "", // Default to empty content
				toolCalls:   toolCallsForThisChunk,
			}

			// Only process content if there are choices and a delta
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				if delta.Content != "" {
					currentContent.WriteString(delta.Content)
					streamResponse.content = delta.Content // Only set content if there's new content
				}
			}

			// Keep track of the last response for history
			lastResponseChunk = &openAIChatStreamResponse{
				streamChunk: chunk,
				accumulator: acc,
				content:     currentContent.String(), // Full accumulated content for history
				toolCalls:   currentToolCalls,
			}

			// Only yield if there's actual content or tool calls to report
			if streamResponse.content != "" || len(streamResponse.toolCalls) > 0 {
				if !yield(streamResponse, nil) {
					return
				}
			}
		}

		// Check for errors after streaming completes
		if err := stream.Err(); err != nil {
			klog.Errorf("Error in OpenAI streaming: %v", err)
			yield(nil, fmt.Errorf("OpenAI streaming error: %w", err))
			return
		}

		// Update conversation history with the complete message
		if lastResponseChunk != nil {
			completeMessage := openai.ChatCompletionMessage{
				Content:   currentContent.String(),
				Role:      "assistant",
				ToolCalls: currentToolCalls,
			}

			// Append the full assistant response to history
			cs.History = append(cs.History, completeMessage.ToParam())
			klog.V(2).InfoS("Added complete assistant message to history",
				"content_present", completeMessage.Content != "",
				"tool_calls", len(completeMessage.ToolCalls))
		}
	}, nil
}

// IsRetryableError determines if an error from the OpenAI API should be retried.
func (cs *OpenAIChatSession) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// 简化的重试逻辑 - 检查是否是网络错误或超时
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network unreachable") ||
		strings.Contains(errStr, "EOF")
}

// 转换用户消息为openai的规范
func (cs *OpenAIChatSession) addContentsToHistory(contents []*Message) error {
	for _, content := range contents {
		switch content.Type {
		case MessageTypeText:
			klog.V(2).Infof("Adding user message to history: %s", content.Payload)
			cs.History = append(cs.History, openai.UserMessage(content.Payload))
		case MessageTypeToolCallResponse:
			klog.V(2).Infof("Adding tool call result to history: Name=%s, ID=%s", content.FunCallResult.Name, content.FunCallResult.ID)
			// Marshal the result map into a JSON string for the message content
			resultJSON, err := json.Marshal(content.FunCallResult)
			if err != nil {
				klog.Errorf("Failed to marshal function call result: %v", err)
				return fmt.Errorf("failed to marshal function call result %q: %w", content.FunCallResult.Name, err)
			}
			cs.History = append(cs.History, openai.ToolMessage(string(resultJSON), content.FunCallResult.ID))
		default:
			klog.V(2).Infof("Adding user message to history: %s", content.Payload)
			cs.History = append(cs.History, openai.UserMessage(content.Payload))
		}
	}
	return nil
}

// convertSchemaToBytes converts a validated schema to JSON bytes using OpenAI-specific marshaling
func (cs *OpenAIChatSession) convertSchemaToBytes(schema *Schema, functionName string) ([]byte, error) {
	// Wrap the schema with OpenAI-specific marshaling behavior
	openAIWrapper := openAISchema{Schema: schema}

	bytes, err := json.Marshal(openAIWrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema: %w", err)
	}

	klog.Infof("OpenAI schema for function %s: %s", functionName, string(bytes))

	return bytes, nil
}

func (cs *OpenAIChatSession) Initialize(messages []*Message) error {
	return fmt.Errorf("Initialize not yet implemented for openai")
}

// 转换schema为openai的函数调用规范
func convertSchemaForOpenAI(schema *Schema) (*Schema, error) {
	if schema == nil {
		// Return a minimal valid object schema for OpenAI
		return &Schema{
			Type:       string(TypeObject),
			Properties: make(map[string]*Schema),
		}, nil
	}

	// Create a deep copy to avoid modifying the original
	validated := &Schema{
		Description: schema.Description,
		Required:    make([]string, len(schema.Required)),
	}
	copy(validated.Required, schema.Required)

	// Handle type validation and normalization based on OpenAI requirements
	switch SchemaType(schema.Type) {
	case TypeObject:
		validated.Type = string(TypeObject)
		// Objects MUST have properties for OpenAI (even if empty)
		validated.Properties = make(map[string]*Schema)
		if schema.Properties != nil {
			for key, prop := range schema.Properties {
				validatedProp, err := convertSchemaForOpenAI(prop)
				if err != nil {
					return nil, fmt.Errorf("validating property %q: %w", key, err)
				}
				validated.Properties[key] = validatedProp
			}
		}

	case TypeArray:
		validated.Type = string(TypeArray)
		// Arrays MUST have items schema for OpenAI
		if schema.Items != nil {
			validatedItems, err := convertSchemaForOpenAI(schema.Items)
			if err != nil {
				return nil, fmt.Errorf("validating array items: %w", err)
			}
			validated.Items = validatedItems
		} else {
			// Default to string items if not specified
			validated.Items = &Schema{Type: string(TypeString)}
		}

	case TypeString:
		validated.Type = string(TypeString)

	case TypeNumber:
		validated.Type = string(TypeNumber)

	case TypeInteger:
		// OpenAI prefers "number" for integers
		validated.Type = string(TypeNumber)

	case TypeBoolean:
		validated.Type = string(TypeBoolean)

	case "":
		// If no type specified, default to object with empty properties
		klog.Warningf("Schema has no type, defaulting to object")
		validated.Type = string(TypeObject)
		validated.Properties = make(map[string]*Schema)

	default:
		// For unknown types, log a warning and default to object
		klog.Warningf("Unknown schema type '%s', defaulting to object", schema.Type)
		validated.Type = string(TypeObject)
		validated.Properties = make(map[string]*Schema)
	}

	// Final validation: Ensure object types always have properties
	// This handles edge cases where malformed schemas might slip through
	if validated.Type == string(TypeObject) && validated.Properties == nil {
		klog.Warningf("Object schema missing properties, initializing empty properties map")
		validated.Properties = make(map[string]*Schema)
	}

	return validated, nil
}

// convertFunctionParameters handles the conversion of llm parameters to OpenAI format
func (cs *OpenAIChatSession) convertFunctionParameters(llmDef *FunctionDefinition) (openai.FunctionParameters, error) {
	var params openai.FunctionParameters

	if llmDef.Parameters == nil {
		return params, nil
	}

	// Convert the schema for OpenAI compatibility
	klog.V(2).Infof("Original schema for function %s: %+v", llmDef.Name, llmDef.Parameters)
	validatedSchema, err := convertSchemaForOpenAI(llmDef.Parameters)
	if err != nil {
		return params, fmt.Errorf("schema conversion failed: %w", err)
	}
	klog.V(2).Infof("Converted schema for function %s: %+v", llmDef.Name, validatedSchema)

	// Convert to raw schema bytes
	schemaBytes, err := cs.convertSchemaToBytes(validatedSchema, llmDef.Name)
	if err != nil {
		return params, err
	}

	// Unmarshal into OpenAI parameters format
	if err := json.Unmarshal(schemaBytes, &params); err != nil {
		return params, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return params, nil
}

// 转化openai的工具调用为我们封装好的FunctionCall
func convertToolCallsToFunctionCalls(toolCalls []openai.ChatCompletionMessageToolCall) ([]FunctionCall, bool) {
	if len(toolCalls) == 0 {
		return nil, false
	}

	calls := make([]FunctionCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		// Skip non-function tool calls
		if tc.Function.Name == "" {
			klog.V(2).Infof("Skipping non-function tool call ID: %s", tc.ID)
			continue
		}

		// Parse function arguments with error handling
		var args map[string]any
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				klog.V(2).Infof("Error unmarshalling function arguments for %s: %v", tc.Function.Name, err)
				args = make(map[string]any)
			}
		} else {
			args = make(map[string]any)
		}

		calls = append(calls, FunctionCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return calls, len(calls) > 0
}
