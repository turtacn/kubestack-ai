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

// Package prompt provides tools for building and managing LLM prompts.
package prompt

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// Template represents a single, versioned prompt template. It uses Go's standard
// `text/template` format, which allows for flexible and safe variable substitution.
type Template struct {
	// ID is the unique identifier for the prompt template (e.g., "diagnose-redis").
	ID string
	// Version allows for tracking and managing different versions of the same prompt.
	Version string
	// Text is the Go template string that forms the body of the prompt.
	Text string
}

// Builder is responsible for constructing final prompts from templates and dynamic
// data. It encapsulates the logic of prompt engineering, separating it from the
// application's core operational logic. It pre-parses templates for efficiency.
type Builder struct {
	templates map[string]*template.Template
}

// NewBuilder creates a new prompt builder. It takes a slice of Template objects,
// pre-parses them using Go's `text/template` engine, and stores them in a map
// for efficient access.
//
// Parameters:
//   templates ([]*Template): A slice of prompt templates to be loaded into the builder.
//
// Returns:
//   *Builder: A new, initialized prompt builder.
//   error: An error if any of the templates fail to parse.
func NewBuilder(templates []*Template) (*Builder, error) {
	parsedTmpls := make(map[string]*template.Template)
	for _, t := range templates {
		tmpl, err := template.New(t.ID).Parse(t.Text)
		if err != nil {
			return nil, fmt.Errorf("failed to parse prompt template '%s': %w", t.ID, err)
		}
		parsedTmpls[t.ID] = tmpl
	}
	return &Builder{templates: parsedTmpls}, nil
}

// Build constructs a final list of messages to be sent to an LLM. It selects a
// template by its ID, executes it with the provided data to generate a system
// prompt, and then combines this with the conversation history and any optional
// few-shot examples to create the final message sequence.
//
// Parameters:
//   templateID (string): The ID of the prompt template to use (e.g., "diagnose-redis").
//   data (interface{}): A struct or map containing data to be injected into the template's variables.
//   history ([]interfaces.Message): The existing conversation history between the user and the assistant.
//   fewShotExamples (...interfaces.Message): Optional one-shot or few-shot examples to guide the model's response.
//
// Returns:
//   []interfaces.Message: The final, ordered list of messages ready to be sent to the LLM.
//   error: An error if the template is not found or fails to execute.
func (b *Builder) Build(templateID string, data interface{}, history []interfaces.Message, fewShotExamples ...interfaces.Message) ([]interfaces.Message, error) {
	tmpl, ok := b.templates[templateID]
	if !ok {
		return nil, fmt.Errorf("prompt template with ID '%s' not found", templateID)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute prompt template '%s' with given data: %w", templateID, err)
	}

	systemPrompt := buf.String()

	// The final message list is constructed in a specific order:
	// 1. The system prompt, which sets the context and instructions for the AI.
	// 2. Few-shot examples, which demonstrate the desired input/output format.
	// 3. The actual conversation history.
	// This structure is standard for many conversational LLMs.
	finalMessages := []interfaces.Message{
		{Role: "system", Content: systemPrompt},
	}

	if len(fewShotExamples) > 0 {
		finalMessages = append(finalMessages, fewShotExamples...)
	}

	finalMessages = append(finalMessages, history...)

	return finalMessages, nil
}

// TODO: Implement A/B testing by allowing a single template ID to map to multiple template variations.
// The Build method could then select a variation based on a given strategy (e.g., random, round-robin).

// TODO: Implement prompt format adaptation for different models. Some models might have specific
// requirements for how few-shot examples are presented or might use different role names.

//Personal.AI order the ending
