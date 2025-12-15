package risk

import "time"

// RiskLevel 风险等级枚举
type RiskLevel int

const (
    RiskLevelLow      RiskLevel = 1  // 低风险: 只读/监控类操作
    RiskLevelMedium   RiskLevel = 2  // 中风险: 配置修改/连接管理
    RiskLevelHigh     RiskLevel = 3  // 高风险: 数据删除/服务重启
    RiskLevelCritical RiskLevel = 4  // 严重风险: 需人工审批
)

// String returns the string representation of the risk level
func (r RiskLevel) String() string {
    switch r {
    case RiskLevelLow:
        return "Low"
    case RiskLevelMedium:
        return "Medium"
    case RiskLevelHigh:
        return "High"
    case RiskLevelCritical:
        return "Critical"
    default:
        return "Unknown"
    }
}

// ImpactEstimate 影响评估
type ImpactEstimate struct {
    AffectedResources []string      // 受影响资源
    DowntimeEstimate  time.Duration // 预估停机时间
    DataLossRisk      bool          // 是否有数据丢失风险
    Reversible        bool          // 是否可逆
}

// RiskAssessmentResult 评估结果
type RiskAssessmentResult struct {
    Level           RiskLevel
    Score           int             // 0-100分
    Reasons         []string        // 风险原因列表
    Mitigations     []string        // 缓解措施建议
    RequiresConfirm bool            // 是否需要确认
    RequiresApproval bool           // 是否需要审批
    EstimatedImpact *ImpactEstimate // 影响评估
}
