# Planning Engine Design Document

## Phase 2: Planning Engine Implementation

**Phase ID:** `P2`  
**Branch:** `feat/round6-phase2-planning-engine`  
**Dependencies:** `P1` (MemoryManager for plan state persistence)

---

## 1. Overview

The Planning Engine is a core component that enables KubeStack AI agents to decompose complex tasks into structured, executable plans. It supports DAG-based task dependency management, parallel execution, rollback mechanisms, and self-reflection capabilities.

### Key Features

- **DAG-based Planning**: Define tasks with dependencies and automatic topological sorting
- **Parallel Execution**: Automatically identify and execute independent steps in parallel
- **State Management**: Persistent tracking of execution state with pause/resume support
- **Rollback Mechanism**: Automatic rollback of completed steps on failure
- **Reflection Loop**: LLM-based post-execution evaluation and improvement suggestions
- **Retry Logic**: Configurable retry policies with exponential backoff

---

## 2. Architecture

### 2.1 Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                         Agent                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              PlanEngine                              │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────┐ │   │
│  │  │     DAG     │  │   Executor   │  │   State   │ │   │
│  │  │  Analysis   │  │  (Serial &   │  │  Manager  │ │   │
│  │  │             │  │  Parallel)   │  │           │ │   │
│  │  └─────────────┘  └──────────────┘  └───────────┘ │   │
│  │  ┌─────────────┐  ┌──────────────┐                │   │
│  │  │  Rollback   │  │  Reflection  │                │   │
│  │  │   Manager   │  │     Loop     │                │   │
│  │  └─────────────┘  └──────────────┘                │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Component Interactions

1. **Agent** creates a Plan (manually or via LLM)
2. **PlanEngine** validates the Plan and builds a DAG
3. **DAG** performs topological sorting and identifies parallel groups
4. **Executor** executes steps sequentially or in parallel
5. **StateManager** persists execution state after each step
6. **RollbackManager** reverses completed steps on failure
7. **ReflectionLoop** evaluates final results and suggests improvements

---

## 3. Data Structures

### 3.1 Core Types

#### StepType
```go
type StepType string

const (
    StepTypeToolCall   StepType = "ToolCall"   // Execute a tool
    StepTypeLLMQuery   StepType = "LLMQuery"   // Query LLM
    StepTypeCondition  StepType = "Condition"  // Evaluate condition
    StepTypeSubPlan    StepType = "SubPlan"    // Execute nested plan
)
```

#### StepStatus
```go
type StepStatus string

const (
    StepStatusPending    StepStatus = "Pending"
    StepStatusRunning    StepStatus = "Running"
    StepStatusCompleted  StepStatus = "Completed"
    StepStatusFailed     StepStatus = "Failed"
    StepStatusSkipped    StepStatus = "Skipped"
    StepStatusRolledBack StepStatus = "RolledBack"
)
```

#### Step
```go
type Step struct {
    ID          string        // Unique step identifier
    Name        string        // Human-readable name
    Type        StepType      // Step type
    DependsOn   []string      // List of step IDs this depends on
    Action      ActionSpec    // Action to execute
    Rollback    *ActionSpec   // Optional rollback action
    Timeout     time.Duration // Execution timeout
    RetryPolicy *RetryPolicy  // Retry configuration
}
```

#### ActionSpec
```go
type ActionSpec struct {
    ToolName   string         // Tool name (for ToolCall)
    ToolArgs   map[string]any // Tool arguments
    Prompt     string         // LLM prompt (for LLMQuery)
    Condition  string         // Condition expression (for Condition)
}
```

#### Plan
```go
type Plan struct {
    ID          string            // Unique plan identifier
    Name        string            // Plan name
    Description string            // Plan description
    Steps       []Step            // List of steps
    CreatedAt   time.Time         // Creation timestamp
    Metadata    map[string]string // Additional metadata
}
```

