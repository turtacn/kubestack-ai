package graph

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
)

// QueryEngine implements graph analysis capabilities.
type QueryEngine struct {
	store graph.GraphStore
}

// NewQueryEngine creates a new QueryEngine.
func NewQueryEngine(store graph.GraphStore) *QueryEngine {
	return &QueryEngine{
		store: store,
	}
}

// FindImpactedServices finds services affected by a middleware issue.
// It traverses "depends_on" edges in reverse ("in" direction).
func (q *QueryEngine) FindImpactedServices(ctx context.Context, middlewareID string, maxDepth int) (*ImpactAnalysisResult, error) {
	node, err := q.store.GetNode(ctx, middlewareID)
	if err != nil {
		return nil, err
	}

	// Traverse "depends_on" incoming edges to find services
	neighbors, err := q.store.GetNeighbors(ctx, middlewareID, "in", maxDepth, []graph.EdgeType{graph.EdgeTypeDependsOn})
	if err != nil {
		return nil, err
	}

	var impactedNodes []*graph.Node
	for _, n := range neighbors {
		if n.Type == graph.NodeTypeService {
			impactedNodes = append(impactedNodes, n)
		} else if n.Type == graph.NodeTypeMiddleware {
            // Also include other middleware that depends on this one
            impactedNodes = append(impactedNodes, n)
        }
	}

    // Determine impact level
	level := "low"
	if len(impactedNodes) > 5 {
		level = "high"
	} else if len(impactedNodes) > 0 {
		level = "medium"
	}

	return &ImpactAnalysisResult{
		SourceNode:     node,
		ImpactedNodes:  impactedNodes,
		ImpactLevel:    level,
		EstimatedScope: fmt.Sprintf("%d dependent services/components impacted", len(impactedNodes)),
	}, nil
}

// TraceRootCause attempts to find the root cause of a symptom.
// It traverses "depends_on" edges forward ("out" direction) to find unhealthy dependencies.
func (q *QueryEngine) TraceRootCause(ctx context.Context, symptomNodeID string) (*RootCauseResult, error) {
	node, err := q.store.GetNode(ctx, symptomNodeID)
	if err != nil {
		return nil, err
	}

	// Simple heuristic: check dependencies
	deps, err := q.store.GetNeighbors(ctx, symptomNodeID, "out", 1, []graph.EdgeType{graph.EdgeTypeDependsOn})
	if err != nil {
		return nil, err
	}

	// For simulation, we assume if a dependency is marked as 'unhealthy' in properties, it's a candidate
    // In real world, we would query metrics/status of these nodes.
    // Here we rely on graph properties.

    var rootCauses []*graph.Node
	for _, dep := range deps {
		if status, ok := dep.Properties["health_status"].(string); ok && status != "healthy" {
			rootCauses = append(rootCauses, dep)
		}
	}

	if len(rootCauses) == 0 {
		return &RootCauseResult{
			SymptomNode:   node,
			RootCauseNode: node, // Self is root cause
			Confidence:    0.5,
			Evidence:      []string{"No unhealthy dependencies found"},
		}, nil
	}

    // If multiple, pick first for now
	return &RootCauseResult{
		SymptomNode:   node,
		RootCauseNode: rootCauses[0],
		CausalChain:   []*graph.Node{node, rootCauses[0]},
		Confidence:    0.8,
		Evidence:      []string{fmt.Sprintf("Dependency %s is unhealthy", rootCauses[0].Name)},
	}, nil
}

// GetDependencyChain retrieves the full dependency chain for visualization.
func (q *QueryEngine) GetDependencyChain(ctx context.Context, serviceID string) ([]*graph.Node, []*graph.Edge, error) {
    // Just a subgraph query
    sub, err := q.store.SubGraph(ctx, serviceID, 3)
    if err != nil {
        return nil, nil, err
    }
    return sub.Nodes, sub.Edges, nil
}
