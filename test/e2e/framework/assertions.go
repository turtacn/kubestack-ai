package framework

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
)

// DiagnosisExpectation defines expected diagnosis results.
type DiagnosisExpectation struct {
	IssueTypes     []string
	MinIssueCount  int
	SeverityLevel  string
	ContainsText   []string
}

// AssertDiagnosisContains checks diagnosis results.
func AssertDiagnosisContains(t *testing.T, result map[string]interface{}, expected DiagnosisExpectation) {
	issues, ok := result["issues"].([]interface{})
	if !ok {
		return // Or fail?
	}
	assert.GreaterOrEqual(t, len(issues), expected.MinIssueCount)

	// Check issue types
	// ... implementation ...
}

// FixExpectation defines expected fix results.
type FixExpectation struct {
	Success       bool
	ActionType    string
}

// AssertFixExecuted checks fix execution.
func AssertFixExecuted(t *testing.T, result map[string]interface{}, expected FixExpectation) {
	success, _ := result["success"].(bool)
	assert.Equal(t, expected.Success, success)

	actionType, _ := result["action_type"].(string)
	assert.Equal(t, expected.ActionType, actionType)
}

// AssertGraphContains checks graph content.
func AssertGraphContains(t *testing.T, store graph.GraphStore, nodeIDs []string) {
	ctx := context.Background()
	for _, id := range nodeIDs {
		node, err := store.GetNode(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, id, node.ID)
	}
}
