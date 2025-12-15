package plan

import (
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// PlanState 计划状态
type PlanState string

const (
	StatePending    PlanState = "pending"     // 待审批
	StateApproved   PlanState = "approved"    // 已批准
	StateExecuting  PlanState = "executing"   // 执行中
	StateCompleted  PlanState = "completed"   // 已完成
	StateFailed     PlanState = "failed"      // 执行失败
	StateRolledBack PlanState = "rolled_back" // 已回滚
	StateCancelled  PlanState = "cancelled"   // 已取消
)

// StateTransition 状态转换
type StateTransition struct {
	From  PlanState
	To    PlanState
	Event string
}

// 合法状态转换表
var validTransitions = []StateTransition{
	{StatePending, StateApproved, "approve"},
	{StatePending, StateCancelled, "cancel"},
	{StateApproved, StateExecuting, "start"},
	{StateApproved, StateCancelled, "cancel"},
	{StateExecuting, StateCompleted, "complete"},
	{StateExecuting, StateFailed, "fail"},
	{StateFailed, StateRolledBack, "rollback"},
	{StateFailed, StatePending, "retry"}, // 重试返回待审批
}

// PlanStateMachine 状态机
type PlanStateMachine struct {
	plan      *models.ExecutionPlan
	listeners []StateChangeListener
	log       logger.Logger
	// We store the state in a separate map or field if ExecutionPlan doesn't support it yet
	// But ExecutionPlan has 'Status' field which is ExecutionStatus enum.
	// We might need to map PlanState to ExecutionStatus or update ExecutionPlan model.
	// For now we assume we map it to string or use a custom field in memory.
	currentState PlanState
}

func NewStateMachine(plan *models.ExecutionPlan) *PlanStateMachine {
	return &PlanStateMachine{
		plan:         plan,
		log:          logger.NewLogger("plan-statemachine"),
		currentState: StatePending, // Default start state
	}
}

// Transition 执行状态转换
func (m *PlanStateMachine) Transition(event string) error {
	current := m.currentState

	// 1. 查找合法转换
	var targetState PlanState
	found := false
	for _, t := range validTransitions {
		if t.From == current && t.Event == event {
			targetState = t.To
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("invalid transition: %s --%s--> ?", current, event)
	}

	// 2. 执行转换
	oldState := m.currentState
	m.currentState = targetState
	// m.plan.UpdatedAt = time.Now() // If field exists

	// Update the underlying plan status if possible
	// Mapping PlanState to models.ExecutionStatus
	// models.ExecutionStatus is string based, so we can cast or map.

	// 3. 通知监听器
	for _, l := range m.listeners {
		l.OnStateChange(m.plan, oldState, targetState, event)
	}

	m.log.Infof("Plan %s transition: %s -> %s (Event: %s)", m.plan.ID, oldState, targetState, event)

	return nil
}

// CurrentState returns current state
func (m *PlanStateMachine) CurrentState() PlanState {
	return m.currentState
}

// StateChangeListener 状态变更监听器
type StateChangeListener interface {
	OnStateChange(plan *models.ExecutionPlan, from, to PlanState, event string)
}
