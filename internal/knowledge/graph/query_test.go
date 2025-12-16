package graph

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph/memory"
	"github.com/stretchr/testify/assert"
)

func TestQuery_FindImpactedServices(t *testing.T) {
	store := memory.NewMemoryGraphStore()
	q := NewQueryEngine(store)
	ctx := context.Background()

	// Setup: Service -> Redis
	svcNode := &graph.Node{ID: "svc1", Type: graph.NodeTypeService, Name: "svc1"}
	mwNode := &graph.Node{ID: "redis1", Type: graph.NodeTypeMiddleware, Name: "redis1"}
	store.AddNode(ctx, svcNode)
	store.AddNode(ctx, mwNode)
	store.AddEdge(ctx, &graph.Edge{
		ID:     "e1",
		FromID: "svc1",
		ToID:   "redis1",
		Type:   graph.EdgeTypeDependsOn,
	})

	// Find impact if Redis fails
	res, err := q.FindImpactedServices(ctx, "redis1", 1)
	assert.NoError(t, err)
	assert.Len(t, res.ImpactedNodes, 1)
	assert.Equal(t, "svc1", res.ImpactedNodes[0].ID)
	assert.Equal(t, "medium", res.ImpactLevel)
}

func TestQuery_TraceRootCause(t *testing.T) {
	store := memory.NewMemoryGraphStore()
	q := NewQueryEngine(store)
	ctx := context.Background()

	svcNode := &graph.Node{ID: "svc1", Type: graph.NodeTypeService, Name: "svc1"}
	mwNode := &graph.Node{
		ID:   "redis1",
		Type: graph.NodeTypeMiddleware,
		Name: "redis1",
		Properties: map[string]interface{}{
			"health_status": "unhealthy",
		},
	}
	store.AddNode(ctx, svcNode)
	store.AddNode(ctx, mwNode)
	store.AddEdge(ctx, &graph.Edge{
		ID:     "e1",
		FromID: "svc1",
		ToID:   "redis1",
		Type:   graph.EdgeTypeDependsOn,
	})

	// Trace root cause for service issue
	res, err := q.TraceRootCause(ctx, "svc1")
	assert.NoError(t, err)
	assert.Equal(t, "redis1", res.RootCauseNode.ID)
}
