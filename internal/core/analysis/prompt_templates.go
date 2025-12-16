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

package analysis

import (
	"encoding/json"
	"fmt"
)

// PromptTemplate defines the structure for AI analysis prompts.
type PromptTemplate struct {
	SystemPrompt string
	UserPrompt   string
}

// GetAIAnalysisPromptTemplate returns the prompt template for AI-based analysis.
// This template is designed to produce structured, JSON-only output that conforms
// to the AIOutput schema.
func GetAIAnalysisPromptTemplate() *PromptTemplate {
	return &PromptTemplate{
		SystemPrompt: getSystemPrompt(),
		UserPrompt:   getUserPromptTemplate(),
	}
}

// getSystemPrompt returns the system prompt that constrains the AI's behavior.
// This prompt enforces JSON-only output and defines the AI's role as an analyzer.
func getSystemPrompt() string {
	return `You are an expert middleware diagnostics assistant specializing in analyzing operational data from cloud-native systems.

Your role is to analyze collected metrics, logs, and configuration data to identify issues, anomalies, and potential problems.

CRITICAL CONSTRAINTS:
1. You MUST respond ONLY with valid JSON. No markdown, no explanations outside the JSON structure.
2. Your response MUST strictly conform to the AIOutput schema defined below.
3. Do NOT include any text before or after the JSON object.
4. If you cannot identify any issues, return an empty issues array.
5. Base your analysis ONLY on the provided data. Do not make assumptions about data you don't have.

AIOutput Schema:
{
  "summary": "string - High-level overview of findings",
  "reasoning": "string (optional) - Your analytical reasoning",
  "issues": [
    {
      "id": "string - Unique identifier (e.g., 'issue-001')",
      "title": "string - Concise issue title",
      "severity": "string - One of: Critical, High, Medium, Low, Info",
      "description": "string - Detailed explanation",
      "evidence": "string - Specific data supporting this finding",
      "recommendations": [
        {
          "id": "string - Unique identifier",
          "description": "string - Actionable recommendation",
          "canAutoFix": boolean,
          "priority": number (0=Low, 1=Medium, 2=High)
        }
      ]
    }
  ]
}

Analysis Guidelines:
- Prioritize issues by severity: Critical > High > Medium > Low > Info
- Provide specific evidence from the data (metrics values, log patterns, config settings)
- Recommend concrete, actionable fixes
- Consider common patterns: memory issues, connection problems, performance degradation, misconfigurations
- Be conservative: only flag genuine issues, not normal operational variations`
}

// getUserPromptTemplate returns the user prompt template.
// This will be populated with actual data at runtime.
func getUserPromptTemplate() string {
	return `Analyze the following middleware diagnostic data and identify any issues or anomalies.

Context:
- Middleware: {{.Middleware}}
- Instance: {{.Instance}}
- Namespace: {{.Namespace}}
- Timestamp: {{.Timestamp}}

Collected Data:
{{.DataJSON}}

Provide your analysis as a JSON object following the AIOutput schema. Remember: JSON ONLY, no additional text.`
}

// RenderPrompt renders the prompt template with actual data.
func RenderPrompt(template *PromptTemplate, input *AIInput) (string, error) {
	// Convert input data to JSON string
	dataJSON, err := json.MarshalIndent(input.Data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal input data: %w", err)
	}

	// Simple template substitution
	// In a production system, consider using a proper template engine
	userPrompt := template.UserPrompt

	// Replace template variables
	userPrompt = replaceTemplateVar(userPrompt, "Middleware", input.Context.Middleware)
	userPrompt = replaceTemplateVar(userPrompt, "Instance", input.Context.Instance)
	userPrompt = replaceTemplateVar(userPrompt, "Namespace", input.Context.Namespace)
	userPrompt = replaceTemplateVar(userPrompt, "Timestamp", input.Context.Timestamp)
	userPrompt = replaceTemplateVar(userPrompt, "DataJSON", string(dataJSON))

	return userPrompt, nil
}

// replaceTemplateVar is a simple template variable replacement function.
func replaceTemplateVar(template, key, value string) string {
	placeholder := fmt.Sprintf("{{.%s}}", key)
	// Handle empty values gracefully
	if value == "" {
		value = "N/A"
	}
	return replaceAll(template, placeholder, value)
}

// replaceAll is a simple string replacement helper.
func replaceAll(s, old, new string) string {
	// Simple implementation - in production, use strings.ReplaceAll
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// indexOf finds the first occurrence of substr in s.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetExampleAIOutput returns an example AIOutput for documentation purposes.
func GetExampleAIOutput() *AIOutput {
	return &AIOutput{
		Summary:   "Analysis identified 2 issues: high memory usage and connection pool exhaustion",
		Reasoning: "Memory usage is at 95% which is above recommended threshold. Connection pool shows 0 available connections indicating saturation.",
		Issues: []AIIssue{
			{
				ID:          "issue-001",
				Title:       "High Memory Usage",
				Severity:    "High",
				Description: "Memory usage is at 95% of allocated limit, risking OOM kills",
				Evidence:    "used_memory: 950MB, maxmemory: 1000MB",
				Recommendations: []AIRecommendation{
					{
						ID:          "rec-001",
						Description: "Increase memory limit to 2GB",
						CanAutoFix:  false,
						Priority:    2,
					},
					{
						ID:          "rec-002",
						Description: "Enable memory eviction policies",
						CanAutoFix:  true,
						Priority:    1,
					},
				},
			},
			{
				ID:          "issue-002",
				Title:       "Connection Pool Exhaustion",
				Severity:    "Critical",
				Description: "All connections in the pool are in use, blocking new requests",
				Evidence:    "connected_clients: 100, maxclients: 100",
				Recommendations: []AIRecommendation{
					{
						ID:          "rec-003",
						Description: "Increase maxclients configuration parameter",
						CanAutoFix:  true,
						Priority:    2,
					},
				},
			},
		},
	}
}