#### ExecutionState
```go
type ExecutionState struct {
    PlanID       string                  // Associated plan ID
    Status       PlanStatus              // Overall status
    StepStates   map[string]*StepState   // State of each step
    StartedAt    time.Time               // Execution start time
    CompletedAt  *time.Time              // Execution completion time
    Error        string                  // Error message if failed
}
```

---

## 4. Key Algorithms

### 4.1 DAG Topological Sort (Kahn's Algorithm)

```go
func (d *DAG) TopologicalSort() ([]string, error) {
    // 1. Initialize queue with all nodes having InDegree == 0
    queue := []string{}
    for id, node := range d.nodes {
        if node.InDegree == 0 {
            queue = append(queue, id)
        }
    }
    
    // 2. Process nodes
    sorted := []string{}
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]
        sorted = append(sorted, current)
        
        // 3. Reduce InDegree of dependents
        for _, dependent := range d.edges[current] {
            d.nodes[dependent].InDegree--
            if d.nodes[dependent].InDegree == 0 {
                queue = append(queue, dependent)
            }
        }
    }
    
    // 4. Check for cycles
    if len(sorted) != len(d.nodes) {
        return nil, errors.New("cyclic dependency detected")
    }
    
    return sorted, nil
}
```

### 4.2 Parallel Group Identification

```go
func (d *DAG) GetParallelGroups() [][]string {
    // 1. Get topological order
    sorted := d.TopologicalSort()
    
    // 2. Calculate level for each node (longest path from source)
    levels := make(map[string]int)
    for _, nodeID := range sorted {
        maxDepLevel := -1
        for depID, dependents := range d.edges {
            if contains(dependents, nodeID) {
                if levels[depID] > maxDepLevel {
                    maxDepLevel = levels[depID]
                }
            }
        }
        levels[nodeID] = maxDepLevel + 1
    }
    
    // 3. Group nodes by level
    groups := [][]string{}
    for level := 0; level <= maxLevel; level++ {
        group := []string{}
        for nodeID, nodeLevel := range levels {
            if nodeLevel == level {
                group = append(group, nodeID)
            }
        }
        groups = append(groups, group)
    }
    
    return groups
}
```

### 4.3 Step Execution with Retry

```go
func (e *DefaultStepExecutor) Execute(ctx context.Context, step *Step, input map[string]any) (any, error) {
    maxAttempts := 1
    if step.RetryPolicy != nil {
        maxAttempts = step.RetryPolicy.MaxRetries + 1
    }
    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        // Apply timeout if specified
        execCtx := ctx
        if step.Timeout > 0 {
            var cancel context.CancelFunc
            execCtx, cancel = context.WithTimeout(ctx, step.Timeout)
            defer cancel()
        }
        
        // Execute based on type
        result, err := e.executeByType(execCtx, step, input)
        if err == nil {
            return result, nil
        }
        
        // Retry logic
        if attempt < maxAttempts {
            backoff := time.Duration(step.RetryPolicy.BackoffMs) * time.Millisecond
            time.Sleep(backoff * time.Duration(attempt))
        }
    }
    
    return nil, fmt.Errorf("step execution failed after %d attempts", maxAttempts)
}
```

### 4.4 Rollback Mechanism

```go
func (r *RollbackManager) Rollback(ctx context.Context, plan *Plan, state *ExecutionState) error {
    // 1. Collect completed steps with rollback actions
    rollbackSteps := []Step{}
    for _, step := range plan.Steps {
        stepState := state.StepStates[step.ID]
        if stepState.Status == StepStatusCompleted && step.Rollback != nil {
            rollbackSteps = append(rollbackSteps, step)
        }
    }
    
    // 2. Reverse order (LIFO)
    for i := len(rollbackSteps) - 1; i >= 0; i-- {
        step := rollbackSteps[i]
        
        // 3. Execute rollback action
        err := r.RollbackStep(ctx, &step)
        if err != nil {
            // Log error but continue (best-effort rollback)
            log.Warnf("failed to rollback step %s: %v", step.ID, err)
        } else {
            state.StepStates[step.ID].Status = StepStatusRolledBack
        }
    }
    
    return nil
}
```

