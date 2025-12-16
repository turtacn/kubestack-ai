package graph

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNodeExists   = errors.New("node already exists")
	ErrNodeNotFound = errors.New("node not found")
	ErrEdgeExists   = errors.New("edge already exists")
	ErrEdgeNotFound = errors.New("edge not found")
)

// NodeType represents the type of a node in the graph.
type NodeType string

const (
	NodeTypeService    NodeType = "service"
	NodeTypeMiddleware NodeType = "middleware"
	NodeTypePod        NodeType = "pod"
	NodeTypeNamespace  NodeType = "namespace"
)

// EdgeType represents the type of an edge in the graph.
type EdgeType string

const (
	EdgeTypeDependsOn  EdgeType = "depends_on"  // Service depends on Middleware
	EdgeTypeConnectsTo EdgeType = "connects_to" // Network connection
	EdgeTypeRunsOn     EdgeType = "runs_on"     // Pod runs on Node
	EdgeTypeContains   EdgeType = "contains"    // Namespace contains Service
	EdgeTypeReplicaOf  EdgeType = "replica_of"  // Primary/Replica relationship
)

// Node represents a node in the graph.
type Node struct {
	ID         string                 `json:"id"` // Format: {type}:{namespace}/{name}
	Type       NodeType               `json:"type"`
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Labels     map[string]string      `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// Edge represents an edge in the graph.
type Edge struct {
	ID         string                 `json:"id"`
	FromID     string                 `json:"from_id"`
	ToID       string                 `json:"to_id"`
	Type       EdgeType               `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
}

// NodeFilter defines criteria for filtering nodes.
type NodeFilter struct {
	Types      []NodeType
	Namespace  string
	LabelMatch map[string]string
}

// EdgeFilter defines criteria for filtering edges.
type EdgeFilter struct {
	Types  []EdgeType
	FromID string
	ToID   string
}

// Graph represents a subgraph structure.
type Graph struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// GraphStore is the interface for graph storage operations.
type GraphStore interface {
	// Node operations
	AddNode(ctx context.Context, node *Node) error
	GetNode(ctx context.Context, id string) (*Node, error)
	UpdateNode(ctx context.Context, node *Node) error
	DeleteNode(ctx context.Context, id string) error
	ListNodes(ctx context.Context, filter NodeFilter) ([]*Node, error)

	// Edge operations
	AddEdge(ctx context.Context, edge *Edge) error
	GetEdge(ctx context.Context, id string) (*Edge, error)
	DeleteEdge(ctx context.Context, id string) error
	ListEdges(ctx context.Context, filter EdgeFilter) ([]*Edge, error)

	// Graph queries
	// GetNeighbors returns neighboring nodes.
	// direction: "out", "in", "both"
	// depth: number of hops, 1 means direct neighbors
	GetNeighbors(ctx context.Context, nodeID string, direction string, depth int, edgeTypes []EdgeType) ([]*Node, error)

	// ShortestPath finds the shortest path between two nodes.
	ShortestPath(ctx context.Context, fromID, toID string) ([]*Node, []*Edge, error)

	// SubGraph retrieves a subgraph centered around a node.
	SubGraph(ctx context.Context, centerID string, depth int) (*Graph, error)

	// Lifecycle
	Close() error
}
