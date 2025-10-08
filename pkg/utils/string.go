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

package utils

import (
	"strings"
	"unicode/utf8"
)

// Truncate cuts a string to a maximum length and appends an ellipsis ("…") if it was cut.
// This function is safe for multi-byte UTF-8 characters.
//
// Parameters:
//   s (string): The input string to truncate.
//   maxLength (int): The maximum number of characters to allow before truncating.
//
// Returns:
//   string: The truncated string, with an ellipsis if it was shortened.
func Truncate(s string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLength {
		return s
	}
	// Convert to a rune slice to handle multi-byte characters correctly.
	runes := []rune(s)
	// Subtract 1 from maxLength to make space for the ellipsis.
	return string(runes[:maxLength-1]) + "…"
}

// IsBlank checks if a string is empty ("") or consists only of whitespace characters.
//
// Parameters:
//   s (string): The string to check.
//
// Returns:
//   bool: True if the string is empty or contains only whitespace, false otherwise.
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// Levenshtein calculates the Levenshtein distance between two strings. This distance is
// the minimum number of single-character edits (insertions, deletions, or substitutions)
// required to change one word into the other. It's a common way to measure string similarity.
//
// Parameters:
//   a (string): The first string for comparison.
//   b (string): The second string for comparison.
//
// Returns:
//   int: The Levenshtein distance (edit distance) between the two strings.
func Levenshtein(a, b string) int {
	s1 := []rune(a)
	s2 := []rune(b)

	lenS1 := len(s1)
	lenS2 := len(s2)

	if lenS1 == 0 {
		return lenS2
	}
	if lenS2 == 0 {
		return lenS1
	}

	// Initialize the distance matrix.
	matrix := make([][]int, lenS1+1)
	for i := range matrix {
		matrix[i] = make([]int, lenS2+1)
	}

	for i := 0; i <= lenS1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= lenS2; j++ {
		matrix[0][j] = j
	}

	// Calculate distances.
	for i := 1; i <= lenS1; i++ {
		for j := 1; j <= lenS2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			deletionCost := matrix[i-1][j] + 1
			insertionCost := matrix[i][j-1] + 1
			substitutionCost := matrix[i-1][j-1] + cost

			matrix[i][j] = min(deletionCost, insertionCost, substitutionCost)
		}
	}

	return matrix[lenS1][lenS2]
}

// min is a helper function for Levenshtein to find the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
	} else {
		if b < c {
			return b
		}
	}
	return c
}

//Personal.AI order the ending
