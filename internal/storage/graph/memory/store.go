package memory

import (
	"context"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
)

// MemoryGraphStore implements GraphStore using in-memory maps.
type MemoryGraphStore struct {
	mu       sync.RWMutex
	nodes    map[string]*graph.Node
	edges    map[string]*graph.Edge
	outEdges map[string][]string // nodeID -> []edgeID
	inEdges  map[string][]string // nodeID -> []edgeID
}

// NewMemoryGraphStore creates a new in-memory graph store.
func NewMemoryGraphStore() *MemoryGraphStore {
	return &MemoryGraphStore{
		nodes:    make(map[string]*graph.Node),
		edges:    make(map[string]*graph.Edge),
		outEdges: make(map[string][]string),
		inEdges:  make(map[string][]string),
	}
}

func (s *MemoryGraphStore) AddNode(ctx context.Context, node *graph.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.ID]; exists {
		return graph.ErrNodeExists
	}

	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	if node.UpdatedAt.IsZero() {
		node.UpdatedAt = time.Now()
	}

	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryGraphStore) GetNode(ctx context.Context, id string) (*graph.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[id]
	if !exists {
		return nil, graph.ErrNodeNotFound
	}
	return node, nil
}

func (s *MemoryGraphStore) UpdateNode(ctx context.Context, node *graph.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.ID]; !exists {
		return graph.ErrNodeNotFound
	}

	node.UpdatedAt = time.Now()
	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryGraphStore) DeleteNode(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[id]; !exists {
		return graph.ErrNodeNotFound
	}

	// Remove associated edges
	for _, edgeID := range s.outEdges[id] {
		delete(s.edges, edgeID)
	}
	for _, edgeID := range s.inEdges[id] {
		delete(s.edges, edgeID)
	}
	delete(s.outEdges, id)
	delete(s.inEdges, id)

	delete(s.nodes, id)
	return nil
}

