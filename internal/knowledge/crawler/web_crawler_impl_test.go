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

func TestAdvancedCrawler_CrawlWithFilter(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		longContent := strings.Repeat("word ", 201)
		if r.URL.Path == "/" {
			w.Write([]byte(fmt.Sprintf(`<html><head><title>Home</title></head><body>%s<a href="/page1">Page 1</a><a href="/ads/ad1">Ad</a></body></html>`, longContent)))
		} else if r.URL.Path == "/page1" {
			w.Write([]byte(fmt.Sprintf(`<html><head><title>Page 1</title></head><body>%s</body></html>`, longContent)))
		} else if r.URL.Path == "/ads/ad1" {
			w.Write([]byte(`<html><head><title>Ad</title></head><body>Ad content</body></html>`))
		}
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Configure the crawler
	cfg := &config.CrawlerConfig{
		MaxConcurrency: 2,
		RequestTimeout: "10s",
		UserAgent:      "test-crawler",
		Quality: config.Quality{
			MinScore: 0,
		},
		Targets: []config.Target{
			{
				Name:     "test-site",
				StartURL: server.URL,
				AllowedDomains: []string{
					u.Hostname(),
				},
				URLPatterns: config.URLPatterns{
					Exclude: []string{
						`/ads/`,
					},
				},
				MaxDepth: 2,
			},
		},
	}

	crawler, err := NewAdvancedCrawler(cfg)
	require.NoError(t, err)

	// Start the crawl
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := crawler.CrawlWithFilter(ctx, server.URL)
	require.NoError(t, err)

	// Assert the results
	assert.Len(t, results, 2)
	// Sort results by URL to have a deterministic order for assertion
	if results[0].URL > results[1].URL {
		results[0], results[1] = results[1], results[0]
	}
	assert.Equal(t, "Home", results[0].Title)
	assert.Equal(t, "Page 1", results[1].Title)
}
