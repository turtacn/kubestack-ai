package planning

import (
	"testing"
)

func TestDAG_TopologicalSort(t *testing.T) {
	steps := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
		{ID: "C", Name: "Step C", DependsOn: []string{"A"}},
		{ID: "D", Name: "Step D", DependsOn: []string{"B", "C"}},
	}

	dag := NewDAG(steps)
	sorted, err := dag.TopologicalSort()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sorted) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(sorted))
	}

	// Check that A comes before B and C
	posA := indexOf(sorted, "A")
	posB := indexOf(sorted, "B")
	posC := indexOf(sorted, "C")
	posD := indexOf(sorted, "D")

	if posA >= posB || posA >= posC {
		t.Error("A should come before B and C")
	}

	if posB >= posD || posC >= posD {
		t.Error("B and C should come before D")
	}
}

func TestDAG_DetectCycle(t *testing.T) {
	// No cycle
	steps1 := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
	}

	dag1 := NewDAG(steps1)
	if dag1.DetectCycle() {
		t.Error("expected no cycle, but cycle detected")
	}

	// With cycle
	steps2 := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{"B"}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
	}

	dag2 := NewDAG(steps2)
	if !dag2.DetectCycle() {
		t.Error("expected cycle, but no cycle detected")
	}
}

func TestDAG_ParallelGroups(t *testing.T) {
	steps := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
		{ID: "C", Name: "Step C", DependsOn: []string{"A"}},
		{ID: "D", Name: "Step D", DependsOn: []string{"B", "C"}},
	}

	dag := NewDAG(steps)
	groups := dag.GetParallelGroups()

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// First group should contain only A
	if len(groups[0]) != 1 || groups[0][0] != "A" {
		t.Errorf("expected first group to be [A], got %v", groups[0])
	}

	// Second group should contain B and C (order doesn't matter)
	if len(groups[1]) != 2 {
		t.Errorf("expected second group to have 2 elements, got %d", len(groups[1]))
	}
	if !contains(groups[1], "B") || !contains(groups[1], "C") {
		t.Errorf("expected second group to contain B and C, got %v", groups[1])
	}

	// Third group should contain only D
	if len(groups[2]) != 1 || groups[2][0] != "D" {
		t.Errorf("expected third group to be [D], got %v", groups[2])
	}
}

func TestDAG_LinearChain(t *testing.T) {
	steps := []Step{
		{ID: "step1", Name: "Step 1", DependsOn: []string{}},
		{ID: "step2", Name: "Step 2", DependsOn: []string{"step1"}},
		{ID: "step3", Name: "Step 3", DependsOn: []string{"step2"}},
	}

	dag := NewDAG(steps)
	sorted, err := dag.TopologicalSort()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"step1", "step2", "step3"}
	for i, id := range expected {
		if sorted[i] != id {
			t.Errorf("expected position %d to be %s, got %s", i, id, sorted[i])
		}
	}
}

func TestDAG_GetExecutableSteps(t *testing.T) {
	steps := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
		{ID: "C", Name: "Step C", DependsOn: []string{"A"}},
	}

	dag := NewDAG(steps)

	// Initially, only A should be executable
	completed := make(map[string]bool)
	executable := dag.GetExecutableSteps(completed)

	if len(executable) != 1 || executable[0] != "A" {
		t.Errorf("expected [A] to be executable, got %v", executable)
	}

	// After completing A, B and C should be executable
	completed["A"] = true
	executable = dag.GetExecutableSteps(completed)

	if len(executable) != 2 {
		t.Errorf("expected 2 executable steps, got %d", len(executable))
	}
	if !contains(executable, "B") || !contains(executable, "C") {
		t.Errorf("expected B and C to be executable, got %v", executable)
	}
}

func TestDAG_GetDependencies(t *testing.T) {
	steps := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
		{ID: "C", Name: "Step C", DependsOn: []string{"A", "B"}},
	}

	dag := NewDAG(steps)

	depsC := dag.GetDependencies("C")
	if len(depsC) != 2 {
		t.Errorf("expected 2 dependencies for C, got %d", len(depsC))
	}
	if !contains(depsC, "A") || !contains(depsC, "B") {
		t.Errorf("expected A and B as dependencies of C, got %v", depsC)
	}
}

func TestDAG_GetDependents(t *testing.T) {
	steps := []Step{
		{ID: "A", Name: "Step A", DependsOn: []string{}},
		{ID: "B", Name: "Step B", DependsOn: []string{"A"}},
		{ID: "C", Name: "Step C", DependsOn: []string{"A"}},
	}

	dag := NewDAG(steps)

	depsA := dag.GetDependents("A")
	if len(depsA) != 2 {
		t.Errorf("expected 2 dependents for A, got %d", len(depsA))
	}
	if !contains(depsA, "B") || !contains(depsA, "C") {
		t.Errorf("expected B and C as dependents of A, got %v", depsA)
	}
}

// Helper functions
func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}

func contains(slice []string, val string) bool {
	return indexOf(slice, val) >= 0
}
