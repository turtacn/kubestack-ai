package scenarios

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/kubestack-ai/kubestack-ai/test/e2e/framework"
)

func TestE2E_Fix_Flow(t *testing.T) {
	suite := framework.NewE2ETestSuite(t)
	suite.Setup()
	defer suite.Teardown()

	// 1. Get Fix Plan
	planReq := []byte(`{"action_id": "fix-1"}`)
	resp, err := suite.HTTPClient.Post(suite.HTTPServer.URL+"/api/v1/fix/plan", "application/json", bytes.NewBuffer(planReq))
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	var planResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&planResult)
	assert.Equal(t, "dry-run", planResult["mode"])

	// 2. Execute Fix with Approval
	execReq := []byte(`{"plan_id": "plan-1", "confirmed": true}`)
	resp2, err := suite.HTTPClient.Post(suite.HTTPServer.URL+"/api/v1/fix/execute", "application/json", bytes.NewBuffer(execReq))
	assert.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, 200, resp2.StatusCode)

	var execResult map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&execResult)

	resultMap, _ := execResult["result"].(map[string]interface{})

	framework.AssertFixExecuted(t, resultMap, framework.FixExpectation{
		Success:    true,
		ActionType: "command",
	})
}
