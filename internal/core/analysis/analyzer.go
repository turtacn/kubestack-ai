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

package analysis

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Analyzer defines the contract for the secondary analysis layer.
// It processes plugin output data and produces structured analysis results.
// This abstraction allows for multiple analyzer implementations (rule-based, AI-enhanced, RAG, etc.)
// to coexist and be easily swapped or combined.
type Analyzer interface {
	// Name returns the unique identifier for this analyzer
	Name() string

	// Analyze processes collected plugin data and returns structured analysis results.
	// This is the main entry point for all analysis operations.
	Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error)
}

// AnalysisResult represents the structured output from an analyzer.
// It contains identified issues with evidence, severity, and actionable suggestions.
type AnalysisResult struct {
	// AnalyzerName identifies which analyzer produced this result
	AnalyzerName string

	// Issues contains all problems identified during analysis
	Issues []*models.Issue

	// Summary provides a high-level overview of the analysis
	Summary string

	// Metadata contains additional context about the analysis process
	Metadata map[string]interface{}
}

// NewAnalysisResult creates a new AnalysisResult with the given analyzer name
func NewAnalysisResult(analyzerName string) *AnalysisResult {
	return &AnalysisResult{
		AnalyzerName: analyzerName,
		Issues:       make([]*models.Issue, 0),
		Metadata:     make(map[string]interface{}),
	}
}
