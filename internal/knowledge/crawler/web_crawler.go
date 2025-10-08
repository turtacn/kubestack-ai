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

// Package crawler provides components for acquiring knowledge from various sources.
package crawler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
)

// WebCrawler defines the interface for a component that fetches and extracts
// content from web pages.
type WebCrawler interface {
	// Crawl visits a given URL, extracts its primary text content, and returns
	// it as a RawDocument.
	//
	// Parameters:
	//   startURL (string): The URL to start crawling from.
	//
	// Returns:
	//   *store.RawDocument: A document containing the extracted text content.
	//   error: An error if the crawl fails.
	Crawl(startURL string) (*store.RawDocument, error)
}

// simpleWebCrawler is a basic implementation of a web crawler using the Colly library.
type simpleWebCrawler struct {
	log       logger.Logger
	collector *colly.Collector
}

// NewWebCrawler creates and configures a new web crawler using the Colly library.
// It sets up important configurations like allowed domains for safety and polite
// rate limiting to avoid overwhelming servers.
//
// Parameters:
//   allowedDomains ([]string): A slice of domains that the crawler is permitted to visit.
//
// Returns:
//   WebCrawler: A new instance of a web crawler.
//   error: An error if initialization fails (nil in this implementation).
func NewWebCrawler(allowedDomains []string) (WebCrawler, error) {
	c := colly.NewCollector(
		// Restrict crawling to specified domains to prevent going off-topic.
		colly.AllowedDomains(allowedDomains...),
		// Use asynchronous requests to be faster, but still respect politeness settings.
		colly.Async(true),
	)

	// Set politeness settings to avoid overwhelming servers.
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2, // Only 2 concurrent requests to any single domain.
		RandomDelay: 2 * time.Second,
	})

	// Set a common user agent to identify the crawler as a good bot.
	c.UserAgent = "KubeStack-AI-Knowledge-Crawler/1.0 (+https://github.com/kubestack-ai/kubestack-ai)"

	return &simpleWebCrawler{
		log:       logger.NewLogger("web-crawler"),
		collector: c,
	}, nil
}

// Crawl implements the WebCrawler interface. It starts the crawling process from
// a given URL and extracts the text content from the body of the page.
//
// NOTE: This is a simple implementation that only crawls the single page provided.
// A more advanced version would follow links to perform a deep crawl.
//
// Parameters:
//   startURL (string): The URL of the page to crawl.
//
// Returns:
//   *store.RawDocument: A document containing the extracted text content.
//   error: An error if the HTTP request or crawling process fails.
func (c *simpleWebCrawler) Crawl(startURL string) (*store.RawDocument, error) {
	c.log.Infof("Starting crawl for URL: %s", startURL)

	var contentBuilder strings.Builder
	var crawlErr error

	// Use a channel to reliably get the result from the async callback.
	resultChan := make(chan bool, 1)

	// OnHTML is the main callback for parsing content.
	// We extract all text from the body, which is a simple way to remove noise (scripts, styles, navs).
	// A more advanced implementation would use more specific selectors.
	c.collector.OnHTML("body", func(e *colly.HTMLElement) {
		contentBuilder.WriteString(e.Text)
	})

	// OnRequest is called before making a request.
	c.collector.OnRequest(func(r *colly.Request) {
		c.log.Debugf("Visiting %s", r.URL)
	})

	// OnError handles any errors that occur during the request.
	c.collector.OnError(func(r *colly.Response, err error) {
		crawlErr = fmt.Errorf("request to %s failed with status %d: %w", r.Request.URL, r.StatusCode, err)
	})

	// OnScraped is called after all OnHTML callbacks have been executed.
	c.collector.OnScraped(func(r *colly.Response) {
		resultChan <- true
	})

	// Start the crawl.
	if err := c.collector.Visit(startURL); err != nil {
		return nil, fmt.Errorf("failed to start visit for %s: %w", startURL, err)
	}

	// Wait for the crawl to finish.
	c.collector.Wait()
	close(resultChan)

	// Check if the crawl was successful.
	if _, ok := <-resultChan; !ok && crawlErr != nil {
		return nil, crawlErr
	}
	if crawlErr != nil {
		return nil, crawlErr
	}

	// TODO: Implement deep crawling by finding links in `OnHTML("a[href]", ...)` and calling `r.Request.Visit(...)`.
	// TODO: Implement content quality assessment and filtering (e.g., discard pages with too little text).
	// TODO: Handle other content types like PDF and Markdown by checking the Content-Type header.

	return &store.RawDocument{
		ID:      uuid.New().String(),
		Content: contentBuilder.String(),
		Source:  startURL,
	}, nil
}

//Personal.AI order the ending
