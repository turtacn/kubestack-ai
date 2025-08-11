package models

import (
	"github.com/turtacn/kubestack-ai/internal/constants"
	"time"

	"github.com/google/uuid"
)

// Metrics 中间件指标。Metrics for middleware.
type Metrics map[string]interface{}

// Logs 日志条目。Logs entries.
type Logs []LogEntry

// LogEntry 单条日志记录。Single log entry.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// Config 配置参数。Config parameters.
type Config map[string]interface{}

// Recommendation 推荐建议。Recommendation for fixes.
type Recommendation struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	AutoFix     bool   `json:"auto_fix"`
	RiskLevel   string `json:"risk_level,omitempty"`
}

// Finding 诊断发现。Finding in diagnosis.
type Finding struct {
	Type               string           `json:"type"`
	Title              string           `json:"title"`
	Detail             string           `json:"detail"`
	Evidence           []string         `json:"evidence"`
	Severity           string           `json:"severity"`
	Recommendations    []Recommendation `json:"recommendations"`
	AffectedComponents []string         `json:"affected_components,omitempty"`
}

// DiagnosisResult 诊断结果。DiagnosisResult structure.
type DiagnosisResult struct {
	DiagnosisID       string    `json:"diagnosis_id"`
	Middleware        string    `json:"middleware"`
	MiddlewareVersion string    `json:"middleware_version,omitempty"`
	Environment       string    `json:"environment"`
	Timestamp         time.Time `json:"timestamp"`
	Status            string    `json:"status"`
	Findings          []Finding `json:"findings"`
	Duration          float64   `json:"duration_seconds,omitempty"`
}

// KnowledgeDocument 知识库文档。Knowledge document for RAG.
type KnowledgeDocument struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Middleware string            `json:"middleware"`
	Source     string            `json:"source"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

// NewDiagnosisResult 创建新的诊断结果。Create new diagnosis result.
func NewDiagnosisResult(middleware string, environment string) *DiagnosisResult {
	return &DiagnosisResult{
		DiagnosisID: uuid.New().String(),
		Middleware:  middleware,
		Environment: environment,
		Timestamp:   time.Now(),
		Status:      constants.StatusHealthy,
		Findings:    []Finding{},
	}
}

//Personal.AI order the ending
