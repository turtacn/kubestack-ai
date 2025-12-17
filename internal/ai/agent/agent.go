package agent

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/memory"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp"
	ncontext "github.com/kubestack-ai/kubestack-ai/internal/nlp/context"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/intent"
	"github.com/kubestack-ai/kubestack-ai/internal/planning"
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
	nlpProcessor  *nlp.NLPProcessor
	memoryManager *memory.MemoryManager
	planEngine    *planning.PlanEngine
	llmClient     planning.LLMClient
}

// NewAgent creates a new Agent.
func NewAgent(nlpProcessor *nlp.NLPProcessor, memoryManager *memory.MemoryManager) *Agent {
	return &Agent{
		nlpProcessor:  nlpProcessor,
		memoryManager: memoryManager,
	}
}

// NewAgentWithPlanning creates a new Agent with planning capabilities.
func NewAgentWithPlanning(nlpProcessor *nlp.NLPProcessor, memoryManager *memory.MemoryManager, planEngine *planning.PlanEngine, llmClient planning.LLMClient) *Agent {
	return &Agent{
		nlpProcessor:  nlpProcessor,
		memoryManager: memoryManager,
		planEngine:    planEngine,
		llmClient:     llmClient,
	}
}

// ProcessUserInput processes the user's input and returns a response.
func (a *Agent) ProcessUserInput(ctx context.Context, input *UserInput) (*AgentResponse, error) {
	// === 0. Memory Management ===
	if a.memoryManager != nil {
		if err := a.memoryManager.LoadSession(input.SessionID); err != nil {
			// If session doesn't exist yet, that's OK
		}

		userEntry := memory.MemoryEntry{
			Role:    "user",
			Content: input.Text,
		}
		if err := a.memoryManager.RecordMessage(input.SessionID, userEntry); err != nil {
			return nil, fmt.Errorf("failed to record user message: %w", err)
		}
	}

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

	// === 4. Record Assistant Response ===
	if a.memoryManager != nil {
		assistantEntry := memory.MemoryEntry{
			Role:    "assistant",
			Content: taskResult.Message,
		}
		if err := a.memoryManager.RecordMessage(input.SessionID, assistantEntry); err != nil {
			return nil, fmt.Errorf("failed to record assistant message: %w", err)
		}
	}

	// === 5. Response Generation ===
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

// GetConversationHistory retrieves conversation history for a session
func (a *Agent) GetConversationHistory(sessionID string, maxTokens int) ([]memory.MemoryEntry, error) {
	if a.memoryManager == nil {
		return []memory.MemoryEntry{}, nil
	}
	return a.memoryManager.GetContext(sessionID, maxTokens)
}

// ClearSession clears the working memory for a session
func (a *Agent) ClearSession() {
	if a.memoryManager != nil {
		a.memoryManager.ClearWorking()
	}
}

// Close closes the agent and its resources
func (a *Agent) Close() error {
	if a.memoryManager != nil {
		return a.memoryManager.Close()
	}
	return nil
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

// ExecutePlan executes a plan and records it in memory
func (a *Agent) ExecutePlan(ctx context.Context, plan *planning.Plan) (*planning.ExecutionState, error) {
	if a.planEngine == nil {
		return nil, fmt.Errorf("plan engine not initialized")
	}

	// Execute the plan
	state, err := a.planEngine.ExecutePlan(ctx, plan)

	// Record execution in memory if available
	if a.memoryManager != nil {
		entry := memory.MemoryEntry{
			Role:    "system",
			Content: fmt.Sprintf("Executed plan: %s (Status: %s)", plan.Name, state.Status),
		}
		a.memoryManager.RecordMessage("system", entry)
	}

	return state, err
}

// CreatePlanFromGoal converts a natural language goal into a structured plan
func (a *Agent) CreatePlanFromGoal(ctx context.Context, goal string) (*planning.Plan, error) {
	if a.llmClient == nil {
		return nil, fmt.Errorf("LLM client not initialized")
	}

	prompt := fmt.Sprintf(`Convert the following goal into a structured execution plan.
Goal: %s

Create a plan with specific steps. Each step should have:
- A unique ID
- A descriptive name
- A type (ToolCall, LLMQuery, Condition)
- Any dependencies on other steps
- The action to perform

Respond with a simple list of steps that can be executed in order.`, goal)

	response, err := a.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// For now, create a simple mock plan
	// In a real implementation, this would parse the LLM response
	plan := planning.NewPlan(
		fmt.Sprintf("plan-%d", len(response)),
		"Generated Plan",
		[]planning.Step{
			{
				ID:   "step1",
				Name: "Initial Step",
				Type: planning.StepTypeLLMQuery,
				Action: planning.ActionSpec{
					Prompt: goal,
				},
			},
		},
	)
	plan.Description = goal

	return plan, nil
}

// GetPlanState retrieves the execution state of a plan
func (a *Agent) GetPlanState(planID string) (*planning.ExecutionState, error) {
	if a.planEngine == nil {
		return nil, fmt.Errorf("plan engine not initialized")
	}
	return a.planEngine.GetState(planID)
}

// CancelPlan cancels an executing plan
func (a *Agent) CancelPlan(planID string) error {
	if a.planEngine == nil {
		return fmt.Errorf("plan engine not initialized")
	}
	return a.planEngine.CancelPlan(planID)
}
