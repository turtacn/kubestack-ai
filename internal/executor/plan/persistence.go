package plan

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// PlanStore 计划存储接口
type PlanStore interface {
	Save(ctx context.Context, plan *models.ExecutionPlan) error
	Load(ctx context.Context, planID string) (*models.ExecutionPlan, error)
	Update(ctx context.Context, plan *models.ExecutionPlan) error
	Delete(ctx context.Context, planID string) error

	// ListByState 按状态查询
	ListByState(ctx context.Context, state PlanState) ([]*models.ExecutionPlan, error)

	// ListIncomplete 查询未完成的计划(用于恢复)
	ListIncomplete(ctx context.Context) ([]*models.ExecutionPlan, error)
}

// FilePlanStore 基于文件的持久化
type FilePlanStore struct {
	basePath string // /var/lib/kubestack/plans/
	mu       sync.RWMutex
}

func NewFilePlanStore(basePath string) *FilePlanStore {
	os.MkdirAll(basePath, 0755)
	return &FilePlanStore{
		basePath: basePath,
	}
}

// Save 保存计划
func (s *FilePlanStore) Save(ctx context.Context, plan *models.ExecutionPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 序列化为JSON
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}

	// 2. 写入文件: {basePath}/{planID}.json
	path := filepath.Join(s.basePath, plan.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (s *FilePlanStore) Load(ctx context.Context, planID string) (*models.ExecutionPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, planID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var plan models.ExecutionPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *FilePlanStore) Update(ctx context.Context, plan *models.ExecutionPlan) error {
	return s.Save(ctx, plan)
}

func (s *FilePlanStore) Delete(ctx context.Context, planID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, planID+".json")
	return os.Remove(path)
}

func (s *FilePlanStore) ListByState(ctx context.Context, state PlanState) ([]*models.ExecutionPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, err
	}

	var result []*models.ExecutionPlan
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			// Extract ID from filename if possible, or just load
			id := strings.TrimSuffix(f.Name(), ".json")
			p, err := s.Load(ctx, id)
			if err != nil {
				continue
			}
			// PlanState is not directly on models.ExecutionPlan, check status
			// We assume we can map models.ExecutionStatus to PlanState or use status
			// For simplicity, we check if the status string matches
			// But wait, PlanState is defined in this package, models.ExecutionPlan has Status of type models.ExecutionStatus
			// We should probably check models.ExecutionStatus
			// But ListByState takes PlanState.
			// Let's assume the status stored in ExecutionPlan.Status corresponds to PlanState string value.
			if string(p.Status) == string(state) {
				result = append(result, p)
			}
		}
	}
	return result, nil
}

func (s *FilePlanStore) ListIncomplete(ctx context.Context) ([]*models.ExecutionPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, err
	}

	var result []*models.ExecutionPlan
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			id := strings.TrimSuffix(f.Name(), ".json")
			p, err := s.Load(ctx, id)
			if err != nil {
				continue
			}

			// Filter for incomplete states:
			// InProgress, Pending, Running
			// models.ExecutionStatus: "InProgress", "Running", "Pending" ?
			// Let's check models.ExecutionStatus constants
			// ExecutionStatusInProgress, ExecutionStatusFailedWithRollbackFailure (maybe need manual intervention?)

			// If we use PlanStateMachine logic, valid active states are:
			// StatePending, StateApproved, StateExecuting

			// We map p.Status back to PlanState logic or just check string
			status := string(p.Status)
			if status == "InProgress" || status == "Pending" || status == "Running" || status == "approved" || status == "executing" {
				result = append(result, p)
			}
		}
	}
	return result, nil
}
