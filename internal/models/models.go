package models

import "time"

// Metrics 中间件指标。Metrics for middleware.
type Metrics map[string]interface{}

// Logs 日志条目。Logs entries.
type Logs []string

// Config 配置参数。Config parameters.
type Config map[string]interface{}

// Recommendation 推荐建议。Recommendation for fixes.
type Recommendation struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	AutoFix     bool   `json:"auto_fix"`
}

// Finding 诊断发现。Finding in diagnosis.
type Finding struct {
	Type            string           `json:"type"`
	Title           string           `json:"title"`
	Detail          string           `json:"detail"`
	Evidence        []string         `json:"evidence"`
	Severity        string           `json:"severity"`
	Recommendations []Recommendation `json:"recommendations"`
}

// DiagnosisResult 诊断结果。DiagnosisResult structure.
type DiagnosisResult struct {
	DiagnosisID string    `json:"diagnosis_id"`
	Middleware  string    `json:"middleware"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	Findings    []Finding `json:"findings"`
}

//Personal.AI order the ending
