package agent

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/nlp"
	ncontext "github.com/kubestack-ai/kubestack-ai/internal/nlp/context"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/intent"
)

// UserInput represents a user's input.
type UserInput struct {
	Text      string
	SessionID string
	UserID    string
}

// AgentResponse represents the agent's response.
type AgentResponse struct {
	Text   string
	Result interface{}
}

// Agent is the AI agent that processes user input.
type Agent struct {
	nlpProcessor *nlp.NLPProcessor
}

// NewAgent creates a new Agent.
func NewAgent(nlpProcessor *nlp.NLPProcessor) *Agent {
	return &Agent{
		nlpProcessor: nlpProcessor,
	}
}

// ProcessUserInput processes the user's input and returns a response.
func (a *Agent) ProcessUserInput(ctx context.Context, input *UserInput) (*AgentResponse, error) {
	// === 1. NLP Processing ===
	nlpResult, err := a.nlpProcessor.Process(ctx, &nlp.ProcessRequest{
		Text:      input.Text,
		SessionID: input.SessionID,
		UserID:    input.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("NLP processing failed: %w", err)
	}

	// === 2. Intent Routing ===
	handler, err := a.routeByIntent(nlpResult.Intent)
	if err != nil {
		return a.handleUnknownIntent(ctx, nlpResult)
	}

	// === 3. Task Execution ===
	taskReq := &TaskRequest{
		Intent:   nlpResult.Intent,
		Entities: nlpResult.Entities,
		Context:  nlpResult.Context,
		RawText:  input.Text,
	}

	taskResult, err := handler.Handle(ctx, taskReq)
	if err != nil {
		return nil, err
	}

	// === 4. Response Generation ===
	return &AgentResponse{
		Text:   taskResult.Message,
		Result: taskResult.Data,
	}, nil
}

func (a *Agent) routeByIntent(i *intent.Intent) (TaskHandler, error) {
	// Basic routing implementation
	switch i.Type {
	case intent.IntentDiagnose:
		return &DiagnoseHandler{}, nil
	case intent.IntentQuery:
		return &QueryHandler{}, nil
	case intent.IntentFix:
		return &FixHandler{}, nil
	case intent.IntentAlert:
		return &AlertHandler{}, nil
	case intent.IntentConfig:
		return &ConfigHandler{}, nil
	case intent.IntentExplain:
		return &ExplainHandler{}, nil
	case intent.IntentHelp:
		return &HelpHandler{}, nil
	default:
		return nil, fmt.Errorf("unknown intent type: %s", i.Type)
	}
}

func (a *Agent) handleUnknownIntent(ctx context.Context, res *nlp.ProcessResult) (*AgentResponse, error) {
	return &AgentResponse{
		Text: "Sorry, I didn't understand that. You can ask me to diagnose issues, check metrics, or help with configuration.",
	}, nil
}

// TaskRequest represents a request to execute a task.
type TaskRequest struct {
	Intent   *intent.Intent
	Entities []entity.Entity
	Context  *ncontext.ConversationContext
	RawText  string
}

// TaskResult represents the result of a task execution.
type TaskResult struct {
	Message string
	Data    interface{}
}

// TaskHandler is the interface for handling tasks.
type TaskHandler interface {
	Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error)
}

// -- Placeholder Handlers --

type DiagnoseHandler struct{}

func (h *DiagnoseHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "Starting diagnosis... (Mock)"}, nil
}

type QueryHandler struct{}

func (h *QueryHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "Querying metrics... (Mock)"}, nil
}

type FixHandler struct{}

func (h *FixHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "Executing fix... (Mock)"}, nil
}

type HelpHandler struct{}

func (h *HelpHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "I can help you diagnose, query, and fix infrastructure issues."}, nil
}

type AlertHandler struct{}

func (h *AlertHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "Configuring alert... (Mock)"}, nil
}

type ConfigHandler struct{}

func (h *ConfigHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "Updating config... (Mock)"}, nil
}

type ExplainHandler struct{}

func (h *ExplainHandler) Handle(ctx context.Context, req *TaskRequest) (*TaskResult, error) {
	return &TaskResult{Message: "Explaining... (Mock)"}, nil
}
