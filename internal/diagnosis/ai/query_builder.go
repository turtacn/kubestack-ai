package ai

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

type QueryBuilder struct {
	log logger.Logger
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		log: logger.NewLogger("query_builder"),
	}
}

// BuildSearchQuery generates a search query from logs, metrics, and middleware type.
// It accepts raw strings for logs and metrics as they come from DiagnosisRequest.
func (qb *QueryBuilder) BuildSearchQuery(logs string, metrics string, mwType string) string {
	var keywords []string

	// 1. Extract from Logs
	// Look for common error patterns
	logKeywords := qb.extractLogKeywords(logs)
	if len(logKeywords) > 0 {
		keywords = append(keywords, logKeywords...)
	}

	// 2. Extract from Metrics
	// This is a naive extraction assuming metrics string format "name=value" or similar.
	// Since the input is just a string, we look for common metric names that might indicate issues.
	metricKeywords := qb.extractMetricKeywords(metrics)
	if len(metricKeywords) > 0 {
		keywords = append(keywords, metricKeywords...)
	}

	// 3. Combine with Middleware Type
	query := fmt.Sprintf("%s", mwType)
	if len(keywords) > 0 {
		// Dedup keywords
		uniqueKeywords := make(map[string]bool)
		var deduped []string
		for _, k := range keywords {
			if !uniqueKeywords[k] {
				uniqueKeywords[k] = true
				deduped = append(deduped, k)
			}
		}
		// Limit to top 5 keywords to avoid too long query
		if len(deduped) > 5 {
			deduped = deduped[:5]
		}
		query += " " + strings.Join(deduped, " ")
	} else {
		// If no keywords found, maybe use a generic query
		query += " troubleshooting guide"
	}

	// Limit query length
	if len(query) > 200 {
		query = query[:200]
	}

	return query
}

func (qb *QueryBuilder) extractLogKeywords(logs string) []string {
	var keywords []string
	lines := strings.Split(logs, "\n")

	// Regex to find log levels and messages
	// Example: 2023-10-27 10:00:00 ERROR [component] message...
	// We want to capture the message part after ERROR/WARN
	errorRegex := regexp.MustCompile(`(?i)(ERROR|WARN|CRITICAL|FATAL)\s*[:\]]\s*(.*)`)

	// Regex to remove common noise (timestamps, IPs, hex IDs)
	noiseRegex := regexp.MustCompile(`\d{4}-\d{2}-\d{2}|\d{2}:\d{2}:\d{2}|0x[0-9a-fA-F]+|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	count := 0
	for _, line := range lines {
		if count >= 2 { // Limit to 2 distinct errors
			break
		}

		matches := errorRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			msg := matches[2]
			// Clean noise
			cleaned := noiseRegex.ReplaceAllString(msg, "")
			// Remove extra spaces
			cleaned = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " "))

			if cleaned != "" {
				keywords = append(keywords, cleaned)
				count++
			}
		} else if strings.Contains(strings.ToLower(line), "error") || strings.Contains(strings.ToLower(line), "fail") {
             // Fallback for lines without standard level indicators but containing error keywords
             // This is risky as it might capture noise, so we be conservative
             cleaned := noiseRegex.ReplaceAllString(line, "")
             cleaned = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " "))
             // Check if it's not too long or too short
             if len(cleaned) > 10 && len(cleaned) < 100 {
                 keywords = append(keywords, cleaned)
                 count++
             }
        }
	}
	return keywords
}

func (qb *QueryBuilder) extractMetricKeywords(metrics string) []string {
	var keywords []string
	// Assuming metrics might be in format "metric_name: value" or similar text description
	// We look for typical problem indicators in text if possible,
	// or just metric names if they are explicitly listed as "abnormal" or just present in the "Metrics" field passed to Analyze.

	// If the Metrics string is a JSON or structured dump, this regex might need adjustment.
	// For now, we assume it contains lines like "cpu_usage: 95%" or "replication_lag: 100s"

	lines := strings.Split(metrics, "\n")
	metricNameRegex := regexp.MustCompile(`([a-zA-Z0-9_]+)`)

	for _, line := range lines {
		// simplistic logic: if the line mentions "high", "low", "exceeded", "error", or just is present in this 'abnormal' context
		if strings.Contains(line, ">") || strings.Contains(strings.ToLower(line), "high") || strings.Contains(strings.ToLower(line), "lag") {
			matches := metricNameRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				keywords = append(keywords, matches[1])
			}
		}
	}
	return keywords
}
