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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultQualityScorer_Score(t *testing.T) {
	scorer := NewDefaultQualityScorer()

	testCases := []struct {
		name        string
		content     string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "short text",
			content:     "This is a short text.",
			expectedMin: 10,
			expectedMax: 30,
		},
		{
			name:        "good document",
			content:     "This is a good document about redis with a ```code block```. " + strings.Repeat("word ", 2000),
			expectedMin: 30,
			expectedMax: 50,
		},
		{
			name:        "long text",
			content:     strings.Repeat("word ", 10000),
			expectedMin: 10,
			expectedMax: 30,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scorer.Score(tc.content)
			assert.GreaterOrEqual(t, score, tc.expectedMin)
			assert.LessOrEqual(t, score, tc.expectedMax)
		})
	}
}
