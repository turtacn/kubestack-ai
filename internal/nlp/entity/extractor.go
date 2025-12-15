package entity

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Extractor is the interface for entity extraction.
type Extractor interface {
	Extract(ctx context.Context, text string, tokens []string) ([]Entity, error)
}

// PatternBasedExtractor is an implementation of Extractor using regex and dictionaries.
type PatternBasedExtractor struct {
	patterns         map[EntityType][]*EntityPattern
	dictionaries     map[EntityType]map[string]string // normalized value map
	dictionaryRegexs map[EntityType]*regexp.Regexp    // Compiled regex for dictionary keys
}

// BuildDefaultExtractor builds a default PatternBasedExtractor.
func BuildDefaultExtractor() *PatternBasedExtractor {
	e := &PatternBasedExtractor{
		patterns: map[EntityType][]*EntityPattern{
			EntityTimeRange:  timeRangePatterns,
			EntityThreshold:  thresholdPatterns,
			EntityInstanceID: instanceIDPatterns,
		},
		dictionaries: map[EntityType]map[string]string{
			EntityMiddlewareType: middlewareTypeDict,
			EntityMetricName:     metricNameDict,
		},
		dictionaryRegexs: make(map[EntityType]*regexp.Regexp),
	}

	// Compile dictionary regexes
	e.compileDictionaryRegexs()

	return e
}

func (e *PatternBasedExtractor) compileDictionaryRegexs() {
	for entityType, dict := range e.dictionaries {
		if len(dict) == 0 {
			continue
		}

		// Sort keys by length descending to match longest first in regex alternation
		keys := make([]string, 0, len(dict))
		for k := range dict {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return len(keys[i]) > len(keys[j])
		})

		// Escape keys
		escapedKeys := make([]string, len(keys))
		for i, k := range keys {
			escapedKeys[i] = regexp.QuoteMeta(k)
		}

		// Construct regex: (?i)(key1|key2|...)
		pattern := fmt.Sprintf("(?i)(%s)", strings.Join(escapedKeys, "|"))
		re, err := regexp.Compile(pattern)
		if err == nil {
			e.dictionaryRegexs[entityType] = re
		}
	}
}

// Extract extracts entities from text using patterns and dictionaries.
func (e *PatternBasedExtractor) Extract(ctx context.Context, text string, tokens []string) ([]Entity, error) {
	var entities []Entity

	// 1. Regex Pattern Matching
	for entityType, patterns := range e.patterns {
		for _, pattern := range patterns {
			matches := pattern.Regex.FindAllStringIndex(text, -1)
			for _, loc := range matches {
				val := text[loc[0]:loc[1]]
				normVal := val
				if pattern.Normalizer != nil {
					normVal = pattern.Normalizer(val)
				}

				entities = append(entities, Entity{
					Type:       entityType,
					Value:      val,
					NormValue:  normVal,
					StartPos:   loc[0],
					EndPos:     loc[1],
					Confidence: 0.9,
				})
			}
		}
	}

	// 2. Dictionary Matching via Regex
	for entityType, re := range e.dictionaryRegexs {
		matches := re.FindAllStringIndex(text, -1)
		dict := e.dictionaries[entityType]
		for _, loc := range matches {
			val := text[loc[0]:loc[1]]
			// Look up normalized value.
			// We need to match the key in dictionary.
			// Since we used case-insensitive regex, 'val' might have different casing than dict key.
			// We iterate dict to find the matching key (case-insensitive).
			// Optimization: keep a lower-case map.

			// For now, iterate (dict is small).
			var normVal string
			found := false
			for k, v := range dict {
				if strings.EqualFold(k, val) {
					normVal = v
					found = true
					break
				}
			}

			if found {
				entities = append(entities, Entity{
					Type:       entityType,
					Value:      val,
					NormValue:  normVal,
					StartPos:   loc[0],
					EndPos:     loc[1],
					Confidence: 0.85,
				})
			}
		}
	}

	// 3. Deduplicate and resolve overlaps
	return e.deduplicateEntities(entities), nil
}

func (e *PatternBasedExtractor) deduplicateEntities(entities []Entity) []Entity {
	if len(entities) == 0 {
		return entities
	}

	// Sort by StartPos, then by Length (descending) to prefer longer matches
	sort.Slice(entities, func(i, j int) bool {
		if entities[i].StartPos == entities[j].StartPos {
			return (entities[i].EndPos - entities[i].StartPos) > (entities[j].EndPos - entities[j].StartPos)
		}
		return entities[i].StartPos < entities[j].StartPos
	})

	var result []Entity
	if len(entities) > 0 {
		result = append(result, entities[0])
	}

	for i := 1; i < len(entities); i++ {
		prev := result[len(result)-1]
		curr := entities[i]

		// Check overlap
		if curr.StartPos < prev.EndPos {
			continue
		}
		result = append(result, curr)
	}

	return result
}
