package llm

import (
	"context"
	"errors"
	"fmt"
	"os"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/turtacn/kubestack-ai/pkg/llm/chat"
	"k8s.io/klog/v2"
)

// 包级环境变量存储（OpenAI环境）
var (
	openAIAPIKey   string // OpenAI API密钥
	openAIEndpoint string // OpenAI端点
	openAIAPIBase  string // OpenAI API基础URL
	openAIModel    string // OpenAI模型
)

// InitOpenai 读取并缓存OpenAI环境变量：
//   - OPENAI_API_KEY, OPENAI_ENDPOINT, OPENAI_API_BASE, OPENAI_MODEL
//
// 这些作为默认值；模型可以通过Cobra --model标志被覆盖。
// 加载环境值后，注册OpenAI提供程序工厂。
func InitOpenai() {
	// 加载环境变量
	openAIAPIKey = os.Getenv("OPENAI_API_KEY")
	openAIEndpoint = os.Getenv("OPENAI_ENDPOINT")
	openAIAPIBase = os.Getenv("OPENAI_API_BASE")
	openAIModel = os.Getenv("OPENAI_MODEL")

	// 将"openai"注册为提供程序ID
	if err := RegisterProvider("openai", newOpenAIClientFactory); err != nil {
		klog.Fatalf("注册openai提供程序失败：%v", err)
	}
}

// OpenAIClient 为OpenAI模型实现llm.Client接口。
type OpenAIClient struct {
	client openai.Client // OpenAI客户端
}

// 确保OpenAIClient实现Client接口。
var _ Client = &OpenAIClient{}

// NewOpenAIClient creates a new client for interacting with OpenAI.
// Supports custom HTTP client (e.g., for skipping SSL verification).
func NewOpenAIClient(ctx context.Context, opts ClientOptions) (*OpenAIClient, error) {
	// Get API key from loaded env var
	apiKey := openAIAPIKey
	if apiKey == "" {
		return nil, errors.New("OpenAI API key not found. Set via OPENAI_API_KEY env var")
	}

	// Set options for client creation
	options := []option.RequestOption{option.WithAPIKey(apiKey)}

	// Check for custom endpoint or API base URL
	baseURL := openAIEndpoint
	if baseURL == "" {
		baseURL = openAIAPIBase
	}

	if baseURL != "" {
		klog.Infof("Using custom OpenAI base URL: %s", baseURL)
		options = append(options, option.WithBaseURL(baseURL))
	}

	// Support custom HTTP client (e.g., skip SSL verification)
	httpClient := createCustomHTTPClient(opts)
	options = append(options, option.WithHTTPClient(httpClient))

	return &OpenAIClient{
		client: openai.NewClient(options...),
	}, nil
}

// Close cleans up any resources used by the client.
func (c *OpenAIClient) Close() error {
	// No specific cleanup needed for the OpenAI client currently.
	return nil
}

// StartChat starts a new chat session.
func (c *OpenAIClient) StartChat(systemPrompt, model string) chat.Chat {
	// Get the model to use for this chat
	selectedModel := getOpenAIModel(model)

	klog.V(1).Infof("Starting new OpenAI chat session with model: %s", selectedModel)

	// Initialize history with system prompt if provided
	history := []openai.ChatCompletionMessageParamUnion{}
	if systemPrompt != "" {
		history = append(history, openai.SystemMessage(systemPrompt))
	}

	return &chat.OpenAIChatSession{
		Client:  c.client,
		History: history,
		Model:   selectedModel,
		// functionDefinitions and tools will be set later via SetFunctionDefinitions
	}
}

// simpleCompletionResponse is a basic implementation of CompletionResponse.
type simpleCompletionResponse struct {
	content string
}

// Response returns the completion content.
func (r *simpleCompletionResponse) Response() string {
	return r.content
}

// UsageMetadata returns nil for now.
func (r *simpleCompletionResponse) UsageMetadata() any {
	return nil
}

// GenerateCompletion sends a completion request to the OpenAI API.
func (c *OpenAIClient) GenerateCompletion(ctx context.Context, req *CompletionRequest) (CompletionResponse, error) {
	klog.Infof("OpenAI GenerateCompletion called with model: %s", req.Model)
	klog.V(1).Infof("Prompt:\n%s", req.Prompt)

	// Use the Chat Completions API with the new v1.0.0 API
	completion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(req.Model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate OpenAI completion: %w", err)
	}

	// Check if there are choices and a message
	if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == "" {
		return nil, errors.New("received an empty response from OpenAI")
	}

	// Return the content of the first choice
	resp := &simpleCompletionResponse{
		content: completion.Choices[0].Message.Content,
	}

	return resp, nil
}

// SetResponseSchema is not implemented yet.
func (c *OpenAIClient) SetResponseSchema(schema *chat.Schema) error {
	klog.Warning("OpenAIClient.SetResponseSchema is not implemented yet")
	return nil
}

// ListModels returns a slice of strings with model IDs.
// Note: This may not work with all OpenAI-compatible providers if they don't fully implement
// the Models.List endpoint or return data in a different format.
func (c *OpenAIClient) ListModels(ctx context.Context) ([]string, error) {
	res, err := c.client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing models from OpenAI: %w", err)
	}

	modelIDs := make([]string, 0, len(res.Data))
	for _, model := range res.Data {
		modelIDs = append(modelIDs, model.ID)
	}

	return modelIDs, nil
}

// newOpenAIClientFactory is the factory function for creating OpenAI clients.
func newOpenAIClientFactory(ctx context.Context, opts ClientOptions) (Client, error) {
	return NewOpenAIClient(ctx, opts)
}

// getOpenAIModel returns the appropriate model based on configuration and explicitly provided model name
func getOpenAIModel(model string) string {
	// If explicit model is provided, use it
	if model != "" {
		klog.V(2).Infof("Using explicitly provided model: %s", model)
		return model
	}

	// Check configuration
	configModel := openAIModel
	if configModel != "" {
		klog.V(1).Infof("Using model from config: %s", configModel)
		return configModel
	}

	// Default model as fallback
	klog.V(2).Info("No model specified, defaulting to gpt-4.1")
	return "gpt-4.1"
}
