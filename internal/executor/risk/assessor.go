package risk

import (
	"context"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RiskAssessor 风险评估器
type RiskAssessor struct {
	rules      []RiskRule      // 评估规则列表
	thresholds *RiskThresholds // 阈值配置
	log        logger.Logger
}

// RiskThresholds 阈值配置
type RiskThresholds struct {
	MediumThreshold   int // 中风险分数阈值, 默认30
	HighThreshold     int // 高风险分数阈值, 默认60
	CriticalThreshold int // 严重风险阈值, 默认90
	AutoConfirmBelow  int // 低于此分数自动确认, 默认20
}

// DefaultThresholds 返回默认阈值
func DefaultThresholds() *RiskThresholds {
	return &RiskThresholds{
		MediumThreshold:   30,
		HighThreshold:     60,
		CriticalThreshold: 90,
		AutoConfirmBelow:  20,
	}
}

// NewRiskAssessor 构造函数
// 默认加载内置规则
func NewRiskAssessor(thresholds *RiskThresholds) *RiskAssessor {
	if thresholds == nil {
		thresholds = DefaultThresholds()
	}
	return &RiskAssessor{
		rules:      BuiltinRules(),
		thresholds: thresholds,
		log:        logger.NewLogger("risk-assessor"),
	}
}

// Assess 执行风险评估
func (a *RiskAssessor) Assess(ctx context.Context, plan *models.ExecutionPlan) (*RiskAssessmentResult, error) {
	result := &RiskAssessmentResult{
		Reasons:     make([]string, 0),
		Mitigations: make([]string, 0),
	}

	// 1. 遍历所有规则，累加分数
	for _, rule := range a.rules {
		if rule.Match(plan) {
			ruleResult := rule.Evaluate(plan)
			result.Score += ruleResult.Score
			result.Reasons = append(result.Reasons, ruleResult.Reason)
			if ruleResult.Mitigation != "" {
				result.Mitigations = append(result.Mitigations, ruleResult.Mitigation)
			}
		}
	}

	// 2. 根据分数确定等级
	result.Level = a.scoreToLevel(result.Score)

	// 3. 设置确认/审批要求
	result.RequiresConfirm = result.Score >= a.thresholds.AutoConfirmBelow
	result.RequiresApproval = result.Level >= RiskLevelCritical

	// 4. 评估影响范围 (Simplified estimation)
	result.EstimatedImpact = a.estimateImpact(plan, result.Level)

	return result, nil
}

func (a *RiskAssessor) scoreToLevel(score int) RiskLevel {
	if score >= a.thresholds.CriticalThreshold {
		return RiskLevelCritical
	}
	if score >= a.thresholds.HighThreshold {
		return RiskLevelHigh
	}
	if score >= a.thresholds.MediumThreshold {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

func (a *RiskAssessor) estimateImpact(plan *models.ExecutionPlan, level RiskLevel) *ImpactEstimate {
	// 简单的影响评估逻辑
	est := &ImpactEstimate{
		AffectedResources: make([]string, 0),
		Reversible:        true,
		DataLossRisk:      false,
	}

	// 假设所有步骤的目标都是受影响资源 (Wait, ExecutionPlan doesn't have explicit target list in top level, check Steps)
	// In the pseudo code it had Targets. But models.ExecutionPlan doesn't seem to have Targets field yet.
	// We will infer from steps or just leave it empty for now.

	if level >= RiskLevelHigh {
		est.DataLossRisk = true // Conservative assumption
		est.Reversible = false
		est.DowntimeEstimate = 5 * time.Minute
	} else if level == RiskLevelMedium {
		est.DowntimeEstimate = 1 * time.Minute
	}

	return est
}
