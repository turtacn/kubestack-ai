package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
	// "k8s.io/client-go/kubernetes" // Assuming this dependency exists or mocked
)

// Builder builds the knowledge graph from various sources.
type Builder struct {
	store graph.GraphStore
	// k8sClient kubernetes.Interface // Placeholder for now
}

// NewBuilder creates a new Graph Builder.
func NewBuilder(store graph.GraphStore) *Builder {
	return &Builder{
		store: store,
	}
}

// BuildFromTopology builds graph from a simple topology definition (for testing/demo).
// Real implementation would inspect K8s resources.
type Topology struct {
	Nodes []TopologyNode `json:"nodes" yaml:"nodes"`
	Edges []TopologyEdge `json:"edges" yaml:"edges"`
}

type TopologyNode struct {
	ID        string            `json:"id" yaml:"id"`
	Type      string            `json:"type" yaml:"type"`
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Labels    map[string]string `json:"labels" yaml:"labels"`
}

type TopologyEdge struct {
	FromID string `json:"from_id" yaml:"from_id"`
	ToID   string `json:"to_id" yaml:"to_id"`
	Type   string `json:"type" yaml:"type"`
}

func (b *Builder) BuildFromTopology(ctx context.Context, topo Topology) error {
	// Add nodes
	for _, n := range topo.Nodes {
		node := &graph.Node{
			ID:        n.ID,
			Type:      graph.NodeType(n.Type),
			Name:      n.Name,
			Namespace: n.Namespace,
			Labels:    n.Labels,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := b.store.AddNode(ctx, node); err != nil {
			if err != graph.ErrNodeExists {
				return fmt.Errorf("failed to add node %s: %w", node.ID, err)
			}
			// Update existing node?
			// For now, we ignore ErrNodeExists as per requirement, but we should return other errors.
		}
	}

	// Add edges
	for i, e := range topo.Edges {
		edge := &graph.Edge{
			ID:        fmt.Sprintf("edge-%d", i), // Simple ID generation
			FromID:    e.FromID,
			ToID:      e.ToID,
			Type:      graph.EdgeType(e.Type),
			CreatedAt: time.Now(),
		}
		// Create a unique ID for edge if not provided, usually hash of from+to+type
		edge.ID = fmt.Sprintf("%s-%s-%s", e.FromID, e.Type, e.ToID)

		if err := b.store.AddEdge(ctx, edge); err != nil {
			if err != graph.ErrEdgeExists {
				return fmt.Errorf("failed to add edge %s: %w", edge.ID, err)
			}
		}
	}
	return nil
}

// AddDependency adds a manual dependency edge.
func (b *Builder) AddDependency(ctx context.Context, serviceID, middlewareID string) error {
	edge := &graph.Edge{
		ID:        fmt.Sprintf("%s-depends_on-%s", serviceID, middlewareID),
		FromID:    serviceID,
		ToID:      middlewareID,
		Type:      graph.EdgeTypeDependsOn,
		CreatedAt: time.Now(),
	}
	return b.store.AddEdge(ctx, edge)
}

// Helper to infer ID from name/namespace/type
func GenerateID(nodeType graph.NodeType, namespace, name string) string {
	return fmt.Sprintf("%s:%s/%s", nodeType, namespace, name)
}

func (b *Builder) AddService(ctx context.Context, namespace, name string) error {
	id := GenerateID(graph.NodeTypeService, namespace, name)
	node := &graph.Node{
		ID:        id,
		Type:      graph.NodeTypeService,
		Name:      name,
		Namespace: namespace,
		CreatedAt: time.Now(),
	}
	return b.store.AddNode(ctx, node)
}

func (b *Builder) AddMiddleware(ctx context.Context, namespace, name, mwType string) error {
	id := GenerateID(graph.NodeTypeMiddleware, namespace, name)
	node := &MiddlewareNode{
		Node: &graph.Node{
			ID:        id,
			Type:      graph.NodeTypeMiddleware,
			Name:      name,
			Namespace: namespace,
			CreatedAt: time.Now(),
			Properties: map[string]interface{}{
				"middleware_type": mwType,
			},
		},
		MiddlewareType: mwType,
	}
	// We only store the base Node part in generic store, or we store properties.
	// The store interface takes *graph.Node.
	// So we should flatten properties into Node.Properties
	return b.store.AddNode(ctx, node.Node)
}
