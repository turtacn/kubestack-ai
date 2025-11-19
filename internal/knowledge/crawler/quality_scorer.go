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

import (
	"regexp"
	"strings"
)

// QualityScorer defines the interface for scoring the quality of crawled content.
type QualityScorer interface {
	Score(content string) int
}

// DefaultQualityScorer is a default implementation of QualityScorer.
type DefaultQualityScorer struct {
	keywordDict map[string]int
}

// NewDefaultQualityScorer creates a new DefaultQualityScorer.
func NewDefaultQualityScorer() *DefaultQualityScorer {
	return &DefaultQualityScorer{
		keywordDict: map[string]int{
			"redis":    1,
			"kafka":    1,
			"mysql":    1,
			"cluster":  1,
			"command":  1,
			"topic":    1,
			"database": 1,
		},
	}
}

// Score assesses the quality of the content based on several heuristics.
func (s *DefaultQualityScorer) Score(content string) int {
	score := 10

	// 1. Word count score (up to 50 points)
	wordCount := len(strings.Fields(content))
	if wordCount > 10 {
		score += 10
	}

	// 2. Code blocks score (up to 30 points)
	codeBlockCount := len(regexp.MustCompile("```|<pre>").FindAllString(content, -1))
	score += codeBlockCount * 10

	// 3. Keyword density score (up to 30 points)
	for keyword, weight := range s.keywordDict {
		if strings.Contains(strings.ToLower(content), keyword) {
			score += weight
		}
	}

	return score
}