---

## 5. API Usage Examples

### 5.1 Creating a Plan Manually

```go
plan := planning.NewPlan("deploy-app", "Deploy Application", []planning.Step{
    {
        ID:   "build-image",
        Name: "Build Docker Image",
        Type: planning.StepTypeToolCall,
        Action: planning.ActionSpec{
            ToolName: "docker_build",
            ToolArgs: map[string]any{
                "dockerfile": "./Dockerfile",
                "tag":        "myapp:latest",
            },
        },
        Rollback: &planning.ActionSpec{
            ToolName: "docker_remove",
            ToolArgs: map[string]any{"image": "myapp:latest"},
        },
        Timeout: 5 * time.Minute,
    },
    {
        ID:        "push-image",
        Name:      "Push to Registry",
        Type:      planning.StepTypeToolCall,
        DependsOn: []string{"build-image"},
        Action: planning.ActionSpec{
            ToolName: "docker_push",
            ToolArgs: map[string]any{"image": "myapp:latest"},
        },
        RetryPolicy: &planning.RetryPolicy{
            MaxRetries: 3,
            BackoffMs:  1000,
        },
    },
    {
        ID:        "deploy-k8s",
        Name:      "Deploy to Kubernetes",
        Type:      planning.StepTypeToolCall,
        DependsOn: []string{"push-image"},
        Action: planning.ActionSpec{
            ToolName: "kubectl_apply",
            ToolArgs: map[string]any{"manifest": "deployment.yaml"},
        },
        Rollback: &planning.ActionSpec{
            ToolName: "kubectl_delete",
            ToolArgs: map[string]any{"manifest": "deployment.yaml"},
        },
    },
})

// Execute the plan
state, err := agent.ExecutePlan(ctx, plan)
if err != nil {
    log.Fatalf("Plan execution failed: %v", err)
}

log.Printf("Plan completed: %s", state.Status)
```

### 5.2 Creating a Plan from Natural Language

```go
goal := "Deploy a new version of the API service with zero downtime"
plan, err := agent.CreatePlanFromGoal(ctx, goal)
if err != nil {
    log.Fatalf("Failed to create plan: %v", err)
}

// Execute the generated plan
state, err := agent.ExecutePlan(ctx, plan)
```

### 5.3 Monitoring Execution State

```go
// Get current state
state, err := agent.GetPlanState("deploy-app")
if err != nil {
    log.Fatalf("Failed to get state: %v", err)
}

// Check individual step states
for stepID, stepState := range state.StepStates {
    log.Printf("Step %s: %s", stepID, stepState.Status)
    if stepState.Error != "" {
        log.Printf("  Error: %s", stepState.Error)
    }
}
```

### 5.4 Cancelling a Running Plan

```go
err := agent.CancelPlan("deploy-app")
if err != nil {
    log.Printf("Failed to cancel plan: %v", err)
}
```

---

## 6. Configuration

### 6.1 PlanEngineConfig

```go
type PlanEngineConfig struct {
    MaxParallel       int  // Maximum parallel step execution (default: 5)
    EnableReflection  bool // Enable post-execution reflection (default: false)
    EnableRollback    bool // Enable automatic rollback on failure (default: true)
}
```

### 6.2 Default Configuration

```go
func DefaultPlanEngineConfig() PlanEngineConfig {
    return PlanEngineConfig{
        MaxParallel:      5,
        EnableReflection: false,
        EnableRollback:   true,
    }
}
```

---

## 7. Testing Strategy

### 7.1 Unit Tests

- **plan_test.go**: Plan validation, step management
- **dag_test.go**: Topological sorting, cycle detection, parallel groups
- **executor_test.go**: Step execution, retry logic, timeout handling
- **engine_test.go**: Full plan execution, state management
- **rollback_test.go**: Rollback ordering, best-effort rollback
- **state_test.go**: State persistence and retrieval

