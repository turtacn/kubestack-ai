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

package manager

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	intplugin "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// DiagnosticPluginAdapter adapts a plugin.DiagnosticPlugin to the interfaces.MiddlewarePlugin interface.
type DiagnosticPluginAdapter struct {
	p intplugin.DiagnosticPlugin
}

func (a *DiagnosticPluginAdapter) Name() string    { return a.p.Name() }
func (a *DiagnosticPluginAdapter) Version() string { return a.p.Version() }
func (a *DiagnosticPluginAdapter) Description() string {
	return fmt.Sprintf("Adapter for %s diagnostic plugin", a.p.Name())
}
func (a *DiagnosticPluginAdapter) SupportedVersions() []string {
	return a.p.SupportedTypes()
}
func (a *DiagnosticPluginAdapter) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return a.p.Diagnose(ctx, req)
}
func (a *DiagnosticPluginAdapter) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *DiagnosticPluginAdapter) CollectLogs(ctx context.Context, opts *models.LogOptions) (*models.LogData, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *DiagnosticPluginAdapter) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *DiagnosticPluginAdapter) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *DiagnosticPluginAdapter) Ping(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
func (a *DiagnosticPluginAdapter) CanAutoFix(issue *models.Issue) bool { return false }
func (a *DiagnosticPluginAdapter) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *DiagnosticPluginAdapter) ValidateFix(ctx context.Context, fix *models.FixAction) error {
	return fmt.Errorf("not implemented")
}
