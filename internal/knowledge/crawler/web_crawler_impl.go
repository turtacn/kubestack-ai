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
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
)

// advancedCrawler is a more sophisticated implementation of the WebCrawler interface.
// It supports deep crawling, filtering, concurrency control, and content processing.
type advancedCrawler struct {
	collector *colly.Collector
	visited   sync.Map // URL -> bool
	config    *config.CrawlerConfig
	logger    logger.Logger
	cleaner   ContentCleaner
	scorer    QualityScorer
}

// NewAdvancedCrawler creates a new instance of the advanced web crawler.
func NewAdvancedCrawler(cfg *config.CrawlerConfig) (WebCrawler, error) {
	c := colly.NewCollector(
		colly.Async(true),
	)

	timeout, err := time.ParseDuration(cfg.RequestTimeout)
	if err != nil {
		return nil, err
	}

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.MaxConcurrency,
		RandomDelay: 5 * time.Second,
	})

	c.UserAgent = cfg.UserAgent
	c.SetRequestTimeout(timeout)

	return &advancedCrawler{
		collector: c,
		config:    cfg,
		logger:    logger.NewLogger("advanced-crawler"),
		cleaner:   NewHTMLCleaner(),
		scorer:    NewDefaultQualityScorer(),
	}, nil
}

// Crawl is not implemented for advancedCrawler, as it's superseded by CrawlWithFilter.
func (c *advancedCrawler) Crawl(startURL string) (*store.RawDocument, error) {
	return nil, fmt.Errorf("not implemented, use CrawlWithFilter instead")
}

// CrawlWithFilter implements the deep crawling logic with filtering and concurrency.
func (c *advancedCrawler) CrawlWithFilter(ctx context.Context, startURL string) ([]*CrawledDocument, error) {
	var docs []*CrawledDocument
	var mu sync.Mutex

	// Find the target configuration for the startURL
	var targetConfig *config.Target
	for i := range c.config.Targets {
		if strings.HasPrefix(startURL, c.config.Targets[i].StartURL) {
			targetConfig = &c.config.Targets[i]
			break
		}
	}

	if targetConfig == nil {
		return nil, fmt.Errorf("no target configuration found for start URL: %s", startURL)
	}

	c.collector.AllowedDomains = targetConfig.AllowedDomains
	c.collector.MaxDepth = targetConfig.MaxDepth

	// Compile regex patterns
	var includeFilters []*regexp.Regexp
	for _, pattern := range targetConfig.URLPatterns.Include {
		includeFilters = append(includeFilters, regexp.MustCompile(pattern))
	}
	var excludeFilters []*regexp.Regexp
	for _, pattern := range targetConfig.URLPatterns.Exclude {
		excludeFilters = append(excludeFilters, regexp.MustCompile(pattern))
	}

	// OnHTML callback to find and visit links
	c.collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		linkParsed, err := url.Parse(link)
		if err != nil {
			return
		}

		// Manual filtering logic
		for _, pattern := range excludeFilters {
			if pattern.MatchString(linkParsed.Path) {
				return
			}
		}

		if len(includeFilters) > 0 {
			matched := false
			for _, pattern := range includeFilters {
				if pattern.MatchString(linkParsed.Path) {
					matched = true
					break
				}
			}
			if !matched {
				return
			}
		}

		if _, visited := c.visited.LoadOrStore(link, true); !visited {
			c.collector.Visit(link)
		}
	})

	// OnResponse callback to process the content
	c.collector.OnResponse(func(r *colly.Response) {
		if len(includeFilters) > 0 {
			matched := false
			for _, pattern := range includeFilters {
				if pattern.MatchString(r.Request.URL.Path) {
					matched = true
					break
				}
			}
			if !matched {
				return
			}
		}

		cleanedContent, err := c.cleaner.Clean(string(r.Body))
		if err != nil {
			c.logger.Errorf("Failed to clean content from %s: %v", r.Request.URL, err)
			return
		}

		// Extract title
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(r.Body)))
		if err != nil {
			c.logger.Errorf("Failed to parse content from %s: %v", r.Request.URL, err)
			return
		}
		title := doc.Find("title").Text()

		score := c.scorer.Score(cleanedContent)
		if float64(score) >= c.config.Quality.MinScore {
			doc := &CrawledDocument{
				URL:     r.Request.URL.String(),
				Content: cleanedContent,
				Title:   title,
			}
			mu.Lock()
			docs = append(docs, doc)
			mu.Unlock()
		}
	})

	// OnRequest to log and handle context cancellation
	c.collector.OnRequest(func(r *colly.Request) {
		select {
		case <-ctx.Done():
			r.Abort()
		default:
			c.logger.Debugf("Visiting %s", r.URL.String())
		}
	})

	// OnError callback to log errors
	c.collector.OnError(func(r *colly.Response, err error) {
		c.logger.Errorf("Request to %s failed with status %d: %v", r.Request.URL, r.StatusCode, err)
	})

	if err := c.collector.Visit(startURL); err != nil {
		return nil, err
	}
	c.collector.Wait()

	return docs, nil
}

// GetMetadata is a placeholder for now.
func (c *advancedCrawler) GetMetadata(doc *CrawledDocument) map[string]string {
	return nil
}
