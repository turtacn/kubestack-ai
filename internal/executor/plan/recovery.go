package plan

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// PlanRecovery 计划恢复器
type PlanRecovery struct {
	store    PlanStore
	log      logger.Logger
	// We might need an executor or manager to resume execution,
	// but circular dependency might prevent importing execution package here.
	// So we might just define the logic to list and identify,
	// and the caller (Manager or Orchestrator) uses it.
}

func NewPlanRecovery(store PlanStore) *PlanRecovery {
	return &PlanRecovery{
		store: store,
		log:   logger.NewLogger("plan-recovery"),
	}
}

// RecoverOnStartup 启动时恢复
// Returns list of plans that need attention
func (r *PlanRecovery) RecoverOnStartup(ctx context.Context) ([]*models.ExecutionPlan, error) {
	r.log.Info("Scanning for incomplete execution plans...")
	plans, err := r.store.ListIncomplete(ctx)
	if err != nil {
		return nil, err
	}

	r.log.Infof("Found %d incomplete plans.", len(plans))
	// In a real implementation, we might try to lock them or separate them into buckets (resumable vs broken)
	return plans, nil
}
