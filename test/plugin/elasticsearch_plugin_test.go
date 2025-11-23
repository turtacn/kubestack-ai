package plugin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/plugins/elasticsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElasticsearchPluginDiagnose(t *testing.T) {
	// Setup: Mock ES
	mockES := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_cluster/health" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":          "yellow",
				"number_of_nodes": 3,
			})
		}
	}))
	defer mockES.Close()

	plugin := &elasticsearch.ElasticsearchPlugin{}
	err := plugin.Init(map[string]interface{}{
		"urls": []interface{}{mockES.URL},
	})
	require.NoError(t, err)

	// Action: 执行诊断
	result, err := plugin.Diagnose(context.Background(), &models.DiagnosisRequest{})

	// Assert: 检测到yellow状态
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Issues, 1)
	assert.Equal(t, "Elasticsearch Cluster Status YELLOW", result.Issues[0].Title)
}
