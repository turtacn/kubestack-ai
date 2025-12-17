package planning

import (
	"time"
)

// StepStatus represents the execution status of a step
type StepStatus string

const (
	StepStatusPending    StepStatus = "Pending"
	StepStatusRunning    StepStatus = "Running"
	StepStatusCompleted  StepStatus = "Completed"
	StepStatusFailed     StepStatus = "Failed"
	StepStatusSkipped    StepStatus = "Skipped"
	StepStatusRolledBack StepStatus = "RolledBack"
)

// StepType represents the type of step
type StepType string

const (
	StepTypeToolCall  StepType = "ToolCall"
	StepTypeLLMQuery  StepType = "LLMQuery"
	StepTypeCondition StepType = "Condition"
	StepTypeSubPlan   StepType = "SubPlan"
)

// RetryPolicy defines retry behavior for a step
type RetryPolicy struct {
	MaxRetries int `json:"max_retries"`
	BackoffMs  int `json:"backoff_ms"`
}

// ActionSpec defines the specific action to be executed
type ActionSpec struct {
	ToolName  string         `json:"tool_name,omitempty"`
	ToolArgs  map[string]any `json:"tool_args,omitempty"`
	Prompt    string         `json:"prompt,omitempty"`
	Condition string         `json:"condition,omitempty"`
}

// Step represents a single execution step in a plan
type Step struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        StepType       `json:"type"`
	DependsOn   []string       `json:"depends_on"`
	Action      ActionSpec     `json:"action"`
	Rollback    *ActionSpec    `json:"rollback,omitempty"`
	Timeout     time.Duration  `json:"timeout"`
	RetryPolicy *RetryPolicy   `json:"retry_policy,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Plan represents a complete execution plan
type Plan struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Steps       []Step            `json:"steps"`
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PlanStatus represents the overall status of a plan execution
type PlanStatus string

const (
	PlanStatusPending    PlanStatus = "Pending"
	PlanStatusRunning    PlanStatus = "Running"
	PlanStatusCompleted  PlanStatus = "Completed"
	PlanStatusFailed     PlanStatus = "Failed"
	PlanStatusRolledBack PlanStatus = "RolledBack"
	PlanStatusPaused     PlanStatus = "Paused"
	PlanStatusCancelled  PlanStatus = "Cancelled"
)

// StepState represents the execution state of a single step
type StepState struct {
	StepID      string     `json:"step_id"`
	Status      StepStatus `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Output      any        `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
	Attempts    int        `json:"attempts"`
}

// ExecutionState represents the complete execution state of a plan
type ExecutionState struct {
	PlanID      string               `json:"plan_id"`
	Status      PlanStatus           `json:"status"`
	StepStates  map[string]*StepState `json:"step_states"`
	StartedAt   time.Time            `json:"started_at"`
	CompletedAt *time.Time           `json:"completed_at,omitempty"`
	Error       string               `json:"error,omitempty"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
}

// NewExecutionState creates a new execution state for a plan
func NewExecutionState(planID string) *ExecutionState {
	return &ExecutionState{
		PlanID:     planID,
		Status:     PlanStatusPending,
		StepStates: make(map[string]*StepState),
		StartedAt:  time.Now(),
		Metadata:   make(map[string]any),
	}
}

// GetStepState returns the state of a specific step, creating it if needed
func (es *ExecutionState) GetStepState(stepID string) *StepState {
	if state, exists := es.StepStates[stepID]; exists {
		return state
	}
	state := &StepState{
		StepID:   stepID,
		Status:   StepStatusPending,
		Attempts: 0,
	}
	es.StepStates[stepID] = state
	return state
}

// MarkStepStarted marks a step as started
func (es *ExecutionState) MarkStepStarted(stepID string) {
	state := es.GetStepState(stepID)
	now := time.Now()
	state.Status = StepStatusRunning
	state.StartedAt = &now
	state.Attempts++
}

// MarkStepCompleted marks a step as completed
func (es *ExecutionState) MarkStepCompleted(stepID string, output any) {
	state := es.GetStepState(stepID)
	now := time.Now()
	state.Status = StepStatusCompleted
	state.CompletedAt = &now
	state.Output = output
	state.Error = ""
}

// MarkStepFailed marks a step as failed
func (es *ExecutionState) MarkStepFailed(stepID string, err error) {
	state := es.GetStepState(stepID)
	now := time.Now()
	state.Status = StepStatusFailed
	state.CompletedAt = &now
	if err != nil {
		state.Error = err.Error()
	}
}

// MarkStepSkipped marks a step as skipped
func (es *ExecutionState) MarkStepSkipped(stepID string) {
	state := es.GetStepState(stepID)
	state.Status = StepStatusSkipped
}

// MarkStepRolledBack marks a step as rolled back
func (es *ExecutionState) MarkStepRolledBack(stepID string) {
	state := es.GetStepState(stepID)
	state.Status = StepStatusRolledBack
}

// IsStepCompleted checks if a step is completed
func (es *ExecutionState) IsStepCompleted(stepID string) bool {
	state, exists := es.StepStates[stepID]
	return exists && state.Status == StepStatusCompleted
}

// HasFailedSteps checks if any step has failed
func (es *ExecutionState) HasFailedSteps() bool {
	for _, state := range es.StepStates {
		if state.Status == StepStatusFailed {
			return true
		}
	}
	return false
}

// GetCompletedSteps returns a list of completed step IDs
func (es *ExecutionState) GetCompletedSteps() []string {
	var completed []string
	for stepID, state := range es.StepStates {
		if state.Status == StepStatusCompleted {
			completed = append(completed, stepID)
		}
	}
	return completed
}
