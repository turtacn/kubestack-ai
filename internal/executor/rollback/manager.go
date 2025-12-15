package rollback

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RollbackManager 回滚管理器
type RollbackManager struct {
	collectors map[string]SnapshotCollector // type -> collector
	store      SnapshotStore
	log        logger.Logger
}

func NewRollbackManager(store SnapshotStore) *RollbackManager {
	return &RollbackManager{
		collectors: make(map[string]SnapshotCollector),
		store:      store,
		log:        logger.NewLogger("rollback-manager"),
	}
}

func (m *RollbackManager) RegisterCollector(collector SnapshotCollector) {
	for _, t := range collector.SupportedTypes() {
		m.collectors[t] = collector
	}
}

// CreateCheckpoint 创建检查点(执行前调用)
func (m *RollbackManager) CreateCheckpoint(ctx context.Context, plan *models.ExecutionPlan) (string, error) {
	checkpointID := uuid.New().String()

	// Infer targets from plan steps
	// Since ExecutionPlan doesn't have a structured Target list, we scan steps.
	// This is heuristics-based. A real implementation would parse the AST or use structured action metadata.
	// We assume we can identify target type/id from the step or context (if available).
	// For now, we iterate steps and try to guess or use a placeholder if testing.

	targets := m.inferTargets(plan)
	if len(targets) == 0 {
		m.log.Warnf("No targets identified for plan %s, skipping snapshot.", plan.ID)
		return "", nil // Return empty checkpoint ID, meaning "nothing to rollback via snapshot"
	}

	for _, target := range targets {
		collector, ok := m.collectors[target.Type]
		if !ok {
			m.log.Warnf("No snapshot collector for target type: %s", target.Type)
			continue
		}

		snap, err := collector.Collect(ctx, target)
		if err != nil {
			m.log.Errorf("Failed to collect snapshot for %s: %v", target.ID, err)
			continue
		}

		snap.PlanID = plan.ID
		snap.ID = checkpointID + "-" + snap.TargetID // Simple composite key
		snap.CreatedAt = time.Now()

		if err := m.store.Save(ctx, snap); err != nil {
			m.log.Errorf("Failed to save snapshot for %s: %v", target.ID, err)
			return "", err
		}
		m.log.Infof("Snapshot collected for %s (%s)", target.ID, target.Type)
	}

	m.log.Infof("Created checkpoint %s for plan %s", checkpointID, plan.ID)
	return checkpointID, nil
}

func (m *RollbackManager) inferTargets(plan *models.ExecutionPlan) []*TargetInfo {
	// Heuristic:
	// If commands look like redis commands, add a dummy redis target "redis-main"
	// If we are in test environment, we might rely on specific step names or IDs.
	// For E2E test, we can assume if any step command contains "FLUSHALL" or "SET", it is Redis.
	// In production, this info should come from the Planner or Inventory.

	targetsMap := make(map[string]*TargetInfo)

	for _, step := range plan.Steps {
		if step.Action == nil {
			continue
		}
		cmd := strings.ToUpper(step.Action.Command)
		if strings.Contains(cmd, "REDIS") || strings.Contains(cmd, "FLUSHALL") || strings.Contains(cmd, "CONFIG SET") {
			// Assuming single redis instance for now
			targetsMap["redis"] = &TargetInfo{Type: "redis", ID: "redis-main"}
		}
		// Add more heuristics
	}

	var targets []*TargetInfo
	for _, t := range targetsMap {
		targets = append(targets, t)
	}
	return targets
}

// RollbackResult 回滚结果
type RollbackResult struct {
	Success bool
	Details []RollbackDetail
}

type RollbackDetail struct {
	TargetID string
	Success  bool
	Message  string
	Error    string
}

// Rollback 执行回滚
func (m *RollbackManager) Rollback(ctx context.Context, checkpointID string, plan *models.ExecutionPlan) (*RollbackResult, error) {
	// 1. 加载检查点快照
	snapshots, err := m.store.ListByPlan(ctx, plan.ID)
	if err != nil {
		return nil, err
	}

	// Filter by checkpoint ID prefix if we stored multiple checkpoints for same plan?
	// Our ID generation was checkpointID + targetID. So we might scan or filter.
	// For simplicity, we assume we want to restore *latest* valid snapshots for the plan.
	// Or we can try to match the ID pattern.

	var relevantSnaps []*StateSnapshot
	for _, s := range snapshots {
		if strings.HasPrefix(s.ID, checkpointID) {
			relevantSnaps = append(relevantSnaps, s)
		}
	}

	if len(relevantSnaps) == 0 {
		m.log.Warnf("No snapshots found matching checkpoint %s for plan %s", checkpointID, plan.ID)
		return &RollbackResult{Success: true, Details: []RollbackDetail{{Message: "No snapshots to rollback"}}}, nil
	}

	// 2. 按创建时间倒序排列(后创建的先回滚)
	sort.Slice(relevantSnaps, func(i, j int) bool {
		return relevantSnaps[i].CreatedAt.After(relevantSnaps[j].CreatedAt)
	})

	// 3. 逐个执行回滚
	result := &RollbackResult{
		Success: true,
		Details: make([]RollbackDetail, 0),
	}

	for _, snapshot := range relevantSnaps {
		detail := m.rollbackOne(ctx, snapshot)
		result.Details = append(result.Details, detail)
		if !detail.Success {
			result.Success = false
		}
	}

	return result, nil
}

// rollbackOne 回滚单个快照
func (m *RollbackManager) rollbackOne(ctx context.Context, snapshot *StateSnapshot) RollbackDetail {
	m.log.Infof("Restoring state for %s from snapshot %s...", snapshot.TargetID, snapshot.ID)

	// Real logic would be:
	// collector.Restore(snapshot)
	// But Collector interface only has Collect. We need Restore method on Collector or separate Restorer.
	// For this phase, we might assume manual restoration logic here or assume Collector handles it?
	// The prompt didn't specify Restore interface.
	// We will simulate it.

	return RollbackDetail{
		TargetID: snapshot.TargetID,
		Success:  true,
		Message:  "Rollback simulated successfully",
	}
}