func (s *MemoryGraphStore) ListNodes(ctx context.Context, filter graph.NodeFilter) ([]*graph.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*graph.Node
	for _, node := range s.nodes {
		if len(filter.Types) > 0 {
			match := false
			for _, t := range filter.Types {
				if node.Type == t {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		if filter.Namespace != "" && node.Namespace != filter.Namespace {
			continue
		}

		if len(filter.LabelMatch) > 0 {
			match := true
			for k, v := range filter.LabelMatch {
				if val, ok := node.Labels[k]; !ok || val != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		result = append(result, node)
	}
	return result, nil
}

func (s *MemoryGraphStore) AddEdge(ctx context.Context, edge *graph.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.edges[edge.ID]; exists {
		return graph.ErrEdgeExists
	}
	// Check nodes exist
	if _, exists := s.nodes[edge.FromID]; !exists {
		return graph.ErrNodeNotFound
	}
	if _, exists := s.nodes[edge.ToID]; !exists {
		return graph.ErrNodeNotFound
	}

	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now()
	}

	s.edges[edge.ID] = edge
	s.outEdges[edge.FromID] = append(s.outEdges[edge.FromID], edge.ID)
	s.inEdges[edge.ToID] = append(s.inEdges[edge.ToID], edge.ID)
	return nil
}

func (s *MemoryGraphStore) GetEdge(ctx context.Context, id string) (*graph.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	edge, exists := s.edges[id]
	if !exists {
		return nil, graph.ErrEdgeNotFound
	}
	return edge, nil
}

func (s *MemoryGraphStore) DeleteEdge(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	edge, exists := s.edges[id]
	if !exists {
		return graph.ErrEdgeNotFound
	}

	// Remove from indices
	s.removeEdgeFromIndex(s.outEdges, edge.FromID, id)
	s.removeEdgeFromIndex(s.inEdges, edge.ToID, id)

	delete(s.edges, id)
	return nil
}

func (s *MemoryGraphStore) removeEdgeFromIndex(index map[string][]string, key, val string) {
	ids := index[key]
	for i, id := range ids {
		if id == val {
			index[key] = append(ids[:i], ids[i+1:]...)
			return
		}
	}
}

func (s *MemoryGraphStore) ListEdges(ctx context.Context, filter graph.EdgeFilter) ([]*graph.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*graph.Edge
	for _, edge := range s.edges {
		if len(filter.Types) > 0 {
			match := false
			for _, t := range filter.Types {
				if edge.Type == t {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if filter.FromID != "" && edge.FromID != filter.FromID {
			continue
		}
		if filter.ToID != "" && edge.ToID != filter.ToID {
			continue
		}
		result = append(result, edge)
	}
	return result, nil
}

func (s *MemoryGraphStore) GetNeighbors(ctx context.Context, nodeID string, direction string, depth int, edgeTypes []graph.EdgeType) ([]*graph.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.nodes[nodeID]; !exists {
		return nil, graph.ErrNodeNotFound
	}

	visited := make(map[string]bool)
	visited[nodeID] = true

	queue := []struct {
		id string
		d  int
	}{{nodeID, 0}}

	var result []*graph.Node
	edgeTypeSet := make(map[graph.EdgeType]bool)
	for _, t := range edgeTypes {
		edgeTypeSet[t] = true
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.d >= depth {
			continue
		}

		var edgeIDs []string
		if direction == "out" || direction == "both" {
			edgeIDs = append(edgeIDs, s.outEdges[curr.id]...)
		}
		if direction == "in" || direction == "both" {
			edgeIDs = append(edgeIDs, s.inEdges[curr.id]...)
		}

		for _, eid := range edgeIDs {
			edge := s.edges[eid]

			if len(edgeTypeSet) > 0 && !edgeTypeSet[edge.Type] {
				continue
			}

			neighborID := edge.ToID
			if direction == "in" {
				neighborID = edge.FromID
			} else if direction == "both" {
				if edge.ToID == curr.id {
					neighborID = edge.FromID
				}
			} else if direction == "out" && edge.FromID != curr.id {
                // Should not happen if outEdges is correct
                continue
            }

			if !visited[neighborID] {
				visited[neighborID] = true
				if node, exists := s.nodes[neighborID]; exists {
					result = append(result, node)
					queue = append(queue, struct {
						id string
						d  int
					}{neighborID, curr.d + 1})
				}
			}
		}
	}

	return result, nil
}

func (s *MemoryGraphStore) ShortestPath(ctx context.Context, fromID, toID string) ([]*graph.Node, []*graph.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

    if _, exists := s.nodes[fromID]; !exists {
        return nil, nil, graph.ErrNodeNotFound
    }
    if _, exists := s.nodes[toID]; !exists {
        return nil, nil, graph.ErrNodeNotFound
    }

	// BFS for shortest path (unweighted)
	queue := []string{fromID}
	visited := map[string]bool{fromID: true}
	parent := make(map[string]string)      // nodeID -> parentNodeID
	parentEdge := make(map[string]*graph.Edge) // nodeID -> edgeFromParent

	found := false
	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]

		if currID == toID {
			found = true
			break
		}

		// Consider out edges for path traversal (directed graph usually)
        // Or should it be undirected? Assuming directed for now.
		for _, eid := range s.outEdges[currID] {
			edge := s.edges[eid]
			neighborID := edge.ToID

			if !visited[neighborID] {
				visited[neighborID] = true
				parent[neighborID] = currID
				parentEdge[neighborID] = edge
				queue = append(queue, neighborID)
			}
		}
	}

	if !found {
		return nil, nil, nil
	}

	// Reconstruct path
	var pathNodes []*graph.Node
	var pathEdges []*graph.Edge
	curr := toID
	for curr != fromID {
		pathNodes = append(pathNodes, s.nodes[curr])
		edge := parentEdge[curr]
		pathEdges = append(pathEdges, edge)
		curr = parent[curr]
	}
	pathNodes = append(pathNodes, s.nodes[fromID])

	// Reverse
	for i, j := 0, len(pathNodes)-1; i < j; i, j = i+1, j-1 {
		pathNodes[i], pathNodes[j] = pathNodes[j], pathNodes[i]
	}
	for i, j := 0, len(pathEdges)-1; i < j; i, j = i+1, j-1 {
		pathEdges[i], pathEdges[j] = pathEdges[j], pathEdges[i]
	}

	return pathNodes, pathEdges, nil
}

func (s *MemoryGraphStore) SubGraph(ctx context.Context, centerID string, depth int) (*graph.Graph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

    if _, exists := s.nodes[centerID]; !exists {
        return nil, graph.ErrNodeNotFound
    }

	nodes := make(map[string]*graph.Node)
	edges := make(map[string]*graph.Edge)

	// Add center
	nodes[centerID] = s.nodes[centerID]

	queue := []struct {
		id string
		d  int
	}{{centerID, 0}}
    visited := map[string]bool{centerID: true}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.d >= depth {
			continue
		}

		// Check both in and out edges
		var edgeIDs []string
		edgeIDs = append(edgeIDs, s.outEdges[curr.id]...)
		edgeIDs = append(edgeIDs, s.inEdges[curr.id]...)

		for _, eid := range edgeIDs {
			edge := s.edges[eid]
			edges[eid] = edge // Add edge to subgraph

			neighborID := edge.ToID
			if edge.ToID == curr.id {
				neighborID = edge.FromID
			}

            if !visited[neighborID] {
                visited[neighborID] = true
                if node, ok := s.nodes[neighborID]; ok {
                    nodes[neighborID] = node
                    queue = append(queue, struct{id string; d int}{neighborID, curr.d + 1})
                }
            } else {
                 // Even if visited, we ensure the node is in the subgraph map (already there)
            }
            if node, ok := s.nodes[neighborID]; ok {
                nodes[neighborID] = node
            }
		}
	}

	res := &graph.Graph{}
	for _, n := range nodes {
		res.Nodes = append(res.Nodes, n)
	}
	for _, e := range edges {
		res.Edges = append(res.Edges, e)
	}
	return res, nil
}

func (s *MemoryGraphStore) Close() error {
	return nil
}
