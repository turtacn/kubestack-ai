// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgresql

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	coremodels "github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

type postgresPlugin struct {
	config *config.PluginConfig
}

func New() (interfaces.DiagnosticPlugin, error) {
	return &postgresPlugin{}, nil
}

func (p *postgresPlugin) Name() string {
	return "postgresql"
}

func (p *postgresPlugin) Version() string {
	return "1.0.0"
}

func (p *postgresPlugin) Description() string {
	return "PostgreSQL diagnostic plugin"
}

func (p *postgresPlugin) SupportedTypes() []enum.MiddlewareType {
	return []enum.MiddlewareType{enum.PostgreSQL}
}

func (p *postgresPlugin) SupportedVersions() []string {
	return []string{"12", "13", "14", "15"}
}

func (p *postgresPlugin) Init(cfg *config.PluginConfig) error {
	p.config = cfg
	return nil
}

func (p *postgresPlugin) Shutdown() error {
	return nil
}

func (p *postgresPlugin) Diagnose(ctx context.Context, target string) (*coremodels.ComponentDiagnosisResult, error) {
	return &coremodels.ComponentDiagnosisResult{
		Component: "postgresql",
		Status:    "Healthy",
	}, nil
}

func (p *postgresPlugin) CollectMetrics(ctx context.Context, target string) (*coremodels.MetricsData, error) {
	return &coremodels.MetricsData{Data: map[string]interface{}{}}, nil
}

func (p *postgresPlugin) CollectLogs(ctx context.Context, target string, opts *coremodels.LogOptions) (*coremodels.LogData, error) {
	return &coremodels.LogData{Entries: []string{}}, nil
}

func (p *postgresPlugin) CollectConfig(ctx context.Context, target string) (*coremodels.ConfigData, error) {
	return &coremodels.ConfigData{Data: map[string]string{}}, nil
}

func (p *postgresPlugin) HealthCheck(ctx context.Context, target string) (*coremodels.HealthStatus, error) {
	return &coremodels.HealthStatus{IsHealthy: true}, nil
}

func (p *postgresPlugin) Ping(ctx context.Context, target string) error {
	return nil
}

// Remediation stubs
func (p *postgresPlugin) CanAutoFix(issue *coremodels.Issue) bool { return false }
func (p *postgresPlugin) ExecuteFix(ctx context.Context, fix *coremodels.FixAction) (*coremodels.FixResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (p *postgresPlugin) ValidateFix(ctx context.Context, fix *coremodels.FixAction) error {
	return fmt.Errorf("not implemented")
}
