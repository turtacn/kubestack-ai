package graph

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph/memory"
	"github.com/stretchr/testify/assert"
)

func TestBuilder_BuildFromTopology(t *testing.T) {
	store := memory.NewMemoryGraphStore()
	builder := NewBuilder(store)
	ctx := context.Background()

	topo := Topology{
		Nodes: []TopologyNode{
			{ID: "svc1", Type: "service", Name: "svc1", Namespace: "default"},
			{ID: "redis1", Type: "middleware", Name: "redis1", Namespace: "default"},
		},
		Edges: []TopologyEdge{
			{FromID: "svc1", ToID: "redis1", Type: "depends_on"},
		},
	}

	err := builder.BuildFromTopology(ctx, topo)
	assert.NoError(t, err)

	// Verify
	node, err := store.GetNode(ctx, "svc1")
	assert.NoError(t, err)
	assert.Equal(t, "svc1", node.Name)

	neighbors, err := store.GetNeighbors(ctx, "svc1", "out", 1, nil)
	assert.NoError(t, err)
	assert.Len(t, neighbors, 1)
	assert.Equal(t, "redis1", neighbors[0].ID)
}

func TestBuilder_AddDependency(t *testing.T) {
	store := memory.NewMemoryGraphStore()
	builder := NewBuilder(store)
	ctx := context.Background()

	builder.AddService(ctx, "default", "app")
	builder.AddMiddleware(ctx, "default", "db", "mysql")

	svcID := GenerateID(graph.NodeTypeService, "default", "app")
	mwID := GenerateID(graph.NodeTypeMiddleware, "default", "db")

	err := builder.AddDependency(ctx, svcID, mwID)
	assert.NoError(t, err)

	neighbors, err := store.GetNeighbors(ctx, svcID, "out", 1, nil)
	assert.NoError(t, err)
	assert.Equal(t, mwID, neighbors[0].ID)
}
