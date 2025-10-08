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

// diagnosisTemplate is a generic template for diagnosing any middleware. It instructs the LLM
// on the process to follow (analyze, identify root cause, suggest solutions) and the desired output format.
const diagnosisTemplate = `You are KubeStack-AI, an expert system designed to diagnose issues in middleware and infrastructure.
Your task is to analyze the provided data, identify the root cause of any problems, and suggest actionable solutions.

**Analysis Context:**
- Middleware Type: {{.MiddlewareName}}
- Instance: {{.InstanceName}}
- Timestamp: {{.Timestamp}}

**Collected Data:**
'''
{{.CollectedData}}
'''

**Instructions:**
1.  **Analyze Data:** Carefully review all sections of the collected data. Look for explicit error messages, metric values that violate common thresholds, and configurations that deviate from best practices.
2.  **Identify Root Cause:** Synthesize your findings to determine the most likely root cause for each potential problem.
3.  **Suggest Solutions:** Provide clear, step-by-step solutions to fix each identified issue. If possible, provide the exact commands to run or configuration changes to make.
4.  **Format Output:** Your final output must be a single, valid JSON object. This object should contain a single key, "issues", which is a list of issue objects. Each issue object must have the following keys: "title" (a short, descriptive title), "severity" (one of "Low", "Medium", "High", "Critical"), "description" (a detailed explanation of the problem), and "recommendations" (a list of recommendation objects, each with a "description" and an optional "command").
`

// redisDiagnosisTemplate is a more specific template for Redis. It includes a few-shot example
// to guide the LLM in generating the correct output format and reasoning process.
const redisDiagnosisTemplate = `You are KubeStack-AI, a Redis performance and reliability expert.
Your task is to analyze the provided Redis INFO and CONFIG data to identify issues and provide solutions.

**Analysis Context:**
- Middleware Type: Redis
- Instance: {{.InstanceName}}

**Collected Data:**
- INFO:
'''
{{.Info}}
'''
- CONFIG:
'''
{{.Config}}
'''

**Instructions:**
Analyze the data and respond with a single, valid JSON object containing a list of "issues", as shown in the example.
Pay close attention to memory usage (especially fragmentation), persistence settings (RDB/AOF), and security configurations (requirepass, bind address).

**Example:**

*Input Data Analysis:*
- The provided INFO shows 'mem_fragmentation_ratio:1.8'. This is greater than the recommended maximum of 1.5.
- The provided CONFIG shows 'requirepass ""'. This means the instance is not password protected.

*Your JSON Output:*
{
  "issues": [
    {
      "title": "High Memory Fragmentation",
      "severity": "Warning",
      "description": "The memory fragmentation ratio is 1.8, which is high. This can lead to the operating system allocating more memory than requested, resulting in wasted resources.",
      "recommendations": [
        {
          "description": "A high fragmentation ratio can often be resolved by restarting the Redis server, which allows the OS to reclaim the fragmented memory. Ensure persistence is enabled to avoid data loss during the restart."
        }
      ]
    },
    {
      "title": "Password Protection Disabled",
      "severity": "Critical",
      "description": "The 'requirepass' configuration is empty, leaving the Redis instance unprotected and open to unauthorized access.",
      "recommendations": [
        {
          "description": "Set a strong password via the 'requirepass' configuration option to secure the instance.",
          "command": "CONFIG SET requirepass \"your-strong-password-here\""
        }
      ]
    }
  ]
}

**Begin Analysis.**
`

// GetDefaultTemplates returns a slice containing all the default, built-in prompt
// templates for the application. In a production system, these templates might be
// loaded from an external configuration file or a dedicated template registry,
// but providing them as code ensures that the application has a working set of
// prompts out of the box.
//
// Returns:
//   []*Template: A slice of the default prompt templates.
func GetDefaultTemplates() []*Template {
	return []*Template{
		{
			ID:      "generic-diagnosis",
			Version: "1.0",
			Text:    diagnosisTemplate,
		},
		{
			ID:      "redis-diagnosis",
			Version: "1.0",
			Text:    redisDiagnosisTemplate,
		},
		// TODO: Add other templates here, e.g., for MySQL, Kafka, etc.
	}
}

//Personal.AI order the ending
