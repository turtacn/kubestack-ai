package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

func init() {
	plugin.RegisterPluginFactory("Elasticsearch", func() plugin.DiagnosticPlugin {
		return &ElasticsearchPlugin{}
	})
}

type ElasticsearchPlugin struct {
	urls []string
}

func (p *ElasticsearchPlugin) Name() string {
	return "elasticsearch"
}

func (p *ElasticsearchPlugin) SupportedTypes() []string {
	return []string{"elasticsearch"}
}

func (p *ElasticsearchPlugin) Version() string {
	return "1.0.0"
}

func (p *ElasticsearchPlugin) Init(config map[string]interface{}) error {
	urlsInterface, ok := config["urls"].([]interface{})
	if !ok {
		return fmt.Errorf("config 'urls' is required")
	}
	for _, u := range urlsInterface {
		p.urls = append(p.urls, u.(string))
	}
	return nil
}

func (p *ElasticsearchPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	result := &models.DiagnosisResult{
		Issues: []*models.Issue{},
	}

	for _, url := range p.urls {
		issue := p.checkClusterHealth(ctx, url)
		if issue != nil {
			result.Issues = append(result.Issues, issue)
		}
	}

	return result, nil
}

func (p *ElasticsearchPlugin) checkClusterHealth(ctx context.Context, baseURL string) *models.Issue {
	resp, err := http.Get(baseURL + "/_cluster/health")
	if err != nil {
		return &models.Issue{
			Title:       "Elasticsearch Connection Failed",
			Severity:    enum.SeverityCritical,
			Description: fmt.Sprintf("Failed to connect to %s: %v", baseURL, err),
			Source:      "ElasticsearchPlugin",
		}
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil
	}

	status, _ := health["status"].(string)
	if status == "red" {
		return &models.Issue{
			Title:       "Elasticsearch Cluster Status RED",
			Severity:    enum.SeverityCritical,
			Description: "Cluster health is RED, indicating data loss or unavailability",
			Source:      "ElasticsearchPlugin",
		}
	} else if status == "yellow" {
		return &models.Issue{
			Title:       "Elasticsearch Cluster Status YELLOW",
			Severity:    enum.SeverityWarning,
			Description: "Cluster health is YELLOW, replicas might be missing",
			Source:      "ElasticsearchPlugin",
		}
	}

	return nil
}

func (p *ElasticsearchPlugin) Shutdown() error {
	return nil
}
