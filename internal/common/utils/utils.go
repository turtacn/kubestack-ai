// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"math"
)

// CosineSimilarity calculates the cosine similarity between two vectors.
func CosineSimilarity(v1, v2 []float32) (float32, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vector lengths do not match")
	}

	var dotProduct float64
	var normV1 float64
	var normV2 float64

	for i := 0; i < len(v1); i++ {
		dotProduct += float64(v1[i] * v2[i])
		normV1 += float64(v1[i] * v1[i])
		normV2 += float64(v2[i] * v2[i])
	}

	if normV1 == 0 || normV2 == 0 {
		return 0, nil
	}

	return float32(dotProduct / (math.Sqrt(normV1) * math.Sqrt(normV2))), nil
}
