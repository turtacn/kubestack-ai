package graph

import (
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
)

// MiddlewareNode extends the basic Node with middleware-specific fields.
type MiddlewareNode struct {
	*graph.Node
	MiddlewareType string   `json:"middleware_type"` // redis/mysql/kafka...
	Version        string   `json:"version"`
	ClusterMode    string   `json:"cluster_mode"` // standalone/cluster/sentinel
	Endpoints      []string `json:"endpoints"`
	HealthStatus   string   `json:"health_status"`
}

// ServiceNode extends the basic Node with service-specific fields.
type ServiceNode struct {
	*graph.Node
	ServiceType  string   `json:"service_type"` // deployment/statefulset
	Replicas     int      `json:"replicas"`
	Dependencies []string `json:"dependencies"` // List of dependent middleware IDs
}

// DependencyEdge extends the basic Edge with dependency-specific properties.
type DependencyEdge struct {
	*graph.Edge
	ConnectionPool int           `json:"connection_pool"`
	Timeout        time.Duration `json:"timeout"`
	Protocol       string        `json:"protocol"` // tcp/http/grpc
}

// ImpactAnalysisResult represents the result of an impact analysis.
type ImpactAnalysisResult struct {
	SourceNode     *graph.Node     `json:"source_node"`
	ImpactedNodes  []*graph.Node   `json:"impacted_nodes"`
	ImpactPaths    [][]*graph.Edge `json:"impact_paths"`
	ImpactLevel    string          `json:"impact_level"` // critical/high/medium/low
	EstimatedScope string          `json:"estimated_scope"`
}

// RootCauseResult represents the result of a root cause analysis.
type RootCauseResult struct {
	SymptomNode   *graph.Node   `json:"symptom_node"`
	RootCauseNode *graph.Node   `json:"root_cause_node"`
	CausalChain   []*graph.Node `json:"causal_chain"`
	Confidence    float64       `json:"confidence"` // 0-1
	Evidence      []string      `json:"evidence"`
}
