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

package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRRFFusion(t *testing.T) {
	fusion := NewRRFFusion(60)

	semanticResults := []*Document{
		{Content: "doc1", Score: 0.9},
		{Content: "doc3", Score: 0.8},
		{Content: "doc2", Score: 0.7},
	}

	bm25Results := []*Document{
		{Content: "doc2", Score: 0.6},
		{Content: "doc1", Score: 0.5},
		{Content: "doc4", Score: 0.4},
	}

	fused := fusion.Fuse(semanticResults, bm25Results)

	assert.Len(t, fused, 4)
	assert.Equal(t, "doc1", fused[0].Content)
	assert.Equal(t, "doc2", fused[1].Content)
	assert.Equal(t, "doc3", fused[2].Content)
	assert.Equal(t, "doc4", fused[3].Content)
}

func TestWeightedFusion(t *testing.T) {
	fusion := NewWeightedFusion([]float64{0.7, 0.3})

	semanticResults := []*Document{
		{Content: "doc1", Score: 1.0},
		{Content: "doc2", Score: 0.5},
	}

	bm25Results := []*Document{
		{Content: "doc2", Score: 1.0},
		{Content: "doc1", Score: 0.5},
	}

	fused := fusion.Fuse(semanticResults, bm25Results)

	assert.Len(t, fused, 2)
	assert.Equal(t, "doc1", fused[0].Content) // 0.7*1 + 0.3*0.5 = 0.85
	assert.Equal(t, "doc2", fused[1].Content) // 0.7*0.5 + 0.3*1 = 0.65
}
