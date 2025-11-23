package elasticsearch

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
)

type mockTransport struct {
	Response *http.Response
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, nil
}

func TestElasticsearchHealthChecker_Check(t *testing.T) {
	// We verify methods individually below
}

func TestElasticsearchHealthChecker_ClusterHealth(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Elastic-Product", "Elasticsearch")
	mockTrans := &mockTransport{
		Response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status": "green"}`)),
			Header:     header,
		},
	}

	client, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: mockTrans,
	})

	p := &ElasticsearchPlugin{client: client}
	checker := &ElasticsearchHealthChecker{plugin: p}

	result := checker.checkClusterHealth(context.Background())
	assert.Equal(t, plugin.HealthyLevel, result.Status)
	assert.Equal(t, "green", result.Message)
}

func TestElasticsearchHealthChecker_DiskWatermark(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Elastic-Product", "Elasticsearch")
	mockTrans := &mockTransport{
		Response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{
				"nodes": {
					"node1": {
						"fs": {
							"total": {
								"total_in_bytes": 100000.0,
								"available_in_bytes": 1000.0
							}
						}
					}
				}
			}`)), // 99% usage
			Header:     header,
		},
	}

	client, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: mockTrans,
	})

	p := &ElasticsearchPlugin{client: client}
	checker := &ElasticsearchHealthChecker{plugin: p}

	result := checker.checkDiskWatermark(context.Background())
	assert.Equal(t, plugin.UnhealthyLevel, result.Status)
	assert.Contains(t, result.Message, "critical")
}

func TestElasticsearchMetricParser_Parse(t *testing.T) {
	p := &ElasticsearchPlugin{}
	parser := &ElasticsearchMetricParser{plugin: p}

	data := &plugin.CollectedData{
		RawData: map[string]interface{}{
			"health": map[string]interface{}{
				"status": "yellow",
				"active_shards_percent_as_number": 98.5,
			},
		},
	}

	metrics, err := parser.Parse(context.Background(), data)
	assert.NoError(t, err)

	assert.Equal(t, "yellow", metrics.Metrics["cluster_status"].Value)
	assert.Equal(t, 98.5, metrics.Metrics["active_shards_percent"].Value)
}
