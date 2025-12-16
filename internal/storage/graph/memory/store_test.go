package memory

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
	"github.com/stretchr/testify/assert"
)

func TestMemoryStore_AddNode(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	node := &graph.Node{
		ID:        "service:default/test-service",
		Type:      graph.NodeTypeService,
		Name:      "test-service",
		Namespace: "default",
	}

	err := store.AddNode(ctx, node)
	assert.NoError(t, err)

	// Test duplicate
	err = store.AddNode(ctx, node)
	assert.Equal(t, graph.ErrNodeExists, err)
}

func TestMemoryStore_AddEdge(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	n1 := &graph.Node{ID: "n1", Type: graph.NodeTypeService}
	n2 := &graph.Node{ID: "n2", Type: graph.NodeTypeMiddleware}
	store.AddNode(ctx, n1)
	store.AddNode(ctx, n2)

	edge := &graph.Edge{
		ID:     "e1",
		FromID: "n1",
		ToID:   "n2",
		Type:   graph.EdgeTypeDependsOn,
	}

	err := store.AddEdge(ctx, edge)
	assert.NoError(t, err)

	// Verify indices
	assert.Contains(t, store.outEdges["n1"], "e1")
	assert.Contains(t, store.inEdges["n2"], "e1")
}

func TestMemoryStore_GetNeighbors(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	// n1 -> n2 -> n3
	nodes := []*graph.Node{
		{ID: "n1"}, {ID: "n2"}, {ID: "n3"},
	}
	for _, n := range nodes {
		store.AddNode(ctx, n)
	}

	edges := []*graph.Edge{
		{ID: "e1", FromID: "n1", ToID: "n2", Type: graph.EdgeTypeConnectsTo},
		{ID: "e2", FromID: "n2", ToID: "n3", Type: graph.EdgeTypeConnectsTo},
	}
	for _, e := range edges {
		store.AddEdge(ctx, e)
	}

	// Test 1 hop out
	neighbors, err := store.GetNeighbors(ctx, "n1", "out", 1, nil)
	assert.NoError(t, err)
	assert.Len(t, neighbors, 1)
	assert.Equal(t, "n2", neighbors[0].ID)

	// Test 2 hops out
	neighbors, err = store.GetNeighbors(ctx, "n1", "out", 2, nil)
	assert.NoError(t, err)
	assert.Len(t, neighbors, 2) // n2, n3 (BFS order guaranteed in implementation?) - order depends on map iteration if using range on visited, but here we append in order
    // Actually implementation returns n2 then n3
    ids := make([]string, len(neighbors))
    for i, n := range neighbors {
        ids[i] = n.ID
    }
	assert.Contains(t, ids, "n2")
    assert.Contains(t, ids, "n3")

	// Test in direction
	neighbors, err = store.GetNeighbors(ctx, "n2", "in", 1, nil)
	assert.NoError(t, err)
	assert.Len(t, neighbors, 1)
	assert.Equal(t, "n1", neighbors[0].ID)
}

func TestMemoryStore_ShortestPath(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	// n1 -> n2 -> n3
	nodes := []*graph.Node{
		{ID: "n1"}, {ID: "n2"}, {ID: "n3"}, {ID: "n4"},
	}
	for _, n := range nodes {
		store.AddNode(ctx, n)
	}

	edges := []*graph.Edge{
		{ID: "e1", FromID: "n1", ToID: "n2"},
		{ID: "e2", FromID: "n2", ToID: "n3"},
		{ID: "e3", FromID: "n1", ToID: "n3"}, // Shortcut
	}
	for _, e := range edges {
        e.CreatedAt = time.Now()
		store.AddEdge(ctx, e)
	}

	// Path n1 -> n3 (should take shortcut e3)
	pathNodes, pathEdges, err := store.ShortestPath(ctx, "n1", "n3")
	assert.NoError(t, err)
	assert.Len(t, pathNodes, 2)
	assert.Equal(t, "n1", pathNodes[0].ID)
	assert.Equal(t, "n3", pathNodes[1].ID)
	assert.Len(t, pathEdges, 1)
	assert.Equal(t, "e3", pathEdges[0].ID)
}