### 7.2 Test Coverage

All unit tests passed with the following coverage:
- Total test cases: 61
- All tests PASS
- Coverage includes:
  - Happy paths
  - Error scenarios
  - Edge cases (empty plans, circular dependencies, etc.)

### 7.3 Key Test Scenarios

1. **Linear Execution**: Steps executed in order
2. **Parallel Execution**: Independent steps run concurrently
3. **Failure Handling**: Proper state update on step failure
4. **Rollback**: Steps rolled back in reverse order
5. **Retry Logic**: Failed steps retried with backoff
6. **Cancellation**: Long-running plans can be cancelled
7. **State Persistence**: State correctly saved and retrieved

---

## 8. Performance Characteristics

### 8.1 Time Complexity

- **DAG Topological Sort**: O(V + E) where V = vertices (steps), E = edges (dependencies)
- **Parallel Group Calculation**: O(V + E)
- **Step Execution**: O(N) for N steps (with parallelization reducing wall-clock time)

### 8.2 Space Complexity

- **DAG Storage**: O(V + E)
- **Execution State**: O(V) for storing state of each step

### 8.3 Scalability

- Tested with plans up to 20 steps
- Parallel execution limited by MaxParallel config (default: 5)
- State persistence uses memory or disk-backed storage

---

## 9. Integration Points

### 9.1 Memory System Integration

The Planning Engine integrates with the Memory system (Phase 1) for:
- Persisting execution state across restarts
- Storing plan history for analysis
- Enabling resume from checkpoint

```go
// PersistentStateStore uses Memory system
stateStore := planning.NewPersistentStateStore(memoryStore)
```

### 9.2 Agent Integration

The Agent is extended with planning capabilities:

```go
type Agent struct {
    // ... existing fields
    planEngine *planning.PlanEngine
}

func (a *Agent) ExecutePlan(ctx context.Context, plan *planning.Plan) (*planning.ExecutionState, error)
func (a *Agent) CreatePlanFromGoal(ctx context.Context, goal string) (*planning.Plan, error)
func (a *Agent) GetPlanState(planID string) (*planning.ExecutionState, error)
func (a *Agent) CancelPlan(planID string) error
```

---

## 10. Future Enhancements

### 10.1 Conditional Branching

Support for conditional step execution based on previous step outputs:

```go
{
    ID:   "check-health",
    Type: StepTypeCondition,
    Action: ActionSpec{
        Condition: "previous_step.status == 'healthy'",
    },
}
```

### 10.2 Dynamic Plan Modification

Allow plans to be modified during execution based on intermediate results.

### 10.3 Distributed Execution

Support for distributing step execution across multiple agents or machines.

### 10.4 Plan Templates

Pre-defined reusable plan templates for common workflows.

### 10.5 Visual Plan Editor

Web-based UI for creating and visualizing plan DAGs.

---

## 11. Known Limitations

1. **SubPlan Execution**: Currently placeholder implementation
2. **Condition Evaluation**: Simple string-based evaluation (only "true"/"false")
3. **Parallel Execution Cancellation**: Cancelling a plan may not immediately stop all parallel steps
4. **State Size**: Large plans with many steps may result in large state objects

---

## 12. References

- [Kahn's Algorithm](https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm)
- [DAG-based Task Scheduling](https://en.wikipedia.org/wiki/Directed_acyclic_graph)
- [Retry Patterns](https://learn.microsoft.com/en-us/azure/architecture/patterns/retry)

---

## Changelog

- **2025-12-17**: Initial implementation of Planning Engine (Phase 2)
  - Implemented all core components: types, DAG, executor, engine, state, rollback, reflection
  - Added comprehensive unit tests (61 test cases, all passing)
  - Integrated with Agent for seamless planning capabilities
