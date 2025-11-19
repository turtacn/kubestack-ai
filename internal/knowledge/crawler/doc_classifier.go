// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package crawler

import "strings"

// DocClassifier defines the interface for classifying the type of a document.
type DocClassifier interface {
	Classify(content string) string
}

// DefaultDocClassifier is a default implementation of DocClassifier.
type DefaultDocClassifier struct{}

// NewDefaultDocClassifier creates a new DefaultDocClassifier.
func NewDefaultDocClassifier() *DefaultDocClassifier {
	return &DefaultDocClassifier{}
}

// Classify determines the type of the document based on its content.
func (c *DefaultDocClassifier) Classify(content string) string {
	lowerContent := strings.ToLower(content)

	if strings.Contains(lowerContent, "tutorial") || strings.Contains(lowerContent, "how-to") {
		return "tutorial"
	}
	if strings.Contains(lowerContent, "reference") || strings.Contains(lowerContent, "api") {
		return "reference"
	}
	if strings.Contains(lowerContent, "troubleshooting") || strings.Contains(lowerContent, "error") {
		return "troubleshooting"
	}
	if strings.Contains(lowerContent, "concept") || strings.Contains(lowerContent, "architecture") {
		return "concept"
	}

	return "general"
}
