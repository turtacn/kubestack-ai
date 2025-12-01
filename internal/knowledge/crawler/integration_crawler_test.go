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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CrawlRedisOfficialDocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}

	longContent := strings.Repeat("redis word ", 201)

	// Create a mock HTTP server that mimics the Redis docs structure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageContent := fmt.Sprintf(`<html><head><title>%s</title></head><body>%s</body></html>`, r.URL.Path, longContent)
		if r.URL.Path == "/" {
			w.Write([]byte(`<html><head><title>Home</title></head><body>
				<a href="/commands/get">Commands Get</a>
				<a href="/download">Download</a>
			</body></html>`))
		} else {
			w.Write([]byte(pageContent))
		}
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Configure the crawler
	cfg := &config.CrawlerConfig{
		MaxConcurrency: 5,
		RequestTimeout: "30s",
		UserAgent:      "test-crawler",
		Quality: config.QualityConfig{
			MinScore: 0,
		},
		Targets: []config.Target{
			{
				StartURL: server.URL,
				AllowedDomains: []string{
					u.Hostname(),
				},
				URLPatterns: config.URLPatterns{
					Include: []string{
						`/commands/`,
					},
					Exclude: []string{
						`/download`,
					},
				},
				MaxDepth: 2,
			},
		},
	}

	crawler, err := NewAdvancedCrawler(cfg)
	require.NoError(t, err)

	// Start the crawl
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results, err := crawler.CrawlWithFilter(ctx, server.URL)
	require.NoError(t, err)

	// Assert the results
	assert.Len(t, results, 1)

	// Calculate average quality score
	scorer := NewDefaultQualityScorer()
	totalScore := 0
	for _, doc := range results {
		totalScore += scorer.Score(doc.Content)
	}
	averageScore := float64(totalScore) / float64(len(results))

	assert.GreaterOrEqual(t, int(averageScore), 10)
}
