// 版权所有 2025 Google LLC
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下网址获取许可证副本：
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，根据许可证分发的软件
// 均按"原样"提供，不附带任何明示或暗示的保证或条件。
// 有关许可证下特定语言的管辖权限和限制，请参阅许可证。

package llm

import (
	"context"
	"io"

	"github.com/turtacn/kubestack-ai/pkg/llm/chat"
)

// 对LLM的抽象
type Client interface {
	io.Closer

	// StartChat 与语言模型开始新的多轮对话。
	StartChat(systemPrompt, model string) chat.Chat

	// GenerateCompletion 为给定提示生成单个完成。
	GenerateCompletion(ctx context.Context, req *CompletionRequest) (CompletionResponse, error)

	// SetResponseSchema 设置响应的JSON Schema，用于结构化输出
	SetResponseSchema(schema *chat.Schema) error

	// ListModels 列出LLM中可用的模型。
	ListModels(ctx context.Context) ([]string, error)
}

// CompletionRequest 是为给定提示生成完成的请求。
type CompletionRequest struct {
	Model  string `json:"model,omitempty"`  // 模型名称
	Prompt string `json:"prompt,omitempty"` // 提示文本
}

// CompletionResponse 是 GenerateCompletion 方法的响应。
type CompletionResponse interface {
	Response() string   // 获取响应文本
	UsageMetadata() any // 获取使用元数据
}
