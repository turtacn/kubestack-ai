package planning

import (
	"fmt"
)

// dagNode represents a node in the DAG
type dagNode struct {
	ID       string
	InDegree int
	Level    int // For parallel group calculation
}

// DAG represents a directed acyclic graph for step dependencies
type DAG struct {
	nodes map[string]*dagNode
	edges map[string][]string // node -> list of nodes that depend on it
}

// NewDAG creates a new DAG from a list of steps
func NewDAG(steps []Step) *DAG {
	dag := &DAG{
		nodes: make(map[string]*dagNode),
		edges: make(map[string][]string),
	}

	// Initialize nodes
	for _, step := range steps {
		dag.nodes[step.ID] = &dagNode{
			ID:       step.ID,
			InDegree: len(step.DependsOn),
		}
	}

	// Build edges
	for _, step := range steps {
		for _, depID := range step.DependsOn {
			dag.edges[depID] = append(dag.edges[depID], step.ID)
		}
	}

	return dag
}

// TopologicalSort performs a topological sort using Kahn's algorithm
func (d *DAG) TopologicalSort() ([]string, error) {
	// Create a copy of in-degrees to avoid modifying original
	inDegree := make(map[string]int)
	for id, node := range d.nodes {
		inDegree[id] = node.InDegree
	}

	// Initialize queue with nodes that have no dependencies
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var result []string
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependent nodes
		for _, dependent := range d.edges[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check if all nodes were processed
	if len(result) != len(d.nodes) {
		return nil, fmt.Errorf("cyclic dependency detected")
	}

	return result, nil
}

// DetectCycle checks if the DAG contains a cycle
func (d *DAG) DetectCycle() bool {
	_, err := d.TopologicalSort()
	return err != nil
}

// GetParallelGroups returns groups of steps that can be executed in parallel
// Steps in the same group have no dependencies on each other
func (d *DAG) GetParallelGroups() [][]string {
	// Calculate level for each node (longest path from source)
	levels := make(map[string]int)
	
	// Initialize all levels to 0
	for id := range d.nodes {
		levels[id] = 0
	}

	// Get topological order
	sorted, err := d.TopologicalSort()
	if err != nil {
		return nil
	}

	// Calculate levels based on dependencies
	for _, nodeID := range sorted {
		maxDepLevel := -1
		
		// Find dependencies by checking all edges
		for depID, dependents := range d.edges {
			for _, dep := range dependents {
				if dep == nodeID {
					if levels[depID] > maxDepLevel {
						maxDepLevel = levels[depID]
					}
				}
			}
		}
		
		levels[nodeID] = maxDepLevel + 1
	}

	// Group nodes by level
	levelGroups := make(map[int][]string)
	maxLevel := 0
	for id, level := range levels {
		levelGroups[level] = append(levelGroups[level], id)
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Convert to ordered slice
	result := make([][]string, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		result[level] = levelGroups[level]
	}

	return result
}

// GetExecutableSteps returns steps whose dependencies are all completed
func (d *DAG) GetExecutableSteps(completed map[string]bool) []string {
	var executable []string

	for id, node := range d.nodes {
		// Skip if already completed
		if completed[id] {
			continue
		}

		// Check if all dependencies are completed
		allDepsCompleted := true
		for depID, dependents := range d.edges {
			for _, dep := range dependents {
				if dep == id && !completed[depID] {
					allDepsCompleted = false
					break
				}
			}
			if !allDepsCompleted {
				break
			}
		}

		// If node has in-degree 0, it's executable
		if node.InDegree == 0 && !completed[id] {
			executable = append(executable, id)
		} else if allDepsCompleted {
			// Check dependencies more carefully
			canExecute := true
			// Find which nodes this one depends on
			for otherID := range d.nodes {
				for _, dependent := range d.edges[otherID] {
					if dependent == id && !completed[otherID] {
						canExecute = false
						break
					}
				}
				if !canExecute {
					break
				}
			}
			if canExecute {
				executable = append(executable, id)
			}
		}
	}

	return executable
}

// GetDependencies returns the direct dependencies of a step
func (d *DAG) GetDependencies(stepID string) []string {
	var deps []string
	for depID, dependents := range d.edges {
		for _, dep := range dependents {
			if dep == stepID {
				deps = append(deps, depID)
			}
		}
	}
	return deps
}

// GetDependents returns the steps that depend on the given step
func (d *DAG) GetDependents(stepID string) []string {
	return d.edges[stepID]
}
