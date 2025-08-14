package chat

import (
	"context"
	"fmt"
	"iter"
)

// Chat 是与语言模型的活跃对话。
// 消息被发送和接收，并添加到对话历史中。
type Chat interface {
	// Send 向对话添加用户消息，并从LLM获取响应。
	// 注意，此方法会自动更新Chat的状态，
	// 您无需"重放"LLM的任何消息。
	Send(ctx context.Context, contents ...*Message) (ChatResponse, error)

	// SendStreaming 是 Send 的流式版本。
	SendStreaming(ctx context.Context, contents ...*Message) (ChatResponseIterator, error)

	// SetFunctionDefinitions 配置可供LLM使用的工具（函数）集合
	// 用于函数调用。
	SetFunctionDefinitions(functionDefinitions []*FunctionDefinition) error

	// IsRetryableError 如果错误可重试则返回 true。
	IsRetryableError(error) bool

	// Initialize 用之前的对话历史初始化对话。
	Initialize(messages []*Message) error
}

// ChatResponse 是LLM的通用对话响应。
type ChatResponse interface {
	UsageMetadata() any // 使用元数据

	// Candidates 是LLM的一组候选响应。
	// LLM可能返回多个候选，我们可以选择最佳的一个。
	Candidates() []Candidate
}

// ChatResponseIterator 是LLM的流式对话响应。
type ChatResponseIterator iter.Seq2[ChatResponse, error]

// Candidate 是LLM的一组候选响应中的一个。
type Candidate interface {
	// String 返回候选的字符串表示。
	fmt.Stringer

	// Parts 返回候选的部分。
	Parts() []Part
}

// Part 是LLM候选响应的一部分。
// 它可以是文本响应或函数调用。
// 响应可能包含多个部分，
// 例如文本响应和函数调用，
// 其中文本响应是"我需要做必要的事情"，
// 然后函数调用是"do_necessary"。
type Part interface {
	// AsText 返回部分的文本。
	// 如果part不是文本，则返回 ("", false)
	AsText() (string, bool)

	// AsFunctionCalls 返回part的函数调用。
	// 如果part不是函数调用，则返回 (nil, false)
	AsFunctionCalls() ([]FunctionCall, bool)
}
