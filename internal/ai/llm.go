package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
)

// LLM 接口定义大模型交互。LLM interface for large language model interaction.
type LLM interface {
	Analyze(ctx context.Context, metrics *models.Metrics, logs models.Logs, config *models.Config,
		knowledge string, envInfo models.Config) ([]models.Finding, error)
	GenerateFix(ctx context.Context, finding models.Finding) (models.Recommendation, error)
	Query(ctx context.Context, input string) (string, error)
}

// llm LLM实现。llm implements LLM.
type llm struct {
	client *openai.Client
	model  string
}

// NewLLM 创建LLM实例。NewLLM creates LLM instance.
func NewLLM(apiKey string, model string) LLM {
	if model == "" {
		model = "gpt-4"
	}

	return &llm{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Analyze 分析数据。Analyze data.
func (l *llm) Analyze(ctx context.Context, metrics *models.Metrics, logs models.Logs, config *models.Config,
	knowledge string, envInfo models.Config) ([]models.Finding, error) {

	logging.Logger.Debug("Starting AI analysis with LLM")

	// 构建提示信息。Build prompt.
	prompt := `You are a senior middleware expert with extensive experience in troubleshooting and optimization.
Analyze the following data to identify issues, their severity, and recommend fixes.

Environment Information:
` + formatData(envInfo) + `

Middleware Metrics:
` + formatData(metrics) + `

Relevant Logs:
` + formatLogs(logs) + `

Configuration:
` + formatData(config) + `

Relevant Knowledge:
` + knowledge + `

Provide your analysis in JSON format with the following structure:
{
  "findings": [
    {
      "type": "issue type (e.g., performance, reliability, security)",
      "title": "brief issue title",
      "detail": "detailed explanation",
      "evidence": ["specific metrics/logs that support this finding"],
      "severity": "low|medium|high",
      "recommendations": [
        {
          "description": "detailed fix recommendation",
          "command": "relevant command if applicable",
          "auto_fix": true|false,
          "risk_level": "low|medium|high"
        }
      ]
    }
  ]
}

Only return the JSON without any additional text.`

	// 调用OpenAI API。Call OpenAI API.
	resp, err := l.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParam{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: openai.F(" %s", prompt),
			},
		},
		Model:       openai.F(l.model),
		Temperature: openai.F(0.3), // 低温度使输出更确定。Low temperature for more deterministic output.
	})

	if err != nil {
		logging.Logger.Errorf("LLM API call failed: %v", err)
		return nil, errors.ErrLLMCallFailed
	}

	if len(resp.Choices) == 0 {
		logging.Logger.Error("No response from LLM")
		return nil, errors.ErrLLMCallFailed
	}

	// 解析响应。Parse response.
	var result struct {
		Findings []models.Finding `json:"findings"`
	}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		logging.Logger.Errorf("Failed to parse LLM response: %v. Response: %s",
			err, resp.Choices[0].Message.Content)
		return nil, errors.ErrLLMCallFailed
	}

	logging.Logger.Debugf("AI analysis completed with %d findings", len(result.Findings))
	return result.Findings, nil
}

// GenerateFix 生成修复命令。GenerateFix generates fix command.
func (l *llm) GenerateFix(ctx context.Context, finding models.Finding) (models.Recommendation, error) {
	logging.Logger.Debugf("Generating fix for finding: %s", finding.Title)

	prompt := fmt.Sprintf(`Generate a detailed fix for the following issue:
Title: %s
Detail: %s
Severity: %s
Evidence: %v

Provide a specific, actionable recommendation with:
1. A clear description of the fix
2. The exact command(s) to execute (if applicable)
3. Whether this can be safely automated
4. The risk level of applying this fix

Return in JSON format:
{
  "description": "detailed explanation",
  "command": "command string",
  "auto_fix": true|false,
  "risk_level": "low|medium|high"
}

Only return the JSON without any additional text.`,
		finding.Title, finding.Detail, finding.Severity, finding.Evidence)

	// 调用OpenAI API。Call OpenAI API.
	resp, err := l.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParam{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: openai.F(" %s", prompt),
			},
		},
		Model:       openai.F(l.model),
		Temperature: openai.F(0.4),
	})

	if err != nil {
		logging.Logger.Errorf("LLM fix generation failed: %v", err)
		return models.Recommendation{}, errors.ErrLLMCallFailed
	}

	if len(resp.Choices) == 0 {
		logging.Logger.Error("No fix response from LLM")
		return models.Recommendation{}, errors.ErrLLMCallFailed
	}

	// 解析响应。Parse response.
	var recommendation models.Recommendation
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &recommendation); err != nil {
		logging.Logger.Errorf("Failed to parse fix recommendation: %v", err)
		return models.Recommendation{}, errors.ErrLLMCallFailed
	}

	return recommendation, nil
}

// Query 自然语言查询。Query natural language input.
func (l *llm) Query(ctx context.Context, input string) (string, error) {
	logging.Logger.Debugf("Processing natural language query: %s", input)

	resp, err := l.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParam{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: openai.F(" %s", input),
			},
		},
		Model:       openai.F(l.model),
		Temperature: openai.F(0.7),
	})

	if err != nil {
		logging.Logger.Errorf("LLM query failed: %v", err)
		return "", errors.ErrLLMCallFailed
	}

	if len(resp.Choices) == 0 {
		logging.Logger.Error("No query response from LLM")
		return "", errors.ErrLLMCallFailed
	}

	return resp.Choices[0].Message.Content, nil
}

// 辅助函数：格式化数据为字符串。Helper function: format data as string.
func formatData(data interface{}) string {
	if data == nil {
		return "No data available"
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting data: %v", err)
	}

	return string(jsonData)
}

// 辅助函数：格式化日志为字符串。Helper function: format logs as string.
func formatLogs(logs models.Logs) string {
	if len(logs) == 0 {
		return "No logs available"
	}

	logStr := ""
	for i, log := range logs {
		if i >= 10 { // 限制日志数量，避免提示过长。Limit log count to prevent long prompts.
			logStr += fmt.Sprintf("\n... and %d more logs", len(logs)-i)
			break
		}
		logStr += fmt.Sprintf("[%s] %s: %s\n", log.Timestamp.Format(time.RFC3339), log.Level, log.Message)
	}

	return logStr
}

//Personal.AI order the ending
