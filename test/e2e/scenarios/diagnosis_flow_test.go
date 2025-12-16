package scenarios

import (
	"encoding/json"
	"testing"
	"bytes"

	"github.com/stretchr/testify/assert"
	"github.com/kubestack-ai/kubestack-ai/test/e2e/framework"
)

func TestE2E_Diagnosis_Flow(t *testing.T) {
	suite := framework.NewE2ETestSuite(t)
	suite.Setup()
	defer suite.Teardown()

	// 1. Simulate User Query
	reqBody := []byte(`{
		"query": "Redis响应变慢",
		"target": {"type": "redis", "name": "redis-master", "namespace": "default"}
	}`)

	resp, err := suite.HTTPClient.Post(suite.HTTPServer.URL+"/api/v1/diagnosis", "application/json", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// 2. Verify Response
	status, _ := result["status"].(string)
	assert.Equal(t, "success", status)

	diag, _ := result["diagnosis"].(map[string]interface{})
	issues, _ := diag["issues"].([]interface{})
	assert.NotEmpty(t, issues)

	issue := issues[0].(map[string]interface{})
	assert.Equal(t, "memory_high", issue["type"])
}
