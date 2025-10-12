// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// MongoPlugin is the plugin for diagnosing MongoDB instances.
type MongoPlugin struct {
	client *mongo.Client
}

// New creates a new instance of the MongoDB plugin.
func New() (interfaces.MiddlewarePlugin, error) {
	return &MongoPlugin{}, nil
}

func (p *MongoPlugin) Name() string {
	return "mongodb"
}

func (p *MongoPlugin) Version() string {
	return "0.1.0"
}

func (p *MongoPlugin) Description() string {
	return "Provides diagnostics for MongoDB instances."
}

func (p *MongoPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	// In a real implementation, connection details would come from config.
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}
	p.client = client
	return nil
}

func (p *MongoPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	var serverStatus bson.M
	err := p.client.Database("admin").RunCommand(ctx, bson.D{{"serverStatus", 1}}).Decode(&serverStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get mongodb server status: %w", err)
	}

	metrics := make(map[string]interface{})
	if connections, ok := serverStatus["connections"].(bson.M); ok {
		if current, ok := connections["current"].(int32); ok {
			metrics["connections_current"] = float64(current)
		}
	}
	if opcounters, ok := serverStatus["opcounters"].(bson.M); ok {
		if query, ok := opcounters["query"].(int64); ok {
			metrics["opcounters_query"] = float64(query)
		}
	}
	// Add more metrics as needed

	return &models.MetricsData{Data: metrics}, nil
}

func (p *MongoPlugin) CollectLogs(ctx context.Context, options *models.LogOptions) (*models.LogData, error) {
	// Log collection from MongoDB is complex and depends on the logging setup.
	// This is a placeholder for a future implementation.
	return &models.LogData{Entries: []string{}}, nil
}

func (p *MongoPlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	// Placeholder: In a real scenario, you might get this from a config file
	// or specific admin commands, but for now, we'll return an empty map.
	return &models.ConfigData{Data: make(map[string]string)}, nil
}

func (p *MongoPlugin) Analyze(ctx context.Context, data *models.CollectedData) ([]*models.Issue, error) {
	analyzer := diagnosis.NewRuleBasedAnalyzer(p.getMetricRules(), p.getLogRules())
	var issues []*models.Issue

	if data.Metrics != nil {
		metricIssues, err := analyzer.AnalyzeMetrics(ctx, data.Metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze mongodb metrics: %w", err)
		}
		issues = append(issues, metricIssues...)
	}

	if data.Logs != nil {
		logIssues, err := analyzer.AnalyzeLogs(ctx, data.Logs)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze mongodb logs: %w", err)
		}
		issues = append(issues, logIssues...)
	}

	return issues, nil
}

func (p *MongoPlugin) getMetricRules() []diagnosis.MetricRule {
	return []diagnosis.MetricRule{
		{
			MetricName:     "connections_current",
			Operator:       ">",
			Threshold:      1000,
			Severity:       enum.SeverityWarning,
			IssueTitle:     "High Number of Connections",
			Recommendation: "The number of current connections is high. This may indicate connection leaks in applications or a need to increase the 'maxPoolSize' in the connection string.",
		},
	}
}

func (p *MongoPlugin) getLogRules() []diagnosis.LogRule {
	return []diagnosis.LogRule{
		{
			Pattern:        "Slow query",
			Severity:       enum.SeverityWarning,
			IssueTitle:     "Slow Query Logged",
			Recommendation: "A slow query was logged. Use the database profiler to investigate the query and consider adding indexes to the relevant collections.",
		},
	}
}

func (p *MongoPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	data, err := p.CollectAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect data for diagnosis: %w", err)
	}
	issues, err := p.Analyze(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze data for diagnosis: %w", err)
	}
	return &models.DiagnosisResult{
		Issues: issues,
	}, nil
}

func (p *MongoPlugin) CollectAll(ctx context.Context) (*models.CollectedData, error) {
	metrics, err := p.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logs, err := p.CollectLogs(ctx, &models.LogOptions{})
	if err != nil {
		return nil, err
	}
	config, err := p.GetConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	return &models.CollectedData{
		Metrics: metrics,
		Logs:    logs,
		Config:  config,
	}, nil
}

func (p *MongoPlugin) GetHealth(ctx context.Context) (*models.HealthStatus, error) {
	if err := p.client.Ping(ctx, nil); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: err.Error()}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "Successfully connected to MongoDB."}, nil
}

func (p *MongoPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	return p.GetHealth(ctx)
}

func (p *MongoPlugin) CanAutoFix(issue *models.Issue) bool {
	return false
}

func (p *MongoPlugin) ExecuteFix(ctx context.Context, action *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("auto-fix is not yet implemented for the mongodb plugin")
}

func (p *MongoPlugin) ValidateFix(ctx context.Context, action *models.FixAction) error {
	return fmt.Errorf("auto-fix validation is not yet implemented for the mongodb plugin")
}

func (p *MongoPlugin) Ping(ctx context.Context) error {
	return p.client.Ping(ctx, nil)
}

func (p *MongoPlugin) SupportedVersions() []string {
	return []string{"4.x", "5.x", "6.x"}
}