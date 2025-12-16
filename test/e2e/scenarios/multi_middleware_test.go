package scenarios

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
	kgraph "github.com/kubestack-ai/kubestack-ai/internal/knowledge/graph"
	"github.com/kubestack-ai/kubestack-ai/test/e2e/framework"
)

func TestE2E_MultiMiddleware_Impact(t *testing.T) {
	suite := framework.NewE2ETestSuite(t)
	suite.Setup()
	defer suite.Teardown()

	// 1. Setup Graph with dependencies
	ctx := context.Background()

	// Service -> Redis
	suite.GraphStore.AddNode(ctx, &graph.Node{ID: "svc:order", Type: graph.NodeTypeService, Name: "order"})
	suite.GraphStore.AddNode(ctx, &graph.Node{ID: "mw:redis", Type: graph.NodeTypeMiddleware, Name: "redis"})
	suite.GraphStore.AddEdge(ctx, &graph.Edge{ID: "e1", FromID: "svc:order", ToID: "mw:redis", Type: graph.EdgeTypeDependsOn})

	// 2. Run Impact Analysis (Directly via QueryEngine for E2E integration verification)
	// In real E2E, this would be triggered via API
	q := kgraph.NewQueryEngine(suite.GraphStore)
	impact, err := q.FindImpactedServices(ctx, "mw:redis", 1)

	assert.NoError(t, err)
	assert.Len(t, impact.ImpactedNodes, 1)
	assert.Equal(t, "svc:order", impact.ImpactedNodes[0].ID)
}
