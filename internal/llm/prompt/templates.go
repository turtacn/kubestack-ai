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

package prompt

// TemplateDiagnosisID is the identifier for the standard diagnosis prompt.
const TemplateDiagnosisID = "diagnosis_v1"

// AllTemplates is a slice containing all the default prompt templates for the application.
// This list is used to initialize the prompt builder.
var AllTemplates = []*Template{
	{
		ID:      TemplateDiagnosisID,
		Version: "1.0",
		Text: `You are an expert SRE assistant. Your goal is to analyze the provided context data from a middleware system and identify potential issues.
The context data is provided in JSON format below.

Context Data:
{{.context_data}}

Analyze the data and respond with a JSON object that follows this exact structure:
{
  "summary": "A brief, one-sentence summary of the main finding.",
  "reasoning": "A step-by-step explanation of how you reached your conclusion based on the provided data.",
  "issues": [
    {
      "id": "issue-uuid-here",
      "title": "A concise title for the issue.",
      "severity": "One of: Low, Medium, High, Critical.",
      "description": "A detailed explanation of the issue, its impact, and why it's a problem.",
      "evidence": "A snippet of data or a summary of the evidence that points to this issue.",
      "recommendations": [
        {
          "id": "rec-uuid-here",
          "description": "A clear, actionable recommendation to fix the issue.",
          "command": "The exact shell command to run for an automated fix, if applicable. Otherwise, an empty string.",
          "canAutoFix": true
        }
      ]
    }
  ]
}

Only return the JSON object. Do not include any other text, greetings, or explanations.`,
	},
}